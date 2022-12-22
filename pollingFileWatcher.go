package logReader

import (
	"context"
	"github.com/spf13/afero"
	"time"
)

type PollingFileWatcher struct {
	filename string
	fs       afero.Fs
	err      error
}

var _ FileWatcher = &PollingFileWatcher{}

func NewOsPollingFileWatcher(filename string) *PollingFileWatcher {
	return &PollingFileWatcher{filename: filename, fs: afero.NewOsFs()}
}

func (pw *PollingFileWatcher) Start() (<-chan UpdateSignal, func()) {
	updates := make(chan UpdateSignal)
	ctx, cancel := context.WithCancel(context.Background())
	go pw.watch(ctx, updates)
	return updates, cancel
}

const pollInterval = 10 * time.Millisecond
const refreshInterval = 1 * time.Second

var updateSignal UpdateSignal = struct{}{}

func (pw *PollingFileWatcher) watch(ctx context.Context, updates chan UpdateSignal) {
	var lastSize int64 = 0
	lastModTime := time.Unix(0, 0)
	for {
		fileInfo, err := pw.fs.Stat(pw.filename)
		if err != nil {
			pw.err = err
			close(updates)
			return
		}
		currentSize := fileInfo.Size()
		currentModTime := fileInfo.ModTime()
		if currentSize != lastSize || !currentModTime.Equal(lastModTime) {
			// if we're blocked for too long, should we refresh lastSize/lastModTime?
			select {
			case updates <- updateSignal:
				// Notice that we update lastSize and ModTIme after we sent the notification.
				// This is needed in order for refreshInterval to work.
				lastSize = currentSize
				lastModTime = currentModTime
			case <-ctx.Done():
				close(updates)
				return
			case <-time.After(refreshInterval):
				// RefreshInterval is an optimization for when the client reacting to watched files takes too long
				// to acknowledge an update signal. Consider the following:
				// This function finds a file change, so it tries to send an update signal.
				// The receiver doesn't read the channel for a couple seconds.
				// In the meantime, the file changes again.
				// The receiver reads the channel and reacts to it, unblocking this sender.
				// On the following loop, this function finds an immediate file change, even if the client possibly
				// already consumed it.
				continue
			}
		}
		select {
		case <-time.After(pollInterval):
			continue
		case <-ctx.Done():
			close(updates)
			return
		}
	}
}

func (pw *PollingFileWatcher) Err() error {
	return pw.err
}
