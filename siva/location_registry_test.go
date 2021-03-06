package siva

import (
	"fmt"
	"testing"

	borges "github.com/src-d/go-borges"
	"github.com/stretchr/testify/require"
)

func point(p interface{}) string {
	return fmt.Sprintf("%p", p)
}

func TestRegistryNoCache(t *testing.T) {
	require := require.New(t)

	lib := setupLibrary(t, "test", &LibraryOptions{
		Transactional: true,
		RegistryCache: 0,
	})

	// to guarantee transactionality mustn't be two instances of the same
	// location, so by default de RegistryCache size is greater than one,
	// that is always location caching in transactional mode.

	require.True(lib.options.RegistryCache == registryCacheSize)

	// when there is a transaction it reuses the location

	loc1, err := lib.Location("foo-bar")
	require.NoError(err)

	r, err := loc1.Get("github.com/foo/bar", borges.RWMode)
	require.NoError(err)

	loc2, err := lib.Location("foo-bar")
	require.NoError(err)

	require.Equal(point(loc1), point(loc2))

	err = r.Close()
	require.NoError(err)

	// same case but with commit

	r, err = loc1.Get("github.com/foo/bar", borges.RWMode)
	require.NoError(err)

	loc2, err = lib.Location("foo-bar")
	require.NoError(err)

	require.Equal(point(loc1), point(loc2))

	err = r.Commit()
	require.True(ErrEmptyCommit.Is(err))
}

func TestRegistryCache(t *testing.T) {
	require := require.New(t)

	lib := setupLibrary(t, "test", &LibraryOptions{
		Transactional: true,
		RegistryCache: 1,
	})

	// as the capacity is 1 getting the same location twice returns the same
	// object
	loc1, err := lib.Location("foo-bar")
	require.NoError(err)
	loc2, err := lib.Location("foo-bar")
	require.NoError(err)

	require.Equal(point(loc1), point(loc2))

	// getting another location should swipe the first location from cache

	_, err = lib.Location("foo-qux")
	require.NoError(err)

	loc2, err = lib.Location("foo-bar")
	require.NoError(err)

	require.NotEqual(point(loc1), point(loc2))
}
