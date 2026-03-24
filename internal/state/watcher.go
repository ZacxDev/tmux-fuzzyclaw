package state

import (
	"github.com/fsnotify/fsnotify"
)

// Watcher watches state directories for changes and sends notifications.
type Watcher struct {
	fsw    *fsnotify.Watcher
	Events chan string // file path that changed
	done   chan struct{}
}

// NewWatcher creates a file watcher on the given directories.
func NewWatcher(dirs ...string) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		fsw:    fsw,
		Events: make(chan string, 32),
		done:   make(chan struct{}),
	}
	for _, dir := range dirs {
		_ = fsw.Add(dir)
	}
	go w.loop()
	return w, nil
}

func (w *Watcher) loop() {
	defer close(w.done)
	for {
		select {
		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				select {
				case w.Events <- event.Name:
				default: // drop if channel full
				}
			}
		case _, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
		}
	}
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	err := w.fsw.Close()
	<-w.done
	return err
}
