package tui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// ReadSecret prints prompt and reads a line from the terminal without
// echoing it. When stdin is not a terminal it reads a plain line.
func ReadSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	if !isTerminal() {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}
	st, err := makeRaw()
	if err != nil {
		return "", err
	}
	defer func() { st.restore(); fmt.Fprint(os.Stderr, "\n") }()
	var b []byte
	buf := make([]byte, 1)
	for {
		if _, err := os.Stdin.Read(buf); err != nil {
			return "", err
		}
		switch c := buf[0]; c {
		case '\r', '\n':
			return string(b), nil
		case 3, 4: // Ctrl+C, Ctrl+D
			return "", errors.New("cancelled")
		case 8, 127:
			_, n := utf8.DecodeLastRune(b)
			b = b[:len(b)-n]
		default:
			b = append(b, c)
		}
	}
}
