package store

import "time"

// RebuildSummaries exposes the rebuild to the tests of this directory.
func (s *Store) RebuildSummaries(from, to time.Time) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	return s.rebuildSummaries(from, to)
}
