package filehandler

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

func TestFileHandler_Copy(t *testing.T) {
	file := "test/somethings/test.txt"

	wd, err := os.Getwd()
	require.NoError(t, err, "get working directory.")

	h, err := NewHandler(wd, lg)
	require.NoError(t, err, "read current working directory.")

	err = h.WriteFile(file, []byte("test -----------> data somethings going on !!!"))
	require.NoError(t, err, "write into file.")

	data, err := h.ReadFile(file)
	require.NoError(t, err, "read file !!!")
	t.Log(string(data))

	err = h.RemoveFile(file)
	require.NoError(t, err, "remove file error !!")
}
