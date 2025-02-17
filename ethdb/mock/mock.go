package mock

import (
	"context"

	"github.com/zeus-fyi/gochain/v4/ethdb"
)

var _ ethdb.Segment = (*Segment)(nil)

type Segment struct {
	CloseFunc    func() error
	NameFunc     func() string
	PathFunc     func() string
	HasFunc      func(key []byte) (bool, error)
	GetFunc      func(key []byte) ([]byte, error)
	IteratorFunc func() ethdb.SegmentIterator
}

func (m *Segment) Close() error                    { return m.CloseFunc() }
func (m *Segment) Name() string                    { return m.NameFunc() }
func (m *Segment) Path() string                    { return m.PathFunc() }
func (m *Segment) Has(key []byte) (bool, error)    { return m.HasFunc(key) }
func (m *Segment) Get(key []byte) ([]byte, error)  { return m.GetFunc(key) }
func (m *Segment) Iterator() ethdb.SegmentIterator { return m.IteratorFunc() }

var _ ethdb.MutableSegment = (*MutableSegment)(nil)

type MutableSegment struct {
	Segment
	PutFunc    func(key, value []byte) error
	DeleteFunc func(key []byte) error
}

func (m *MutableSegment) Put(key, value []byte) error { return m.PutFunc(key, value) }
func (m *MutableSegment) Delete(key []byte) error     { return m.DeleteFunc(key) }

var _ ethdb.SegmentOpener = (*SegmentOpener)(nil)

type SegmentOpener struct {
	ListSegmentNamesFunc func(path, table string) ([]string, error)
	OpenSegmentFunc      func(table, name, path string) (ethdb.Segment, error)
}

func (m *SegmentOpener) ListSegmentNames(path, table string) ([]string, error) {
	return m.ListSegmentNamesFunc(path, table)
}

func (m *SegmentOpener) OpenSegment(table, name, path string) (ethdb.Segment, error) {
	return m.OpenSegmentFunc(table, name, path)
}

var _ ethdb.SegmentCompactor = (*SegmentCompactor)(nil)

type SegmentCompactor struct {
	CompactSegmentFunc   func(ctx context.Context, table string, s *ethdb.LDBSegment) (ethdb.Segment, error)
	UncompactSegmentFunc func(ctx context.Context, table string, s ethdb.Segment) (*ethdb.LDBSegment, error)
}

func (m *SegmentCompactor) CompactSegment(ctx context.Context, table string, s *ethdb.LDBSegment) (ethdb.Segment, error) {
	return m.CompactSegmentFunc(ctx, table, s)
}

func (m *SegmentCompactor) UncompactSegment(ctx context.Context, table string, s ethdb.Segment) (*ethdb.LDBSegment, error) {
	return m.UncompactSegmentFunc(ctx, table, s)
}
