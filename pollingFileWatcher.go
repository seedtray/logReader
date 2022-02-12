package logReader

import (
	"errors"
	"github.com/spf13/afero"
	"time"
)

type PollingFileWatcher struct {
	filename string
	fs       afero.Fs
	watching bool
}

var _ FileWatcher = &PollingFileWatcher{}

func NewOsPollingFileWatcher(filename string) *PollingFileWatcher {
	return &PollingFileWatcher{filename, afero.NewOsFs(), false}
}

func (pw *PollingFileWatcher) Start() (<-chan updateSignal, error) {
	if pw.watching {
		return nil, errors.New("already watching")
	}
	updates := make(chan updateSignal)
	pw.watching = true
	go pw.watch(updates)
	return updates, nil
}

func (pw *PollingFileWatcher) Stop() error {
	if !pw.watching {
		return errors.New("not watching")
	}
	pw.watching = false
	return nil
}

const pollInterval = 1 * time.Millisecond

func (pw *PollingFileWatcher) watch(updates chan updateSignal) {
	var lastSize int64 = 0
	lastModTime := time.UnixMilli(0)
	for {
		if !pw.watching {
			close(updates)
			return
		}
		fileInfo, err := pw.fs.Stat(pw.filename)
		if err != nil {
			select {
			case updates <- err:
			}
			close(updates)
			return
		}
		currentSize := fileInfo.Size()
		currentModTime := fileInfo.ModTime()
		if currentSize != lastSize || !currentModTime.Equal(lastModTime) {
			select {
			case updates <- nil:
			default:
			}
			lastSize = currentSize
			lastModTime = currentModTime
		} else {
			time.Sleep(pollInterval)
		}
	}
}
