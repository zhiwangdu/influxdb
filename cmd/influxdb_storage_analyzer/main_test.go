package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/influxdata/influxdb/models"
	"github.com/influxdata/influxdb/tsdb"
	"github.com/influxdata/influxdb/tsdb/engine/tsm1"
)

func TestAnalyzeTSM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "000000001-000000001.tsm")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatal(err)
	}
	values := tsm1.Values{
		tsm1.NewFloatValue(time.Unix(1, 0).UnixNano(), 1.0),
		tsm1.NewFloatValue(time.Unix(2, 0).UnixNano(), 2.0),
	}
	if err := w.Write([]byte("cpu,host=a#!~#value"), values); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteIndex(); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	result := run(config{input: path, kind: "tsm", maxSamples: 5, maxFiles: 10})
	if result.Status != "OK" {
		t.Fatalf("status=%s errors=%v", result.Status, result.Errors)
	}
	if got := result.Input.AnalyzedCount; got != 1 {
		t.Fatalf("analyzed count=%d", got)
	}
	if len(result.Files) != 1 || result.Files[0].TSM == nil {
		t.Fatalf("missing tsm file report: %#v", result.Files)
	}
	if got := result.Files[0].TSM.KeyCount; got != 1 {
		t.Fatalf("key count=%d", got)
	}
	if got := result.Files[0].TSM.PointCount; got != 2 {
		t.Fatalf("point count=%d", got)
	}
}

func TestAnalyzeSeriesDirReadOnlySegments(t *testing.T) {
	root := t.TempDir()
	seriesDir := filepath.Join(root, tsdb.SeriesFileDirectory)
	partitionDir := filepath.Join(seriesDir, "00")
	if err := os.MkdirAll(partitionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	segmentPath := filepath.Join(partitionDir, "0000")
	segment, err := tsdb.CreateSeriesSegment(0, segmentPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := segment.InitForWrite(); err != nil {
		t.Fatal(err)
	}
	key := tsdb.AppendSeriesKey(nil, []byte("cpu"), models.NewTags(map[string]string{"host": "a"}))
	entry := tsdb.AppendSeriesEntry(nil, tsdb.SeriesEntryInsertFlag, 1, key)
	if _, err := segment.WriteLogEntry(entry); err != nil {
		t.Fatal(err)
	}
	if err := segment.Close(); err != nil {
		t.Fatal(err)
	}

	result := run(config{input: seriesDir, kind: "series", maxSamples: 5, maxFiles: 10})
	if result.Status != "OK" {
		t.Fatalf("status=%s errors=%v", result.Status, result.Errors)
	}
	if len(result.Files) != 1 || result.Files[0].Series == nil {
		t.Fatalf("missing series report: %#v", result.Files)
	}
	if got := result.Files[0].Series.SeriesCount; got != 1 {
		t.Fatalf("series count=%d", got)
	}
	if got := result.Files[0].Series.SampleSeries[0].Measurement; got != "cpu" {
		t.Fatalf("measurement=%q", got)
	}
}

func TestBadInputReturnsErrorReport(t *testing.T) {
	result := run(config{input: filepath.Join(t.TempDir(), "missing.tsm"), kind: "auto", maxSamples: 5, maxFiles: 10})
	if result.Status != "ERROR" {
		t.Fatalf("status=%s", result.Status)
	}
	if len(result.Errors) == 0 || len(result.Findings) == 0 {
		t.Fatalf("expected errors and findings: %#v", result)
	}
}
