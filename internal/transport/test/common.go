package test

import (
	"net"
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5"
	fixtures "github.com/go-git/go-git-fixtures/v5"
	"github.com/stretchr/testify/require"
)

func FixturesFactory(base, name string) func() string {
	return func() string {
		return filepath.Join(base, name)
	}
}

func PrepareRepository(t testing.TB, f *fixtures.Fixture, base string, name string) billy.Filesystem {
	fs := f.DotGit(fixtures.WithTargetDir(FixturesFactory(base, name)))
	err := fixtures.EnsureIsBare(fs)
	require.NoError(t, err)
	return fs
}

func FreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}

	return l.Addr().(*net.TCPAddr).Port, l.Close()
}

// ListenTCP listens localhost:0.
// It reserves the listener to be closed on t.CleanUp.
func ListenTCP(t testing.TB) *net.TCPListener {
	t.Helper()
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	t.Cleanup(func() {
		err := l.Close()
		if err != nil {
			require.ErrorIs(t, err, net.ErrClosed)
		}
	})

	return l.(*net.TCPListener)
}
