package internal

import (
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"testing"
)

var (
	lg *log.Logger
)

func TestMain(m *testing.M) {
	lg = log.New(os.Stdout, "test --> ", 1|4)
	m.Run()
}

func TestFileHandler_New(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err, "get working directory.")

	h, err := NewHandler(wd, lg)
	require.NoError(t, err, "read current working directory.")

	h.ListFiles()
}
