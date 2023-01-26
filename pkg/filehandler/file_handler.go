package filehandler

import (
	"fmt"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/model"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type Meta struct {
	Name       string
	Size       int64
	ModifyTime time.Time
}

func (f Meta) String() string {
	return fmt.Sprintf("file meata :: file-name: %s, size: %d, modified_at: %v", f.Name, f.Size, f.ModifyTime.String())
}

type Handler struct {
	// meta
	// inorder to handle list of files and their statuses we will
	// use following map to handle metadata information about files.
	meta   map[string]Meta
	rwM    sync.RWMutex
	path   string
	logger *log.Logger
}

func NewHandler(path string, logger *log.Logger) (*Handler, error) {
	logger.Printf("NEW handler :: on path %s\n", path)

	h := Handler{
		meta:   make(map[string]Meta),
		rwM:    sync.RWMutex{},
		path:   path,
		logger: logger,
	}

	h.rwM.Lock()
	defer h.rwM.Unlock()
	if err := h.readDir(path, 0); err != nil {
		return nil, err
	}

	return &h, nil
}

func (h *Handler) GetMeta(name string) *Meta {
	name = strings.TrimPrefix(name, h.path)
	if m, c := h.meta[name]; c {
		metaCopy := m
		return &metaCopy
	}
	return nil
}

func (h *Handler) readDir(path string, level int) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	for _, f := range files {
		var fName string
		if level > 0 {
			fName = fmt.Sprintf("%s/%s", path, f.Name())
		} else {
			fName = f.Name()
		}
		if !f.IsDir() {
			meta := Meta{
				Name:       fName,
				Size:       f.Size(),
				ModifyTime: f.ModTime(),
			}

			h.logger.Printf("handler :: got file with following meta --> %s\n", meta)
			h.meta[fName] = meta
			continue
		} else {
			err = h.readDir(fName, level+1)
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

func (h *Handler) RemoveFile(name string) error {
	h.rwM.Lock()
	defer h.rwM.Unlock()

	name = strings.TrimPrefix(name, "./")
	name = fmt.Sprintf("%s/%s", h.path, name)

	_ = os.RemoveAll(name)
	return h.readDir(h.path, 0)
}

func (h *Handler) ReadFile(name string) ([]byte, error) {
	h.rwM.RLock()
	defer h.rwM.RUnlock()

	name = strings.TrimPrefix(name, ".")
	name = strings.TrimPrefix(name, h.path)

	_, ok := h.meta[name]
	if !ok {
		return nil, fmt.Errorf("invalid file name %s", name)
	}

	name = fmt.Sprintf("%s/%s", h.path, name)

	return os.ReadFile(name)
}

func (h *Handler) WriteFile(name string, data []byte) error {
	h.rwM.Lock()
	defer h.rwM.Unlock()

	name = strings.TrimPrefix(name, ".")
	name = strings.TrimPrefix(name, "/")
	name = fmt.Sprintf("%s/%s", h.path, name)
	_, err := os.Stat(name)
	if err != nil {
		nl := strings.Split(name, "/")

		if len(nl) == 1 {
			// create file only
			f, err := os.Create(name)
			if err != nil {
				return fmt.Errorf("error create file %v", err)
			}

			n, err := f.Write(data)
			if n != len(data) || err != nil {
				return fmt.Errorf("error on write into file %s - %d,%d - %v", name, n, len(data), err)
			}

			return h.readDir(h.path, 0) // update local cache files
		}

		path := strings.Join(nl[:len(nl)-1], "/")
		err = os.MkdirAll(path, 0777)
		if err != nil {
			return fmt.Errorf("error %v create path %s", err, path)
		}

		f, err := os.Create(name)
		if err != nil {
			return fmt.Errorf("error %v create file, %s", err, name)
		}

		n, err := f.Write(data)
		if n != len(data) || err != nil {
			return fmt.Errorf("error on write into file %s - %d,%d - %v", name, n, len(data), err)
		}

		return h.readDir(h.path, 0) // update local cache files
	}

	// rewrite given file
	return os.WriteFile(name, data, 0777)
}

// EventHook
// handler callback function inorder to bee used in watcher
func (h *Handler) EventHook(e model.Event, err error) {
	if err != nil {
		h.logger.Printf("ERROR handler :: got error %v on hook\n", err)
		return
	}

	if strings.Contains(e.Name, "swp") ||
		strings.Contains(e.Name, ".goutputstream") ||
		strings.HasSuffix(e.Name, "~") ||
		strings.HasPrefix(e.Name, "exit") ||
		e.Op.Has(model.Chmod) {
		return
	}

	if e.Op == model.Remove || e.Op == model.Rename {
		h.logger.Printf("handler :: remove file meta --> %s, on event %s\n", h.meta[e.Name], e)
		delete(h.meta, e.Name)
		return
	}

	fs, err := os.Stat(e.Name)
	if err != nil {
		h.logger.Printf("ERROR handler :: got error %v, on event %s\n", err, e)
		return
	}

	e.Name = strings.TrimPrefix(e.Name, h.path)
	if !fs.IsDir() {
		h.rwM.Lock()
		defer h.rwM.Unlock()

		meta := Meta{
			Name:       e.Name,
			Size:       fs.Size(),
			ModifyTime: fs.ModTime(),
		}
		if _, contains := h.meta[e.Name]; contains {
			h.logger.Printf("handler :: got modification on file meta --> %s, on event %s\n", h.meta[e.Name], e)
		} else {
			h.logger.Printf("handler :: got new file meta --> %s, on event %s\n", h.meta[e.Name], e)
		}
		h.meta[e.Name] = meta
	}
}
