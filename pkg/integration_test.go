package pkg

import (
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/client"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/filehandler"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/server"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/watcher"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"testing"
	"time"
)

func TestIntegration(t *testing.T) {
	t.Log("Start integration test ...")
	lg := log.New(os.Stdout, "integration --> ", 1|4)

	fileHandler, err := filehandler.NewHandler(".", lg)
	require.NoError(t, err, "internal handler !")

	address := "localhost:9801"
	s := server.NewServer(address, ".", lg, fileHandler)
	w, err := watcher.NewWatcher(".", watcher.WithCallbackFunction(fileHandler.EventHook), watcher.WithCallbackFunction(s.EventHook))
	require.NoError(t, err, "new watcher error !")
	defer w.Close()

	go func() {
		err := s.Run()
		require.NoError(t, err, "server error !")
	}()

	time.Sleep(time.Second)

	c := client.NewClient(address, lg, fileHandler)
	go func() {
		err := c.Run()
		require.NoError(t, err, "client error !")
	}()

	exit := make(chan struct{})
	_ = time.AfterFunc(time.Second*2, func() {
		defer close(exit)
		err = c.Exit()
		require.NoError(t, err, "client exit !!")

		err = s.Exit()
		require.NoError(t, err, "server exit !!")
	})

	<-exit
	t.Log("Integration test done.")
}
