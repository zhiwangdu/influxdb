// Command influxdb_storage_analyzer emits a read-only JSON summary for InfluxDB
// 1.x storage files. It intentionally lives outside existing influx_inspect
// command registration so normal InfluxDB builds and CLI behavior are unchanged.
package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/influxdata/influxdb/models"
	"github.com/influxdata/influxdb/tsdb"
	"github.com/influxdata/influxdb/tsdb/engine/tsm1"
	"github.com/influxdata/influxdb/tsdb/index/tsi1"
)

const toolID = "influxdb_storage_analyzer"

type config struct {
	input        string
	kind         string
	seriesFile   string
	maxSamples   int
	maxFiles     int
	maxFileBytes int64
}

type report struct {
	SchemaVersion int            `json:"schemaVersion"`
	Tool          string         `json:"tool"`
	Status        string         `json:"status"`
	Summary       string         `json:"summary"`
	Input         inputSummary   `json:"input"`
	Files         []fileReport   `json:"files"`
	Findings      []finding      `json:"findings"`
	Errors        []errorSummary `json:"errors,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
}

type inputSummary struct {
	Path          string `json:"path"`
	RequestedKind string `json:"requestedKind"`
	FileCount     int    `json:"fileCount"`
	AnalyzedCount int    `json:"analyzedCount"`
	SkippedCount  int    `json:"skippedCount"`
	MaxSamples    int    `json:"maxSamples"`
}

type fileReport struct {
	Path      string        `json:"path"`
	Kind      string        `json:"kind"`
	SizeBytes int64         `json:"sizeBytes"`
	Status    string        `json:"status"`
	Summary   string        `json:"summary"`
	TSM       *tsmReport    `json:"tsm,omitempty"`
	TSI       *tsiReport    `json:"tsi,omitempty"`
	Series    *seriesReport `json:"series,omitempty"`
	Error     string        `json:"error,omitempty"`
}

type tsmReport struct {
	KeyCount         int              `json:"keyCount"`
	BlockCount       int              `json:"blockCount"`
	PointCount       int64            `json:"pointCount"`
	IndexSizeBytes   uint32           `json:"indexSizeBytes"`
	TimeRange        *timeRange       `json:"timeRange,omitempty"`
	KeyRange         *keyRange        `json:"keyRange,omitempty"`
	TypeCounts       map[string]int   `json:"typeCounts"`
	BlockSizeStats   sizeStats        `json:"blockSizeStats"`
	HasTombstones    bool             `json:"hasTombstones"`
	TombstoneStats   tombstoneSummary `json:"tombstoneStats"`
	SampleKeys       []tsmKeySample   `json:"sampleKeys,omitempty"`
	DecodeErrorCount int              `json:"decodeErrorCount"`
}

type timeRange struct {
	MinUnixNano int64  `json:"minUnixNano"`
	MaxUnixNano int64  `json:"maxUnixNano"`
	Min         string `json:"min"`
	Max         string `json:"max"`
}

type keyRange struct {
	Min string `json:"min"`
	Max string `json:"max"`
}

type sizeStats struct {
	MinBytes int64   `json:"minBytes"`
	MaxBytes int64   `json:"maxBytes"`
	AvgBytes float64 `json:"avgBytes"`
}

type tombstoneSummary struct {
	Exists       bool  `json:"exists"`
	FileSize     int64 `json:"fileSize"`
	LastModified int64 `json:"lastModified"`
}

type tsmKeySample struct {
	Key         string `json:"key"`
	SeriesKey   string `json:"seriesKey,omitempty"`
	Field       string `json:"field,omitempty"`
	Type        string `json:"type"`
	BlockCount  int    `json:"blockCount"`
	MinUnixNano int64  `json:"minUnixNano"`
	MaxUnixNano int64  `json:"maxUnixNano"`
}

type tsiReport struct {
	Level                           int                    `json:"level"`
	ID                              int                    `json:"id"`
	MeasurementCount                uint64                 `json:"measurementCount"`
	MeasurementSeriesReferences     uint64                 `json:"measurementSeriesReferences"`
	MeasurementSeriesDataBytes      uint64                 `json:"measurementSeriesDataBytes"`
	TagKeyCount                     uint64                 `json:"tagKeyCount"`
	TagValueCount                   uint64                 `json:"tagValueCount"`
	TagValueSeriesReferences        uint64                 `json:"tagValueSeriesReferences"`
	TagValueSeriesDataBytes         uint64                 `json:"tagValueSeriesDataBytes"`
	SeriesIDSetCardinality          uint64                 `json:"seriesIdSetCardinality"`
	TombstoneSeriesIDSetCardinality uint64                 `json:"tombstoneSeriesIdSetCardinality"`
	SeriesSketchEstimate            uint64                 `json:"seriesSketchEstimate"`
	TombstoneSeriesSketchEstimate   uint64                 `json:"tombstoneSeriesSketchEstimate"`
	SampleMeasurements              []tsiMeasurementSample `json:"sampleMeasurements,omitempty"`
}

type tsiMeasurementSample struct {
	Name        string `json:"name"`
	Deleted     bool   `json:"deleted"`
	SeriesRefs  uint64 `json:"seriesRefs"`
	TagKeyCount int    `json:"tagKeyCount"`
}

type seriesReport struct {
	PartitionCount    int               `json:"partitionCount"`
	SegmentCount      int               `json:"segmentCount"`
	SeriesCount       uint64            `json:"seriesCount"`
	TombstoneCount    uint64            `json:"tombstoneCount"`
	InvalidEntryCount int               `json:"invalidEntryCount"`
	MinSeriesID       uint64            `json:"minSeriesId,omitempty"`
	MaxSeriesID       uint64            `json:"maxSeriesId,omitempty"`
	Partitions        []seriesPartition `json:"partitions"`
	SampleSeries      []seriesSample    `json:"sampleSeries,omitempty"`
}

type seriesPartition struct {
	ID             int    `json:"id"`
	SegmentCount   int    `json:"segmentCount"`
	SeriesCount    uint64 `json:"seriesCount"`
	TombstoneCount uint64 `json:"tombstoneCount"`
	SizeBytes      int64  `json:"sizeBytes"`
}

type seriesSample struct {
	ID          uint64            `json:"id"`
	Measurement string            `json:"measurement"`
	Tags        map[string]string `json:"tags,omitempty"`
	Key         string            `json:"key"`
}

type finding struct {
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
	Message  string `json:"message"`
}

type errorSummary struct {
	File    string `json:"file,omitempty"`
	Message string `json:"message"`
}

type candidate struct {
	path string
	kind string
}

func main() {
	cfg, err := parseFlags(os.Args[1:])
	if err != nil {
		writeErrorReport(cfg, err)
		os.Exit(2)
	}

	r := run(cfg)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(r); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode report: %v\n", err)
		os.Exit(2)
	}
	if r.Status == "ERROR" {
		os.Exit(1)
	}
}

func parseFlags(args []string) (config, error) {
	cfg := config{kind: "auto", maxSamples: 10, maxFiles: 200}
	fs := flag.NewFlagSet(toolID, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.input, "input", "", "TSM/TSI file, _series directory, or directory to scan")
	fs.StringVar(&cfg.kind, "kind", "auto", "input kind: auto, tsm, tsi, or series")
	fs.StringVar(&cfg.seriesFile, "series-file", "", "optional _series directory for future TSI context")
	fs.IntVar(&cfg.maxSamples, "max-samples", 10, "maximum sample keys/measurements/series to include")
	fs.IntVar(&cfg.maxFiles, "max-files", 200, "maximum discovered files/directories to analyze")
	fs.Int64Var(&cfg.maxFileBytes, "max-file-bytes", 0, "skip regular files larger than this many bytes; 0 disables the limit")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	cfg.kind = strings.ToLower(strings.TrimSpace(cfg.kind))
	if cfg.input == "" {
		return cfg, errors.New("-input is required")
	}
	switch cfg.kind {
	case "auto", "tsm", "tsi", "series":
	default:
		return cfg, fmt.Errorf("unsupported -kind %q", cfg.kind)
	}
	if cfg.maxSamples < 0 {
		return cfg, errors.New("-max-samples must be non-negative")
	}
	if cfg.maxFiles <= 0 {
		return cfg, errors.New("-max-files must be positive")
	}
	return cfg, nil
}

func run(cfg config) report {
	r := report{
		SchemaVersion: 1,
		Tool:          toolID,
		Status:        "OK",
		Input: inputSummary{
			Path:          cfg.input,
			RequestedKind: cfg.kind,
			MaxSamples:    cfg.maxSamples,
		},
		CreatedAt: time.Now().UTC(),
	}

	candidates, findings, err := discoverCandidates(cfg)
	r.Findings = append(r.Findings, findings...)
	if err != nil {
		r.Errors = append(r.Errors, errorSummary{Message: err.Error()})
		r.Findings = append(r.Findings, finding{Severity: "high", Message: err.Error()})
		r.Status = "ERROR"
		r.Summary = "input discovery failed"
		return r
	}
	if len(candidates) > cfg.maxFiles {
		r.Findings = append(r.Findings, finding{
			Severity: "medium",
			Message:  fmt.Sprintf("discovered %d candidates, analyzing first %d due to max-files", len(candidates), cfg.maxFiles),
		})
		candidates = candidates[:cfg.maxFiles]
	}
	r.Input.FileCount = len(candidates)

	for _, c := range candidates {
		fr := analyzeCandidate(c, cfg)
		if fr.Status == "OK" {
			r.Input.AnalyzedCount++
		} else {
			r.Input.SkippedCount++
			r.Errors = append(r.Errors, errorSummary{File: fr.Path, Message: fr.Error})
			r.Findings = append(r.Findings, finding{Severity: "high", File: fr.Path, Message: fr.Error})
		}
		r.Files = append(r.Files, fr)
	}

	if r.Input.AnalyzedCount == 0 {
		r.Status = "ERROR"
		r.Summary = fmt.Sprintf("no %s storage file could be analyzed", cfg.kind)
	} else if len(r.Errors) > 0 || hasHighFinding(r.Findings) {
		r.Status = "WARNING"
		r.Summary = fmt.Sprintf("analyzed %d storage file(s) with %d finding(s) and %d error(s)", r.Input.AnalyzedCount, len(r.Findings), len(r.Errors))
	} else {
		r.Status = "OK"
		r.Summary = fmt.Sprintf("analyzed %d storage file(s) successfully", r.Input.AnalyzedCount)
	}
	return r
}

func writeErrorReport(cfg config, err error) {
	r := report{
		SchemaVersion: 1,
		Tool:          toolID,
		Status:        "ERROR",
		Summary:       err.Error(),
		Input: inputSummary{
			Path:          cfg.input,
			RequestedKind: cfg.kind,
			MaxSamples:    cfg.maxSamples,
		},
		Findings:  []finding{{Severity: "high", Message: err.Error()}},
		Errors:    []errorSummary{{Message: err.Error()}},
		CreatedAt: time.Now().UTC(),
	}
	_ = json.NewEncoder(os.Stdout).Encode(r)
}

func discoverCandidates(cfg config) ([]candidate, []finding, error) {
	info, err := os.Stat(cfg.input)
	if err != nil {
		return nil, nil, err
	}
	if !info.IsDir() {
		kind, err := detectFileKind(cfg.input, cfg.kind)
		if err != nil {
			return nil, []finding{{Severity: "high", File: cfg.input, Message: err.Error()}}, nil
		}
		return []candidate{{path: cfg.input, kind: kind}}, nil, nil
	}
	if cfg.kind == "series" || looksLikeSeriesDir(cfg.input) {
		return []candidate{{path: cfg.input, kind: "series"}}, nil, nil
	}

	var out []candidate
	var findings []finding
	err = filepath.WalkDir(cfg.input, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			findings = append(findings, finding{Severity: "high", File: path, Message: walkErr.Error()})
			return nil
		}
		if path != cfg.input && d.IsDir() && (cfg.kind == "auto" || cfg.kind == "series") && looksLikeSeriesDir(path) {
			out = append(out, candidate{path: path, kind: "series"})
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		kind, err := detectFileKind(path, cfg.kind)
		if err == nil {
			out = append(out, candidate{path: path, kind: kind})
		}
		return nil
	})
	return out, findings, err
}

func detectFileKind(path, requested string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if requested == "tsm" {
		if ext != ".tsm" {
			return "", fmt.Errorf("expected .tsm input, got %q", filepath.Base(path))
		}
		return "tsm", nil
	}
	if requested == "tsi" {
		if ext != tsi1.IndexFileExt {
			return "", fmt.Errorf("expected .tsi input, got %q", filepath.Base(path))
		}
		return "tsi", nil
	}
	if requested == "series" {
		return "", fmt.Errorf("series input must be a _series directory")
	}
	switch ext {
	case ".tsm":
		return "tsm", nil
	case tsi1.IndexFileExt:
		return "tsi", nil
	default:
		if hasTSMHeader(path) {
			return "tsm", nil
		}
		if hasTSIHeader(path) {
			return "tsi", nil
		}
		return "", fmt.Errorf("unsupported storage file suffix %q", ext)
	}
}

func hasTSMHeader(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var buf [5]byte
	if _, err := io.ReadFull(f, buf[:]); err != nil {
		return false
	}
	return binary.BigEndian.Uint32(buf[0:4]) == tsm1.MagicNumber && buf[4] == tsm1.Version
}

func hasTSIHeader(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var buf [4]byte
	if _, err := io.ReadFull(f, buf[:]); err != nil {
		return false
	}
	return string(buf[:]) == tsi1.FileSignature
}

func looksLikeSeriesDir(path string) bool {
	if filepath.Base(path) != tsdb.SeriesFileDirectory {
		return false
	}
	for i := 0; i < tsdb.SeriesFilePartitionN; i++ {
		if _, err := os.Stat(filepath.Join(path, fmt.Sprintf("%02x", i))); err == nil {
			return true
		}
	}
	return false
}

func analyzeCandidate(c candidate, cfg config) fileReport {
	info, err := os.Stat(c.path)
	if err != nil {
		return fileReport{Path: c.path, Kind: c.kind, Status: "ERROR", Error: err.Error()}
	}
	size := info.Size()
	if !info.IsDir() && cfg.maxFileBytes > 0 && size > cfg.maxFileBytes {
		return fileReport{
			Path: c.path, Kind: c.kind, SizeBytes: size, Status: "ERROR",
			Error: fmt.Sprintf("file size %d exceeds max-file-bytes %d", size, cfg.maxFileBytes),
		}
	}
	switch c.kind {
	case "tsm":
		return analyzeTSM(c.path, size, cfg.maxSamples)
	case "tsi":
		return analyzeTSI(c.path, size, cfg.maxSamples)
	case "series":
		return analyzeSeriesDir(c.path, cfg.maxSamples)
	default:
		return fileReport{Path: c.path, Kind: c.kind, Status: "ERROR", Error: "unknown candidate kind"}
	}
}

func analyzeTSM(path string, size int64, maxSamples int) fileReport {
	fr := fileReport{Path: path, Kind: "tsm", SizeBytes: size}
	f, err := os.Open(path)
	if err != nil {
		return fileError(fr, err)
	}
	defer f.Close()
	reader, err := tsm1.NewTSMReader(f)
	if err != nil {
		return fileError(fr, err)
	}
	defer reader.Close()

	minTime, maxTime := reader.TimeRange()
	minKey, maxKey := reader.KeyRange()
	tr := &timeRange{MinUnixNano: minTime, MaxUnixNano: maxTime, Min: unixNanoString(minTime), Max: unixNanoString(maxTime)}
	kr := &keyRange{Min: string(minKey), Max: string(maxKey)}
	stats := reader.TombstoneStats()
	out := &tsmReport{
		KeyCount:       reader.KeyCount(),
		IndexSizeBytes: reader.IndexSize(),
		TimeRange:      tr,
		KeyRange:       kr,
		TypeCounts:     make(map[string]int),
		HasTombstones:  reader.HasTombstones(),
		TombstoneStats: tombstoneSummary{
			Exists:       stats.TombstoneExists,
			FileSize:     int64(stats.Size),
			LastModified: stats.LastModified,
		},
	}

	var sampleCount int
	var totalBlockBytes int64
	for i := 0; i < reader.KeyCount(); i++ {
		key, typ := reader.KeyAt(i)
		entries := reader.Entries(key)
		out.TypeCounts[blockTypeName(typ)]++
		out.BlockCount += len(entries)
		for _, entry := range entries {
			if out.BlockSizeStats.MinBytes == 0 || int64(entry.Size) < out.BlockSizeStats.MinBytes {
				out.BlockSizeStats.MinBytes = int64(entry.Size)
			}
			if int64(entry.Size) > out.BlockSizeStats.MaxBytes {
				out.BlockSizeStats.MaxBytes = int64(entry.Size)
			}
			totalBlockBytes += int64(entry.Size)
			_, block, err := reader.ReadBytes(&entry, nil)
			if err != nil {
				out.DecodeErrorCount++
				continue
			}
			n, err := tsm1.BlockCount(block)
			if err != nil {
				out.DecodeErrorCount++
				continue
			}
			out.PointCount += int64(n)
		}
		if sampleCount < maxSamples {
			seriesKey, field := tsm1.SeriesAndFieldFromCompositeKey(key)
			sample := tsmKeySample{
				Key:        string(key),
				SeriesKey:  string(seriesKey),
				Field:      string(field),
				Type:       blockTypeName(typ),
				BlockCount: len(entries),
			}
			if len(entries) > 0 {
				sample.MinUnixNano = entries[0].MinTime
				sample.MaxUnixNano = entries[len(entries)-1].MaxTime
			}
			out.SampleKeys = append(out.SampleKeys, sample)
			sampleCount++
		}
	}
	if out.BlockCount > 0 {
		out.BlockSizeStats.AvgBytes = float64(totalBlockBytes) / float64(out.BlockCount)
	}
	fr.TSM = out
	fr.Status = "OK"
	fr.Summary = fmt.Sprintf("TSM keys=%d blocks=%d points=%d tombstones=%v", out.KeyCount, out.BlockCount, out.PointCount, out.HasTombstones)
	return fr
}

func analyzeTSI(path string, size int64, maxSamples int) fileReport {
	fr := fileReport{Path: path, Kind: "tsi", SizeBytes: size}
	idx := tsi1.NewIndexFile(nil)
	idx.SetPath(path)
	if err := idx.Open(); err != nil {
		return fileError(fr, err)
	}
	defer idx.Close()

	out := &tsiReport{Level: idx.Level(), ID: idx.ID()}
	if ss, err := idx.SeriesIDSet(); err == nil {
		out.SeriesIDSetCardinality = ss.Cardinality()
	}
	if ss, err := idx.TombstoneSeriesIDSet(); err == nil {
		out.TombstoneSeriesIDSetCardinality = ss.Cardinality()
	}
	if sketch, tombstoneSketch, err := idx.SeriesSketches(); err == nil {
		out.SeriesSketchEstimate = sketch.Count()
		out.TombstoneSeriesSketchEstimate = tombstoneSketch.Count()
	}
	if itr := idx.MeasurementIterator(); itr != nil {
		for me, _ := itr.Next().(*tsi1.MeasurementBlockElem); me != nil; me, _ = itr.Next().(*tsi1.MeasurementBlockElem) {
			out.MeasurementCount++
			out.MeasurementSeriesReferences += me.SeriesN()
			out.MeasurementSeriesDataBytes += uint64(len(me.SeriesData()))
			tagKeys := 0
			if kitr := idx.TagKeyIterator(me.Name()); kitr != nil {
				for ke, _ := kitr.Next().(*tsi1.TagBlockKeyElem); ke != nil; ke, _ = kitr.Next().(*tsi1.TagBlockKeyElem) {
					out.TagKeyCount++
					tagKeys++
					if vitr := idx.TagValueIterator(me.Name(), ke.Key()); vitr != nil {
						for ve, _ := vitr.Next().(*tsi1.TagBlockValueElem); ve != nil; ve, _ = vitr.Next().(*tsi1.TagBlockValueElem) {
							out.TagValueCount++
							out.TagValueSeriesReferences += uint64(ve.SeriesN())
							out.TagValueSeriesDataBytes += uint64(len(ve.SeriesData()))
						}
					}
				}
			}
			if len(out.SampleMeasurements) < maxSamples {
				out.SampleMeasurements = append(out.SampleMeasurements, tsiMeasurementSample{
					Name:        string(me.Name()),
					Deleted:     me.Deleted(),
					SeriesRefs:  me.SeriesN(),
					TagKeyCount: tagKeys,
				})
			}
		}
	}
	fr.TSI = out
	fr.Status = "OK"
	fr.Summary = fmt.Sprintf("TSI measurements=%d tagKeys=%d tagValues=%d seriesIdSet=%d", out.MeasurementCount, out.TagKeyCount, out.TagValueCount, out.SeriesIDSetCardinality)
	return fr
}

func analyzeSeriesDir(path string, maxSamples int) fileReport {
	fr := fileReport{Path: path, Kind: "series"}
	out := &seriesReport{}
	for i := 0; i < tsdb.SeriesFilePartitionN; i++ {
		partitionPath := filepath.Join(path, fmt.Sprintf("%02x", i))
		info, err := os.Stat(partitionPath)
		if err != nil || !info.IsDir() {
			continue
		}
		out.PartitionCount++
		part := seriesPartition{ID: i}
		entries, err := os.ReadDir(partitionPath)
		if err != nil {
			return fileError(fr, fmt.Errorf("read partition %s: %w", partitionPath, err))
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			segmentID, err := tsdb.ParseSeriesSegmentFilename(entry.Name())
			if err != nil {
				continue
			}
			segmentPath := filepath.Join(partitionPath, entry.Name())
			segmentInfo, err := os.Stat(segmentPath)
			if err == nil {
				part.SizeBytes += segmentInfo.Size()
				fr.SizeBytes += segmentInfo.Size()
			}
			segment := tsdb.NewSeriesSegment(segmentID, segmentPath)
			if err := segment.Open(); err != nil {
				return fileError(fr, fmt.Errorf("open series segment %s: %w", segmentPath, err))
			}
			err = segment.ForEachEntry(func(flag uint8, id uint64, _ int64, key []byte) error {
				switch flag {
				case tsdb.SeriesEntryInsertFlag:
					part.SeriesCount++
					out.SeriesCount++
					if out.MinSeriesID == 0 || id < out.MinSeriesID {
						out.MinSeriesID = id
					}
					if id > out.MaxSeriesID {
						out.MaxSeriesID = id
					}
					if len(out.SampleSeries) < maxSamples {
						name, tags := tsdb.ParseSeriesKey(key)
						out.SampleSeries = append(out.SampleSeries, seriesSample{
							ID:          id,
							Measurement: string(name),
							Tags:        tagsToMap(tags),
							Key:         string(models.MakeKey(name, tags)),
						})
					}
				case tsdb.SeriesEntryTombstoneFlag:
					part.TombstoneCount++
					out.TombstoneCount++
				default:
					out.InvalidEntryCount++
				}
				return nil
			})
			closeErr := segment.Close()
			if err != nil {
				return fileError(fr, fmt.Errorf("read series segment %s: %w", segmentPath, err))
			}
			if closeErr != nil {
				return fileError(fr, fmt.Errorf("close series segment %s: %w", segmentPath, closeErr))
			}
			part.SegmentCount++
			out.SegmentCount++
		}
		out.Partitions = append(out.Partitions, part)
	}
	sort.Slice(out.Partitions, func(i, j int) bool { return out.Partitions[i].ID < out.Partitions[j].ID })
	if out.SegmentCount == 0 {
		return fileError(fr, errors.New("no series segment files found"))
	}
	fr.Series = out
	fr.Status = "OK"
	fr.Summary = fmt.Sprintf("series partitions=%d segments=%d series=%d tombstones=%d", out.PartitionCount, out.SegmentCount, out.SeriesCount, out.TombstoneCount)
	return fr
}

func fileError(fr fileReport, err error) fileReport {
	fr.Status = "ERROR"
	fr.Error = err.Error()
	fr.Summary = "analysis failed"
	return fr
}

func hasHighFinding(findings []finding) bool {
	for _, f := range findings {
		if f.Severity == "high" {
			return true
		}
	}
	return false
}

func blockTypeName(typ byte) string {
	switch typ {
	case tsm1.BlockFloat64:
		return "float"
	case tsm1.BlockInteger:
		return "integer"
	case tsm1.BlockUnsigned:
		return "unsigned"
	case tsm1.BlockBoolean:
		return "boolean"
	case tsm1.BlockString:
		return "string"
	default:
		return fmt.Sprintf("unknown_%d", typ)
	}
}

func unixNanoString(v int64) string {
	if v == 0 {
		return ""
	}
	return time.Unix(0, v).UTC().Format(time.RFC3339Nano)
}

func tagsToMap(tags models.Tags) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for _, tag := range tags {
		out[string(tag.Key)] = string(tag.Value)
	}
	return out
}
