package sendme

import (
	"fmt"
)

type consoleUi struct {
}

// NewUi for the console
func NewUi(conf *Config) (Ui, error) {
	return &consoleUi{}, nil
}

func (c *consoleUi) Confirm(msg string) (int, error) {
	fmt.Print("Send email (Y/N/Abort)?")

	// TODO:
	return ActSend, nil
}
func (c *consoleUi) Logf(format string, args ...any) (int, error) {
	return fmt.Printf(format, args...)
}
