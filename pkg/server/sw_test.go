package server

import (
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

var (
	lg *log.Logger
)

func TestMain(m *testing.M) {
	lg = log.New(os.Stdout, "test --> ", 1|4)
	m.Run()
}

func TestServer_Listen(t *testing.T) {
	wg := sync.WaitGroup{}
	s := NewServer("localhost:12001", nil, lg)

	err := s.Listen()
	require.NoError(t, err, "server listening !")

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = s.Run()
		require.Error(t, err, "server running !")
	}()

	time.Sleep(time.Second)

	err = s.Close()
	require.NoError(t, err, "close server !")
	wg.Wait()
}
