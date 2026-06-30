package tsm1

import (
	"context"
	"math"
	"os"
	"testing"

	"github.com/influxdata/influxdb/tsdb"
)

func testKeyCursor(ctx context.Context, fs *FileStore, key []byte, seek int64, ascending bool) *KeyCursor {
	if ascending {
		return fs.KeyCursor(ctx, key, seek, seek, math.MaxInt64, ascending)
	}
	return fs.KeyCursor(ctx, key, seek, math.MinInt64, seek, ascending)
}

func TestKeyCursor_RangeFiltersOverlappedBlocks(t *testing.T) {
	const key = "cpu"

	dir := MustTempDir()
	defer os.RemoveAll(dir)

	fs := NewFileStore(dir)
	oldBlock := func(start int64) keyValues {
		return keyValues{key, []Value{
			NewFloatValue(start, float64(start)),
			NewFloatValue(start+1, float64(start+1)),
		}}
	}

	coveringValues := make([]Value, 0, 12)
	for ts := int64(1); ts <= 12; ts++ {
		coveringValues = append(coveringValues, NewFloatValue(ts, float64(100+ts)))
	}

	files, err := newFiles(dir,
		oldBlock(1),
		oldBlock(3),
		oldBlock(5),
		oldBlock(7),
		oldBlock(9),
		keyValues{key, coveringValues},
	)
	if err != nil {
		t.Fatalf("unexpected error creating files: %v", err)
	}
	if err := fs.Replace(nil, files); err != nil {
		t.Fatalf("unexpected error replacing files: %v", err)
	}

	t.Run("ascending narrow", func(t *testing.T) {
		got, decoded := readFloatArrayRange(t, fs, []byte(key), 1, 1, 1, true)
		assertFloatArray(t, got, []int64{1}, []float64{101})
		if decoded != 2 {
			t.Fatalf("decoded blocks mismatch: got %d, exp 2", decoded)
		}
	})

	t.Run("descending narrow", func(t *testing.T) {
		got, decoded := readFloatArrayRange(t, fs, []byte(key), 10, 10, 10, false)
		assertFloatArray(t, got, []int64{10}, []float64{110})
		if decoded != 2 {
			t.Fatalf("decoded blocks mismatch: got %d, exp 2", decoded)
		}
	})

	t.Run("ascending wide", func(t *testing.T) {
		got, decoded := readFloatArrayRange(t, fs, []byte(key), 1, 1, 12, true)
		assertFloatArray(t, got,
			[]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			[]float64{101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112},
		)
		if decoded != 6 {
			t.Fatalf("decoded blocks mismatch: got %d, exp 6", decoded)
		}
	})
}

func readFloatArrayRange(t *testing.T, fs *FileStore, key []byte, seek, min, max int64, ascending bool) (*tsdb.FloatArray, int64) {
	t.Helper()

	ctx := NewContextWithMetricsGroup(context.Background())
	kc := fs.KeyCursor(ctx, key, seek, min, max, ascending)
	defer kc.Close()

	var got *tsdb.FloatArray
	if ascending {
		cur := newFloatArrayAscendingCursor()
		defer cur.Close()
		if err := cur.reset(seek, max, nil, kc); err != nil {
			t.Fatalf("unexpected error resetting cursor: %v", err)
		}
		got = cur.Next()
	} else {
		cur := newFloatArrayDescendingCursor()
		defer cur.Close()
		if err := cur.reset(seek, min, nil, kc); err != nil {
			t.Fatalf("unexpected error resetting cursor: %v", err)
		}
		got = cur.Next()
	}

	decoded := MetricsGroupFromContext(ctx).GetCounter(floatBlocksDecodedCounter).Value()
	return got, decoded
}

func assertFloatArray(t *testing.T, got *tsdb.FloatArray, timestamps []int64, values []float64) {
	t.Helper()

	if len(got.Timestamps) != len(timestamps) {
		t.Fatalf("timestamp count mismatch: got %d, exp %d", len(got.Timestamps), len(timestamps))
	}
	if len(got.Values) != len(values) {
		t.Fatalf("value count mismatch: got %d, exp %d", len(got.Values), len(values))
	}
	for i := range timestamps {
		if got.Timestamps[i] != timestamps[i] {
			t.Fatalf("timestamp %d mismatch: got %d, exp %d", i, got.Timestamps[i], timestamps[i])
		}
		if got.Values[i] != values[i] {
			t.Fatalf("value %d mismatch: got %v, exp %v", i, got.Values[i], values[i])
		}
	}
}
