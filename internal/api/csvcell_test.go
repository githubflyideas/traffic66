package api

import "testing"

func TestCSVCell(t *testing.T) {
	for in, want := range map[string]string{
		"Gi0/1":                   "Gi0/1",
		"":                        "",
		"=HYPERLINK(\"x\",\"y\")": "'=HYPERLINK(\"x\",\"y\")",
		"+1":                      "'+1",
		"-cmd":                    "'-cmd",
		"@SUM(A1)":                "'@SUM(A1)",
		"\tx":                     "'\tx",
		"10.0.0.1":                "10.0.0.1",
	} {
		if got := csvCell(in); got != want {
			t.Errorf("csvCell(%q) = %q, want %q", in, got, want)
		}
	}
}
