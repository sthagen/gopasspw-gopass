package termio

import (
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"golang.org/x/term"
)

const (
	fps = 25
)

var now = time.Now

// ProgressBar is a gopass progress bar.
type ProgressBar struct {
	// keep both int64 fields at the top to ensure correct
	// 8-byte alignment on 32 bit systems. See https://golang.org/pkg/sync/atomic/#pkg-note-BUG
	// and https://github.com/golang/go/issues/36606
	total   int64
	current int64

	mutex   chan struct{}
	lastUpd time.Time

	Hidden bool
	Bytes  bool
}

// NewProgressBar creates a new progress bar.
func NewProgressBar(total int64) *ProgressBar {
	return &ProgressBar{
		total:   total,
		current: 0,
		mutex:   make(chan struct{}, 1),
	}
}

// Add adds the given amount to the progress.
func (p *ProgressBar) Add(v int64) {
	if p == nil {
		return
	}

	cur := atomic.AddInt64(&p.current, v)
	if maxVal := atomic.LoadInt64(&p.total); cur > maxVal {
		atomic.StoreInt64(&p.total, cur)
	}

	p.print()
}

// Inc adds one to the progress.
func (p *ProgressBar) Inc() {
	if p == nil {
		return
	}

	cur := atomic.AddInt64(&p.current, 1)
	if maxVal := atomic.LoadInt64(&p.total); cur > maxVal {
		atomic.StoreInt64(&p.total, cur)
	}

	p.print()
}

// Set sets an arbitrary progress.
func (p *ProgressBar) Set(v int64) {
	if p == nil {
		return
	}

	atomic.StoreInt64(&p.current, v)

	if maxVal := atomic.LoadInt64(&p.total); v > maxVal {
		atomic.StoreInt64(&p.total, v)
	}

	p.print()
}

// Done finalizes the progress bar.
func (p *ProgressBar) Done() {
	if p == nil {
		return
	}

	if p.Hidden {
		return
	}

	fmt.Fprintln(Stderr, "")
}

// Clear removes the progress bar.
func (p *ProgressBar) Clear() {
	if p == nil {
		return
	}

	clearLine()
}

// print will print the progress bar, if necessary.
func (p *ProgressBar) print() {
	if p == nil {
		return
	}

	if p.Hidden {
		return
	}

	// try to lock
	select {
	case p.mutex <- struct{}{}:
		// lock acquired
		p.tryPrint()
		<-p.mutex
	default:
		// lock not acquired
		return
	}
}

func (p *ProgressBar) tryPrint() {
	ts := now()
	if p.current == 0 || p.current >= p.total-1 || ts.Sub(p.lastUpd) > time.Second/fps {
		p.lastUpd = ts
		p.doPrint()
	}
}

// doPrint redraws the current line.
// This method is based on https://github.com/muesli/goprogressbar/blob/master/progressbar.go#L96
func (p *ProgressBar) doPrint() {
	clearLine()

	cur, maxVal, pct := p.percent()
	pctStr := fmt.Sprintf("%.2f%%", pct*100)
	// ensure consistent length
	for len(pctStr) < 7 {
		pctStr = " " + pctStr
	}

	termWidth, _, _ := term.GetSize(int(syscall.Stdin)) //nolint:unconvert
	if termWidth < 0 {
		// if we can determine the size (e.g. windows, fake term, mock)
		// assume a sane default of 80
		termWidth = 80
	}

	barWidth := uint(termWidth)
	digits := int(math.Log10(float64(maxVal))) + 1
	// Log10(0) is undefined
	if maxVal < 1 {
		digits = 1
	}

	text := fmt.Sprintf(fmt.Sprintf(" %%%dd / %%%dd ", digits, digits), cur, maxVal)

	if p.Bytes {
		curStr := humanize.Bytes(uint64(cur))
		maxStr := humanize.Bytes(uint64(maxVal))
		digits := len(maxStr) + 1
		text = fmt.Sprintf(fmt.Sprintf(" %%%ds / %%%ds ", digits, digits), curStr, maxStr)
	}

	size := int(barWidth) - len(text) - len(pctStr) - 5
	fill := int(math.Max(2, math.Floor((float64(size)*pct)+.5)))

	fmt.Fprint(Stderr, text)

	// not enough space
	if size < 11 {
		return
	}

	// Rgggggggggggmcyy
	// Gooooooooooopass
	tg := color.RedString("G")
	to := strings.Repeat(color.GreenString("o"), gteZero(fill-5))
	tp := strings.Repeat(color.YellowString("p"), boundedMin(1, fill-4))
	ta := strings.Repeat(color.MagentaString("a"), boundedMin(1, fill-3))
	ts := strings.Repeat(color.CyanString("s"), boundedMin(2, fill-1))
	spc := strings.Repeat(" ", gteZero(size-fill))
	fmt.Fprintf(Stderr, "[%s%s%s%s%s%s] %s ",
		tg,
		to,
		tp,
		ta,
		ts,
		spc,
		pctStr,
	)
}

func gteZero(a int) int {
	if a >= 0 {
		return a
	}

	return 0
}

func boundedMin(a, b int) int {
	return gteZero(min(a, b))
}

func (p *ProgressBar) percent() (int64, int64, float64) {
	cur := atomic.LoadInt64(&p.current)
	maxVal := atomic.LoadInt64(&p.total)
	pct := float64(cur) / float64(maxVal)

	if p.total < 1 {
		if p.current < 1 {
			pct = 1
		} else {
			pct = 0
		}
	}

	// normalized between 0.0 and 1.0
	return cur, maxVal, math.Min(1, math.Max(0, pct))
}

func clearLine() {
	fmt.Fprintf(Stderr, "\033[2K\r]")
}
