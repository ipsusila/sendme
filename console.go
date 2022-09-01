package sendme

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type consoleUi struct {
}

// NewUi for the console
func NewUi(conf *Config) (Ui, error) {
	return &consoleUi{}, nil
}

func (c *consoleUi) Confirm(msg string) (int, error) {
	fmt.Print(msg)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(scanner.Text())
		switch answer {
		case "Y", "y":
			return ActSend, nil
		case "N", "n":
			return ActDontSend, nil
		case "A", "a":
			return ActSendAll, nil
		case "C", "c":
			return ActAbortSend, nil
		}
	}
	return ActDontSend, nil
}
func (c *consoleUi) Logf(format string, args ...any) (int, error) {
	return fmt.Printf(format, args...)
}
