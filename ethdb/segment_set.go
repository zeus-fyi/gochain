package ethdb

import (
	"context"
	"fmt"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/zeus-fyi/gochain/v4/log"
	"golang.org/x/sync/semaphore"
)

// SegmentSet represents a set of segments.
type SegmentSet struct {
	mu       sync.RWMutex
	segments map[string]Segment // all segments

	semaphore *semaphore.Weighted // cache semaphore
	cache     *lru.Cache          // opened segments
}

// NewSegmentSet returns a new instance of SegmentSet.
func NewSegmentSet(maxOpenCount int) *SegmentSet {
	if maxOpenCount < 1 {
		maxOpenCount = 1
	}

	ss := &SegmentSet{
		semaphore: semaphore.NewWeighted(int64(maxOpenCount)),
		segments:  make(map[string]Segment),
	}
	ss.cache, _ = lru.NewWithEvict(maxOpenCount, ss.onEvicted)
	return ss
}

// Len returns the number of segments in the set.
func (ss *SegmentSet) Len() int {
	ss.mu.RLock()
	n := len(ss.segments)
	ss.mu.RUnlock()
	return n
}

// Add adds s to the set.
func (ss *SegmentSet) Add(s Segment) {
	ss.mu.Lock()
	ss.segments[s.Name()] = s
	ss.mu.Unlock()
}

// Contains returns true if name is in the set.
func (ss *SegmentSet) Contains(name string) bool {
	ss.mu.Lock()
	_, ok := ss.segments[name]
	ss.mu.Unlock()
	return ok
}

// Remove removes the segment with the given name from the set.
func (ss *SegmentSet) Remove(ctx context.Context, name string) {
	ss.mu.Lock()
	delete(ss.segments, name)
	ss.mu.Unlock()

	if ss.semaphore.Acquire(ctx, 1) != nil {
		return // cache Purge will Remove this segment
	}
	ss.cache.Remove(name)
	ss.semaphore.Release(1)
}

// Acquire returns a segment by name from the set and adds increments the semaphore.
// If the segment is unopened then it is opened before returning. If a segment
// is successfully retruns then Release() must always be called by the caller.
func (ss *SegmentSet) Acquire(name string) (Segment, error) {
	ss.semaphore.Acquire(context.Background(), 1)

	// Fetch from open segment cache first.
	if s, ok := ss.cache.Get(name); ok {
		return s.(Segment), nil
	}

	// Attempt to fetch from set of all segments.
	ss.mu.RLock()
	s := ss.segments[name]
	ss.mu.RUnlock()
	if s == nil {
		ss.semaphore.Release(1)
		return nil, nil
	}

	// Open and add to cache.
	if s, ok := s.(interface {
		Open() error
	}); ok {
		if err := s.Open(); err != nil {
			ss.semaphore.Release(1)
			return nil, err
		}
	}
	ss.cache.Add(name, s)

	return s, nil
}

// Release decrements the semaphore on the set.
func (ss *SegmentSet) Release() {
	ss.semaphore.Release(1)
}

func (ss *SegmentSet) onEvicted(key, value interface{}) {
	ss.mu.Lock()
	s := ss.segments[key.(string)]
	ss.mu.Unlock()

	if s == nil {
		return
	}

	var err error
	var action string
	switch s := s.(type) {
	case interface {
		Segment
		Purge() error
	}:
		log.Info("Purge local segment", "name", s.Name(), "path", s.Path())
		err = s.Purge()
		action = "purge"
	default:
		err = s.Close()
		action = "close"
	}
	if err != nil {
		log.Error(fmt.Sprintf("Failed to %s segment", action), "name", s.Name(), "path", s.Path(), "error", err)
	}
}

func (ss *SegmentSet) Close() error {
	ss.mu.Lock()
	ss.segments = map[string]Segment{}
	ss.mu.Unlock()
	// Evict all
	ss.cache.Purge()
	return nil
}

// Slice returns a slice of all segments.
func (ss *SegmentSet) Slice() []Segment {
	a := make([]Segment, 0, len(ss.segments))
	for _, s := range ss.segments {
		a = append(a, s)
	}
	SortSegments(a)
	return a
}
