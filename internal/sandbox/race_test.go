package sandbox

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/enrich"
)

// Add used to hand back a pointer into the sandbox's own state, and the
// upload handler then serialised it outside the lock — while the import
// worker, which Add has just started, was writing the same file's status.
// Run this with -race.
func TestAddDoesNotHandOutSharedState(t *testing.T) {
	var buf bytes.Buffer
	if err := Sample(&buf, time.Now().Add(-2*time.Hour).Truncate(time.Minute)); err != nil {
		t.Fatal(err)
	}
	sb := New(t.TempDir(), enrich.NewInventory(), enrich.NewASNDB(), enrich.NewThreats())
	defer sb.Close()
	f, err := sb.Add(SampleName, &buf, true)
	if err != nil {
		t.Fatal(err)
	}
	// what the handler does with what Add returned, while the import runs
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 4000; i++ {
			if _, err := json.Marshal(f); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	for i := 0; i < 400; i++ {
		sb.Info()
		time.Sleep(time.Millisecond)
	}
	<-done
}
