package main

import "testing"

func TestCommandOf(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
		rest int
	}{
		{nil, "serve", 0},
		{[]string{"--version"}, "--version", 0},
		{[]string{"-version"}, "-version", 0},
		{[]string{"--help"}, "--help", 0},
		{[]string{"-h"}, "-h", 0},
		{[]string{"-listen", ":8066"}, "serve", 2},
		{[]string{"TUI", "-x"}, "tui", 1},
	} {
		got, rest := commandOf(c.args)
		if got != c.want || len(rest) != c.rest {
			t.Errorf("commandOf(%q) = %q, %d args; want %q, %d", c.args, got, len(rest), c.want, c.rest)
		}
	}
}

// Every command has help in the same form: usage, what it does, examples.
func TestEveryCommandHasHelp(t *testing.T) {
	for _, c := range append(commands, "pcap") {
		if c == "help" {
			continue
		}
		h, ok := helps[c]
		if !ok || h.usage == "" || h.about == "" {
			t.Errorf("command %q has no help", c)
		}
	}
}
