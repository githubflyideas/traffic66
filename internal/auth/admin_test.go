package auth

import (
	"fmt"
	"sync"
	"testing"
)

// The administrator is who the file records, not whoever is first or is
// called "admin": a new password used to move the administrator to the end
// of the file, and adding a user named admin used to hand it the rights.
func TestAdminSurvivesPasswordChangeAndNamedAdmin(t *testing.T) {
	dir := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(Set(dir, "ops", "first-password-1"))
	must(Set(dir, "alice", "alice-password-1"))
	must(Set(dir, "ops", "second-password-1"))
	if a := NewFileChecker(dir, nil).Admin(); a != "ops" {
		t.Fatalf("after ops changed password, admin = %q", a)
	}
	must(Set(dir, "admin", "observer-pass-01"))
	if a := NewFileChecker(dir, nil).Admin(); a != "ops" {
		t.Fatalf("after adding a user named admin, admin = %q", a)
	}
	es, _ := Load(dir)
	if len(es) != 3 || es[0].User != "ops" {
		t.Fatalf("order not kept: %+v", es)
	}
}

// A file written before the record keeps its old rule, and the user the
// service starts with settles it.
func TestRecordAdminForOlderFiles(t *testing.T) {
	dir := t.TempDir()
	if err := write(dir, []Entry{{"alice", Hash("x")}, {"ops", Hash("y")}}, ""); err != nil {
		t.Fatal(err)
	}
	if a := NewFileChecker(dir, nil).Admin(); a != "alice" {
		t.Fatalf("old rule: admin = %q", a)
	}
	if err := RecordAdmin(dir, "ops"); err != nil {
		t.Fatal(err)
	}
	if a := NewFileChecker(dir, nil).Admin(); a != "ops" {
		t.Fatalf("recorded: admin = %q", a)
	}
	if err := RecordAdmin(dir, "alice"); err != nil {
		t.Fatal(err)
	}
	if a := NewFileChecker(dir, nil).Admin(); a != "ops" {
		t.Fatalf("a record is not replaced: admin = %q", a)
	}
}

// Changes at the same time are all kept.
func TestConcurrentSetKeepsEveryUser(t *testing.T) {
	dir := t.TempDir()
	if err := Set(dir, "admin", "admin-password-1"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- Set(dir, fmt.Sprintf("u%d", i), "user-password-1")
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	es, _ := Load(dir)
	if len(es) != 41 {
		t.Fatalf("%d users saved, want 41", len(es))
	}
}
