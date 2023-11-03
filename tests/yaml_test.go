package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLAndSecret(t *testing.T) {
	ts := newTester(t)
	defer ts.teardown()

	t.Run("show key from uninitialized store", func(t *testing.T) {
		_, err := ts.run("show foo/bar baz")
		require.Error(t, err)
	})

	ts.initStore()

	t.Run("default action (show) from initialized store", func(t *testing.T) {
		out, err := ts.run("foo/bar baz")
		require.Error(t, err)
		assert.Contains(t, out, "entry is not in the password store")
	})

	t.Run("insert key", func(t *testing.T) {
		_, err := ts.runCmd([]string{ts.Binary, "insert", "foo/bar", "password"}, []byte("moar"))
		require.NoError(t, err)
	})

	t.Run("insert another key", func(t *testing.T) {
		_, err := ts.runCmd([]string{ts.Binary, "insert", "foo/bar", "baz"}, []byte("moar"))
		require.NoError(t, err)
	})

	t.Run("insert into the body", func(t *testing.T) {
		out, err := ts.runCmd([]string{ts.Binary, "insert", "-a", "foo/bar"}, []byte("body"))
		require.NoError(t, err, out)
	})

	t.Run("show a key", func(t *testing.T) {
		out, err := ts.run("show foo/bar baz")
		require.NoError(t, err)
		assert.Equal(t, "moar", out)
	})

	t.Run("show the whole secret", func(t *testing.T) {
		out, err := ts.run("show foo/bar")
		require.NoError(t, err)
		assert.Equal(t, "password: moar\nbaz: moar\nbody", out)
	})
}

func TestInvalidYAML(t *testing.T) {
	testBody := `somepasswd
---
Test / test.com
username: myuser@test.com
password: someotherpasswd
url: http://www.test.com/`

	ts := newTester(t)
	defer ts.teardown()

	t.Run("show secret from uninitialized store", func(t *testing.T) {
		_, err := ts.run("show foo/bar")
		require.Error(t, err)
	})

	ts.initStore()

	t.Run("show non-existing secret", func(t *testing.T) {
		out, err := ts.run("foo/bar")
		require.Error(t, err)
		assert.Contains(t, out, "entry is not in the password store")
	})

	t.Run("insert new secret", func(t *testing.T) {
		_, err := ts.runCmd([]string{ts.Binary, "insert", "foo/bar"}, []byte(testBody))
		require.NoError(t, err)
	})

	t.Run("show newly inserted secret", func(t *testing.T) {
		_, err := ts.run("show foo/bar")
		require.NoError(t, err)
	})
}
