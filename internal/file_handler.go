package internal

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type fMeta struct {
	fName      string
	size       int64
	modifyTime time.Time
}

func (f fMeta) String() string {
	return fmt.Sprintf("file meata :: file-name: %s, size: %d, modified_at: %v", f.fName, f.size, f.modifyTime.String())
}

type Handler struct {
	// meta
	// inorder to handle list of files and their statuses we will
	// use following map to handle metadata information about files.
	meta   map[string]fMeta
	rwM    sync.RWMutex
	path   string
	logger *log.Logger
}

func NewHandler(path string, logger *log.Logger) (*Handler, error) {
	logger.Printf("NEW handler :: on path %s\n", path)

	h := Handler{
		meta:   make(map[string]fMeta),
		rwM:    sync.RWMutex{},
		path:   path,
		logger: logger,
	}

	h.rwM.Lock()
	defer h.rwM.Unlock()
	if err := h.readDir(path); err != nil {
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
		fName := fmt.Sprintf("%s/%s", path, f.Name())
		if !f.IsDir() {
			meta := fMeta{
				fName:      fName,
				size:       f.Size(),
				modifyTime: f.ModTime(),
			}

			h.logger.Printf("handler :: got file with following meta --> %s\n", meta)
			h.meta[fmt.Sprintf("%s/%s", path, f.Name())] = meta
			continue
		} else {
			err = h.readDir(fName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *Handler) ListFiles() {
	h.rwM.RLock()
	defer h.rwM.RUnlock()

	h.logger.Printf("handler :: list files ---- %d\n", len(h.meta))
	for _, meta := range h.meta {
		fmt.Println(meta)
	}
}

// EventHook
// handler callback function inorder to bee used in watcher
func (h *Handler) EventHook(e Event, err error) {
	if err != nil {
		h.logger.Printf("ERROR handler :: got error %v on hook\n", err)
		return
	}

	if strings.Contains(e.Name, "swp") ||
		strings.Contains(e.Name, ".goutputstream") ||
		e.Op.Has(Chmod) {
		return
	}

	if !strings.HasPrefix(e.Name, h.path) {
		// unrelated to this handler
		return
	}

	if e.Op == Remove || e.Op == Rename {
		h.logger.Printf("handler :: remove file meta --> %s, on event %s\n", h.meta[e.Name], e)
		delete(h.meta, e.Name)
		return
	}

	fs, err := os.Stat(e.Name)
	if err != nil {
		h.logger.Printf("ERROR handler :: got error %v, on event %s\n", err, e)
		return
	}

	if !fs.IsDir() {
		h.rwM.Lock()
		defer h.rwM.Unlock()

		meta := fMeta{
			fName:      e.Name,
			size:       fs.Size(),
			modifyTime: fs.ModTime(),
		}
		if _, contains := h.meta[e.Name]; contains {
			h.logger.Printf("handler :: got modification on file meta --> %s, on event %s\n", h.meta[e.Name], e)
		} else {
			h.logger.Printf("handler :: got new file meta --> %s, on event %s\n", h.meta[e.Name], e)
		}
		h.meta[e.Name] = meta

	}
}
