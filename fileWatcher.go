package logReader

type updateSignal error

type FileWatcher interface {
	Start() (<-chan updateSignal, error)
	Stop() error
}
