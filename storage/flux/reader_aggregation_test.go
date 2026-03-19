package storageflux

import (
	"testing"

	"github.com/influxdata/influxdb/models"
	"github.com/influxdata/influxdb/tsdb/cursors"
)

type testFloatCursor struct {
	arrays []*cursors.FloatArray
	idx    int
	closed bool
	err    error
	stats  cursors.CursorStats
}

func (c *testFloatCursor) Next() *cursors.FloatArray {
	if c.idx >= len(c.arrays) {
		return nil
	}
	a := c.arrays[c.idx]
	c.idx++
	return a
}

func (c *testFloatCursor) Close()                     { c.closed = true }
func (c *testFloatCursor) Err() error                 { return c.err }
func (c *testFloatCursor) Stats() cursors.CursorStats { return c.stats }

type testIntegerCursor struct{}

func (c *testIntegerCursor) Next() *cursors.IntegerArray { return nil }
func (c *testIntegerCursor) Close()                      {}
func (c *testIntegerCursor) Err() error                  { return nil }
func (c *testIntegerCursor) Stats() cursors.CursorStats  { return cursors.CursorStats{} }

func TestSeriesBufferAddAndGetGroups(t *testing.T) {
	buffer := NewSeriesBuffer()
	tagsA := models.NewTags(map[string]string{"host": "a"})
	tagsB := models.NewTags(map[string]string{"host": "b"})

	buffer.Add(tagsA, &testFloatCursor{})
	buffer.Add(tagsA, &testFloatCursor{})
	buffer.Add(tagsB, &testFloatCursor{})

	groups := buffer.GetGroups()
	if got, want := len(groups), 2; got != want {
		t.Fatalf("unexpected group count: got %d, want %d", got, want)
	}

	if got, want := len(groups[0].cursors), 2; got != want {
		t.Fatalf("unexpected cursor count for first group: got %d, want %d", got, want)
	}
	if got, want := len(groups[1].cursors), 1; got != want {
		t.Fatalf("unexpected cursor count for second group: got %d, want %d", got, want)
	}
}

func TestSeriesBufferTypeMismatch(t *testing.T) {
	buffer := NewSeriesBuffer()
	tags := models.NewTags(map[string]string{"host": "a"})

	buffer.Add(tags, &testFloatCursor{})
	buffer.Add(tags, &testIntegerCursor{})

	groups := buffer.GetGroups()
	if got, want := len(groups), 1; got != want {
		t.Fatalf("unexpected group count: got %d, want %d", got, want)
	}
	if got, want := len(groups[0].cursors), 1; got != want {
		t.Fatalf("type mismatch cursor should be ignored: got %d, want %d", got, want)
	}
}

func TestGetCursorType(t *testing.T) {
	tests := []struct {
		name string
		cur  cursors.Cursor
		want string
	}{
		{name: "float", cur: &testFloatCursor{}, want: "float"},
		{name: "integer", cur: &testIntegerCursor{}, want: "integer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getCursorType(tt.cur); got != tt.want {
				t.Fatalf("unexpected cursor type: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnableSeriesAggregation(t *testing.T) {
	original := SeriesAggregationEnabled
	t.Cleanup(func() { SeriesAggregationEnabled = original })

	EnableSeriesAggregation(true)
	if !SeriesAggregationEnabled {
		t.Fatal("expected series aggregation to be enabled")
	}

	EnableSeriesAggregation(false)
	if SeriesAggregationEnabled {
		t.Fatal("expected series aggregation to be disabled")
	}
}

func TestMergeSeriesGroupFloat(t *testing.T) {
	group := &SeriesGroup{
		tags:       models.NewTags(map[string]string{"host": "a"}),
		cursorType: "float",
		cursors: []cursors.Cursor{
			&testFloatCursor{arrays: []*cursors.FloatArray{{Timestamps: []int64{1, 2}, Values: []float64{1, 2}}}},
			&testFloatCursor{arrays: []*cursors.FloatArray{{Timestamps: []int64{2, 3}, Values: []float64{20, 3}}}},
		},
	}

	cur, err := mergeSeriesGroup(group)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fc, ok := cur.(cursors.FloatArrayCursor)
	if !ok {
		t.Fatalf("unexpected cursor type %T", cur)
	}

	a := fc.Next()
	if a == nil {
		t.Fatal("expected merged array")
	}
	if got, want := len(a.Timestamps), 3; got != want {
		t.Fatalf("unexpected merged length: got %d, want %d", got, want)
	}
	if got, want := a.Values[1], 20.0; got != want {
		t.Fatalf("expected latest value for duplicate timestamp, got %v want %v", got, want)
	}
}
