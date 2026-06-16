# influxdb_storage_analyzer

`influxdb_storage_analyzer` is a read-only JSON CLI for inspecting InfluxDB 1.x
storage artifacts. It is intentionally separate from `influx_inspect` command
registration so existing InfluxDB commands and release builds are not changed.

## Build

From the InfluxDB repository root:

```bash
PKG_CONFIG=/abs/path/to/pkg-config.sh go build -o influxdb_storage_analyzer ./cmd/influxdb_storage_analyzer
```

The module baseline is Go 1.26. Some local development environments pin
`GOROOT`; when using Go toolchain auto download, ensure `PATH` and `GOROOT`
point to the same Go version.

## CLI

```bash
influxdb_storage_analyzer \
  -input /path/to/file-or-dir \
  -kind auto \
  -max-samples 10 \
  -max-files 200 \
  -max-file-bytes 0
```

Flags:

- `-input`: required TSM file, TSI `.tsi` file, `_series` directory, or a
  directory to scan.
- `-kind`: `auto`, `tsm`, `tsi`, or `series`.
- `-max-samples`: caps sample keys, measurements, and series in JSON output.
- `-max-files`: caps discovered files when scanning a directory.
- `-max-file-bytes`: skips regular files above this size; `0` disables the
  limit.

## Output Protocol

Stdout is always JSON:

```json
{
  "schemaVersion": 1,
  "tool": "influxdb_storage_analyzer",
  "status": "OK",
  "summary": "analyzed 1 storage file(s) successfully",
  "findings": [],
  "files": []
}
```

Top-level `summary` and `findings` are compatible with LogAgent Tool Runner's
generic JSON stdout parser. Corrupt or unsupported inputs produce
`status=ERROR`, high-severity findings, and a non-zero exit code.

## Capabilities

- TSM: key count, block count, point count, time/key range, type counts,
  tombstone summary, block size stats, and key samples.
- TSI `.tsi`: measurement count, tag key/value counts, series id set
  cardinality, sketch estimates, and measurement samples.
- Series file: read-only `_series` segment scan using `SeriesSegment.Open`;
  does not call `SeriesFile.Open` because that API may create or initialize
  writable files.

The tool does not decode arbitrary values, mutate storage files, rebuild
indexes, compact data, or call network services.
