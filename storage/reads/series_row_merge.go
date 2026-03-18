package reads

import (
	"bytes"

	"github.com/influxdata/influxdb/tsdb/cursors"
)

func sameSeriesRow(a, b *SeriesRow) bool {
	if a == nil || b == nil {
		return false
	}

	// SortKey should uniquely identify a series row in storage paths.
	// If present on both rows, prefer this fast comparison.
	if len(a.SortKey) > 0 && len(b.SortKey) > 0 {
		return bytes.Equal(a.SortKey, b.SortKey)
	}

	// Fallback for tests and codepaths that do not populate SortKey.
	if !bytes.Equal(a.Name, b.Name) || a.Field != b.Field {
		return false
	}
	return a.Tags.Equal(b.Tags)
}

func cloneSeriesRow(row *SeriesRow) SeriesRow {
	cloned := *row
	cloned.Query = append(cursors.CursorIterators(nil), row.Query...)
	return cloned
}
