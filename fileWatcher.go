package logReader

type UpdateSignal struct{}

type FileWatcher interface {
	// Start starts watching the file.
	// It returns a channel that will receive updates on file changes, and a cancel function that stops the watcher.
	// The returned channel will be closed upon any errors found while watching.
	Start() (<-chan UpdateSignal, func())

	// Err returns any error condition found while watching.
	Err() error
}
