package main

import (
	"reflect"
	"testing"
)

func TestCleanArgs(t *testing.T) {
	got := cleanArgs([]string{"　interfaces'.", "－capture", "“Wi-Fi”", " ", "-listen", ""})
	want := []string{"interfaces", "-capture", "Wi-Fi", "-listen", ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
	if s := nearestCommand("interface"); s != "interfaces" {
		t.Fatalf("nearest %q", s)
	}
	if s := nearestCommand("zzzzzz"); s != "" {
		t.Fatalf("nearest %q", s)
	}
}
