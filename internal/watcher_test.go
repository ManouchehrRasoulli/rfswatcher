package internal

import (
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatcher_WithFastExit(t *testing.T) {
	testPath := "./"
	defer goleak.VerifyNone(t)
	var run int64

	c := func(e Event) {
		t.Log(e)
		require.Equal(t, ExitName, e.Name)
		require.Equal(t, Exit, e.Op)
		atomic.AddInt64(&run, 1)
	}

	w, e := NewWatcher(testPath, WithCallbackFunction(c))
	require.NoError(t, e, "create watcher on test path.")

	w.Close()
	require.Equal(t, int64(1), run)
}

func TestWatcher_WithFastExitForTwoHooks(t *testing.T) {
	testPath := "./"
	defer goleak.VerifyNone(t)
	var run int64

	c1 := func(e Event) {
		t.Log("hook-1 : ", e)
		require.Equal(t, ExitName, e.Name)
		require.Equal(t, Exit, e.Op)
		atomic.AddInt64(&run, 1)
	}
	c2 := func(e Event) {
		t.Log("hook-2 : ", e)
		require.Equal(t, ExitName, e.Name)
		require.Equal(t, Exit, e.Op)
		atomic.AddInt64(&run, 1)
	}

	w, e := NewWatcher(testPath, WithCallbackFunction(c1), WithCallbackFunction(c2))
	require.NoError(t, e, "create watcher on test path.")

	w.Close()
	require.Equal(t, int64(2), run)
}

func TestWatcher_WithCreateChanges(t *testing.T) {
	testPath := "./"
	testFilePath := "./test.txt"

	type testData struct {
		Op       Op
		Name     string
		FileName string
	}

	testTable := [4]testData{
		{
			Op:       Create,
			Name:     testFilePath,
			FileName: testFilePath,
		},
		{
			Op:       Write,
			Name:     testFilePath,
			FileName: testFilePath,
		},
		{
			Op:       Remove,
			Name:     testFilePath,
			FileName: testFilePath,
		},
		{
			Op:       Exit,
			Name:     ExitName,
			FileName: "",
		},
	}

	var run1 int64
	c1 := func(e Event) {
		t.Log("hook - 1 : ", run1, e.String())
		td := testTable[run1]
		require.Equal(t, td.Op, e.Op)
		require.Equal(t, td.Name, e.Name)

		atomic.AddInt64(&run1, 1)
	}
	var run2 int64
	c2 := func(e Event) {
		t.Log("hook - 2 : ", run2, e.String())
		td := testTable[run2]
		require.Equal(t, td.Op, e.Op)
		require.Equal(t, td.Name, e.Name)

		atomic.AddInt64(&run2, 1)
	}

	w, e := NewWatcher(testPath, WithCallbackFunction(c1), WithCallbackFunction(c2))
	require.NoError(t, e, "create watcher on test path.")

	var f *os.File
	var err error
	{ // CREATE
		f, err = os.Create(testFilePath)
		require.NoError(t, err, "create temporary file.")
	}

	{ // WRITE
		_, err = f.WriteString("test string !")
		require.NoError(t, err, "write into file.")
		err = f.Close()
		require.NoError(t, err, "close file.")
	}

	{ // REMOVE
		err = os.Remove(f.Name())
		require.NoError(t, err, "remove file.")
	}

	time.Sleep(time.Millisecond * 10)
	{ // EXIT_DAEMON
		w.Close()
	}
	require.Equal(t, int64(4), run1)
	require.Equal(t, int64(4), run2)
}
