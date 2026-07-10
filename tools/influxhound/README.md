# InfluxHound (v1)

InfluxHound is an offline startup bug analyzer for InfluxDB 1.x bug bundles.

## Usage

```bash
python -m tools.influxhound.cli <bundle_path>
```

Outputs:

- stdout: markdown report
- `analysis_output/events.json`
- `analysis_output/timeline.json`
- `analysis_output/diagnosis.json`

## Bundle layout

```text
bug_bundle/
├── bug.yaml
├── arch.yaml
├── runtime.yaml
├── semantic_event_catalog.yaml   (optional)
└── logs/
    └── influxd.log
```
