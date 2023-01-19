package pkg

import (
	"github.com/ManouchehrRasoulli/rfswatcher/internal"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"testing"
	"time"
)

func TestIntegration(t *testing.T) {
	t.Log("Start integration test ...")
	lg := log.New(os.Stdout, "integration --> ", 1|4)

	fileHandler, err := internal.NewHandler(".", lg)
	require.NoError(t, err, "internal handler !")

	address := "localhost:9801"
	s := NewServer(address, lg, fileHandler)
	w, err := internal.NewWatcher(".", internal.WithCallbackFunction(fileHandler.EventHook), internal.WithCallbackFunction(s.EventHook))
	require.NoError(t, err, "new watcher error !")
	defer w.Close()

	go func() {
		err := s.Run()
		require.NoError(t, err, "server error !")
	}()

	c := NewClient(address, lg)
	go func() {
		err := c.Run()
		require.NoError(t, err, "client error !")
	}()

	timer := time.AfterFunc(time.Minute*2, func() {
		err = c.Exit()
		require.NoError(t, err, "client exit !!")

		err = s.Exit()
		require.NoError(t, err, "server exit !!")
	})

	<-timer.C
	t.Log("Integration test done.")
}
