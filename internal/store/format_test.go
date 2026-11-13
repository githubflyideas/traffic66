package store

import (
	"strings"
	"testing"
)

// Data written in a newer layout is refused with a message that says what
// to do, rather than misread; the version that opened it is recorded.
func TestFormatCheck(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(Options{Dir: dir, Version: "2.6.0"})
	if err != nil {
		t.Fatal(err)
	}
	if st.Meta("format") != "1" || st.Meta("written_by") != "2.6.0" {
		t.Fatalf("format %q written_by %q", st.Meta("format"), st.Meta("written_by"))
	}
	st.SetMeta("format", "99")
	st.SetMeta("written_by", "9.0.0")
	st.Close()
	_, err = Open(Options{Dir: dir, Version: "2.6.0"})
	if err == nil || !strings.Contains(err.Error(), "written by traffic66 9.0.0") || !strings.Contains(err.Error(), "another -data directory") {
		t.Fatalf("newer data opened: %v", err)
	}
}
