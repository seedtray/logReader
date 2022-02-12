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
	watcher := PollingFileWatcher{t.Name(), TestFs, false}
	updates, err := watcher.Start()
	if err != nil {
		t.Error(err)
	}
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

func TestSendsErrorIfStatFails(t *testing.T) {
	_, err := TestFs.OpenFile(t.Name(), syscall.O_CREAT|syscall.O_APPEND|syscall.O_SYNC, 0600)
	if err != nil {
		t.Error(err)
	}
	watcher := PollingFileWatcher{t.Name(), TestFs, false}
	updates, err := watcher.Start()
	if err != nil {
		t.Error(err)
	}
	select {
	case <-updates:
	case <-time.After(10 * pollInterval):
		t.Errorf("didn't get initial update")
	}
	if err = TestFs.Remove(t.Name()); err != nil {
		t.Error(err)
	}
	select {
	case err = <-updates:
		if err == nil {
			t.Errorf("didn't receive an expected error")
		}
	case <-time.After(10 * pollInterval):
		t.Errorf("watcher didn't signal a file change.")
	}

}
