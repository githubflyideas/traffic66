package tui

import "unicode/utf8"

// runeWidth returns the terminal cell width of r (East Asian wide = 2).
func runeWidth(r rune) int {
	switch {
	case r == 0:
		return 0
	case r < 0x20 || (r >= 0x7f && r < 0xa0):
		return 0
	case r >= 0x300 && r <= 0x36f, r >= 0x200b && r <= 0x200f, r >= 0xfe00 && r <= 0xfe0f:
		return 0 // combining marks, zero-width, variation selectors
	case r >= 0x0900 && r <= 0x0DFF && isIndicMark(r):
		return 0
	case r >= 0x1100 && r <= 0x115f, r >= 0x2e80 && r <= 0x303e, r >= 0x3041 && r <= 0x33ff,
		r >= 0x3400 && r <= 0x4dbf, r >= 0x4e00 && r <= 0x9fff, r >= 0xa000 && r <= 0xa4cf,
		r >= 0xac00 && r <= 0xd7a3, r >= 0xf900 && r <= 0xfaff, r >= 0xfe30 && r <= 0xfe4f,
		r >= 0xff00 && r <= 0xff60, r >= 0xffe0 && r <= 0xffe6, r >= 0x1f300 && r <= 0x1f64f,
		r >= 0x1f900 && r <= 0x1f9ff, r >= 0x20000 && r <= 0x3fffd:
		return 2
	}
	return 1
}

// isIndicMark reports Devanagari/Bengali vowel signs and viramas that
// combine with the previous letter.
func isIndicMark(r rune) bool {
	o := r & 0x7F
	return (o >= 0x01 && o <= 0x03) || o == 0x3C || (o >= 0x3E && o <= 0x4D) || (o >= 0x51 && o <= 0x57) || o == 0x62 || o == 0x63
}

func strWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

// fit truncates (with "…") or pads s to exactly w cells.
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if sw := strWidth(s); sw <= w {
		return s + spaces(w-sw)
	}
	cur := 0
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		rw := runeWidth(r)
		if cur+rw > w-1 {
			break
		}
		out = append(out, s[i:i+n]...)
		cur += rw
		i += n
	}
	return string(out) + "…" + spaces(w-1-cur)
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}

// fitRight right-aligns s in w cells.
func fitRight(s string, w int) string {
	sw := strWidth(s)
	if sw >= w {
		return fit(s, w)
	}
	return spaces(w-sw) + s
}
