package internal

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func _TestIntegration(t *testing.T) {
	path := "test"
	fh, err := NewHandler(path, lg)
	require.NoError(t, err, "create file handler.")

	w, err := NewWatcher(path, WithCallbackFunction(fh.EventHook))
	require.NoError(t, err, "create watcher.")
	defer w.Close()

	time.Sleep(time.Second * 100)

	fh.ListFiles()
}
