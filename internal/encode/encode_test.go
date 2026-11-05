package encode

import (
	"encoding/binary"
	"testing"
)

// A v9 header counts every record: templates, options templates and data.
func TestV9CountsTemplateRecords(t *testing.T) {
	m := NewV9()
	m.AddTemplate(Template{ID: 256, Fields: []Field{{ID: 1, Len: 4}}})
	m.AddTemplate(Template{ID: 257, Scope: []Field{{ID: 1, Len: 4}}, Fields: []Field{{ID: 34, Len: 4}}})
	m.AddData(257, []Values{{U(0, 4), U(100, 4)}})
	m.AddData(256, []Values{{U(1, 4)}, {U(2, 4)}, {U(3, 4)}})
	if got := binary.BigEndian.Uint16(m.V9(0, 0, 1, 1)[2:]); got != 6 {
		t.Fatalf("v9 count = %d, want 6 (2 templates + 4 data records)", got)
	}
}
