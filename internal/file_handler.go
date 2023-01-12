package internal

import (
	"fmt"
	"io/ioutil"
	"log"
	"sync"
)

type fStatus int32

const (
	clean fStatus = iota
	dirty
)

func (s fStatus) String() string {
	switch s {
	case clean:
		return "clean"
	case dirty:
		return "dirty"
	default:
		return "-"
	}
}

type fMeta struct {
	fName string
	size  int64
	stat  fStatus
}

func (f fMeta) String() string {
	return fmt.Sprintf("file-name: %s, size: %d, status: %s", f.fName, f.stat, f.stat)
}

type Handler struct {
	// meta
	// inorder to handle list of files and their statuses we will
	// use following map to handle metadata information about files.
	meta   map[string]fMeta
	w      *Watcher
	rwM    sync.RWMutex
	path   string
	logger *log.Logger
}

func NewHandler(path string, logger *log.Logger) (*Handler, error) {
	h := Handler{
		meta:   make(map[string]fMeta),
		w:      nil,
		rwM:    sync.RWMutex{},
		path:   path,
		logger: logger,
	}

	w, err := NewWatcher(path, WithCallbackFunction(h.changeHook))
	if err != nil {
		return nil, err
	}
	h.w = w

	if err = h.readDir(path); err != nil {
		return nil, err
	}

	return &h, nil
}

func (h *Handler) readDir(path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !f.IsDir() { // TODO :: remove inorder to watch sub-paths too
			meta := fMeta{
				fName: f.Name(),
				size:  f.Size(),
				stat:  clean,
			}

			h.logger.Printf("handler :: got file with following meta --> %s\n", meta)

			h.meta[f.Name()] = meta
		}
	}

	return nil
}

func (h *Handler) ListFiles() {
	for _, meta := range h.meta {
		fmt.Println(meta)
	}
}

func (h *Handler) Stop() {
	h.w.Close()
}

func (h *Handler) changeHook(e Event) {
	fmt.Println("handler --> ", e)
}
