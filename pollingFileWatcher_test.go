package logReader

import (
	"fmt"
	"syscall"
	"testing"
	"time"
)

func TestNotifiesOnAppend(t *testing.T) {
	file, err := TestFs.OpenFile(t.Name(), syscall.O_CREAT|syscall.O_APPEND|syscall.O_SYNC, 0600)
	if err != nil {
		t.Error(err)
	}
	watcher := PollingFileWatcher{filename: t.Name(), fs: TestFs}
	updates, stop := watcher.Start()
	defer stop()
	select {
	case <-updates:
	case <-time.After(10 * pollInterval):
		t.Errorf("didn't get initial update from file watcher")
	}
	select {
	case <-updates:
		t.Errorf("got a second update from watcher without an update")
	case <-time.After(10 * pollInterval):
	}
	if _, err = fmt.Fprintln(file, "An update"); err != nil {
		t.Error(err)
	}
	select {
	case <-updates:
	case <-time.After(10 * pollInterval):
		t.Errorf("watcher didn't signal a file change.")
	}
}

func TestClosesAndStoresErrorIfStatFails(t *testing.T) {
	_, err := TestFs.OpenFile(t.Name(), syscall.O_CREAT|syscall.O_APPEND|syscall.O_SYNC, 0600)
	if err != nil {
		t.Error(err)
	}
	watcher := PollingFileWatcher{filename: t.Name(), fs: TestFs}
	updates, stop := watcher.Start()
	defer stop()
	select {
	case <-updates:
	case <-time.After(10 * pollInterval):
		t.Errorf("didn't get initial update")
	}
	if err = TestFs.Remove(t.Name()); err != nil {
		t.Error(err)
	}
	select {
	case _, ok := <-updates:
		if ok {
			t.Errorf("didn't close the channel on an error")
		}
		if watcher.Err() == nil {
			t.Errorf("Error is nil after closing the channel.")
		}
	case <-time.After(10 * pollInterval):
		t.Errorf("watcher didn't signal a file change.")
	}
}
