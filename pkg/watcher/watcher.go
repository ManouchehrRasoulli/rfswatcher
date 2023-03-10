package watcher

import (
	"fmt"
	"github.com/ManouchehrRasoulli/rfswatcher/pkg/model"
	"github.com/fsnotify/fsnotify"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

type Option func(w *Watcher)

func WithCallbackFunction(hook func(e model.Event, err error)) Option {
	return func(w *Watcher) {
		ech := w.sub()

		go func(ech chan model.Event) {
			defer w.wg.Done()
			for {
				select {
				case e := <-ech:
					if e.Op == model.Create {
						fs, _ := os.Stat(e.Name)
						if fs != nil && fs.IsDir() {
							_ = w.fw.Add(e.Name)
						}
					}
					hook(e, nil)
				case <-w.closed:
					for e := range ech {
						hook(e, nil)
					}
					return
				}
			}
		}(ech)
	}
}

func WithBufferSize(size int32) Option {
	return func(w *Watcher) {
		w.bufferSize = size
	}
}

type Watcher struct {
	fw         *fsnotify.Watcher
	closed     chan struct{}
	subs       []chan model.Event
	bufferSize int32
	wg         sync.WaitGroup
	path       string
}

func NewWatcher(path string, options ...Option) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := Watcher{
		fw:         fw,
		closed:     make(chan struct{}),
		subs:       make([]chan model.Event, 0),
		bufferSize: 25,
		wg:         sync.WaitGroup{},
		path:       path,
	}

	err = w.watchPath(path)
	if err != nil {
		return nil, err
	}

	for _, op := range options {
		op(&w)
	}

	go w.run()

	return &w, nil
}

func (w *Watcher) watchPath(path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() {
			err = w.watchPath(fmt.Sprintf("%s/%s", path, f.Name()))
			if err != nil {
				return err
			}
		}
	}

	err = w.fw.Add(path)
	if err != nil {
		return err
	}
	return nil
}

func (w *Watcher) fanOut(e fsnotify.Event) {
	event := model.Event{
		Name: e.Name,
		Op:   model.Op(e.Op),
	}

	for i := range w.subs {
		w.subs[i] <- event
	}
}

func (w *Watcher) run() {
	for {
		select {
		case e := <-w.fw.Events:
			if len(e.Name) == 0 { // no event !
				continue
			}

			e.Name = strings.TrimPrefix(e.Name, "./")
			w.fanOut(e)
		case <-w.closed:
			exitEvent := fsnotify.Event{
				Name: model.ExitName,
				Op:   fsnotify.Op(model.Exit),
			}
			w.fanOut(exitEvent)
			for i := range w.subs {
				close(w.subs[i])
			}
			return
		}
	}
}

func (w *Watcher) sub() chan model.Event {
	ch := make(chan model.Event, w.bufferSize)
	w.subs = append(w.subs, ch)
	w.wg.Add(1)
	return ch
}

func (w *Watcher) Close() {
	_ = w.fw.Close() // Close filesystem watcher
	close(w.closed)  // Close local threads
	w.wg.Wait()
}
