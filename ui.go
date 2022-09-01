package sendme

// Ui interface
type Ui interface {
	Logf(format string, args ...any) (int, error)
	Confirm(msg string) (int, error)
}
