package reads_test

import (
	"context"
	"testing"

	"github.com/influxdata/influxdb/storage/reads"
	"github.com/influxdata/influxdb/storage/reads/datatypes"
	"github.com/influxdata/influxdb/tsdb/cursors"
)

func TestNewFilteredResultSet_TimeRange(t *testing.T) {
	newCursor := newMockReadCursor(
		"clicks click=1 1",
	)
	for i := range newCursor.rows {
		newCursor.rows[0].Query[i] = &mockCursorIterator{
			newCursorFn: func(req *cursors.CursorRequest) cursors.Cursor {
				if want, got := int64(0), req.StartTime; want != got {
					t.Errorf("unexpected start time -want/+got:\n\t- %d\n\t+ %d", want, got)
				}
				if want, got := int64(29), req.EndTime; want != got {
					t.Errorf("unexpected end time -want/+got:\n\t- %d\n\t+ %d", want, got)
				}
				return &mockIntegerArrayCursor{}
			},
		}
	}

	ctx := context.Background()
	req := datatypes.ReadFilterRequest{
		Range: &datatypes.TimestampRange{
			Start: 0,
			End:   30,
		},
	}

	resultSet := reads.NewFilteredResultSet(ctx, req.Range.GetStart(), req.Range.GetEnd(), &newCursor)
	if !resultSet.Next() {
		t.Fatal("expected result")
	}
}

func TestFilteredResultSet_MergesConsecutiveSeriesRows(t *testing.T) {
	newCursor := newMockReadCursor(
		"clicks,host=a value=1 1",
		"clicks,host=a value=2 2",
	)

	resultSet := reads.NewFilteredResultSet(context.Background(), 0, 30, &newCursor)
	if !resultSet.Next() {
		t.Fatal("expected first merged result")
	}
	if resultSet.Next() {
		t.Fatal("expected consecutive rows for same series to be merged into one result")
	}
}
