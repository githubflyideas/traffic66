package store

import (
	"fmt"
	"log"
	"strconv"
)

// FormatVersion is the layout of the data directory this version writes:
// the tables, their columns and the flow files. It goes up only when a
// change means an older traffic66 could not read the data correctly; such
// a version must back up what it changes before changing it.
const FormatVersion = 1

// checkFormat records the layout and the version that last opened the
// data directory, and refuses a directory written in a newer layout than
// this version knows (going back to an older version after an upgrade),
// rather than misreading it or writing it in the old layout.
func (s *Store) checkFormat() error {
	prev := s.Meta("written_by")
	if v := s.Meta("format"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n > FormatVersion {
			by := prev
			if by == "" {
				by = "a newer version"
			}
			return fmt.Errorf("the data in %s was written by traffic66 %s (data format %s), which this version (%s, data format %d) cannot read: "+
				"start that version again, or this one with another -data directory", s.opt.Dir, by, v, s.opt.Version, FormatVersion)
		}
	}
	if err := s.SetMeta("format", strconv.Itoa(FormatVersion)); err != nil {
		return err
	}
	if s.opt.Version != "" && prev != s.opt.Version {
		if prev != "" {
			log.Printf("store: data directory last opened by traffic66 %s, now %s", prev, s.opt.Version)
		}
		return s.SetMeta("written_by", s.opt.Version)
	}
	return nil
}
