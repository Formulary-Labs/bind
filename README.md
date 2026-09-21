# bind

Give it a `MappingDocument`. Get a resolved mapping table — which source controls correspond to which target controls, which ones have no match, and which source IDs are missing from the catalog.

```bash
go install github.com/Formulary-Labs/bind/cmd/bind@latest
```

Or download a pre-built binary from the [releases page](https://github.com/Formulary-Labs/bind/releases) for `linux/amd64`, `darwin/arm64`, `darwin/amd64`, or `windows/amd64`.

## What it does

`bind` reads a gemara `MappingDocument` (Layer 1 artifact) and resolves the cross-framework control mappings it defines. Pass in source and target `ControlCatalog` YAMLs to get human-readable titles alongside every control ID and to catch IDs that don't exist in the catalog.

Without catalogs, `bind` works purely from the `MappingDocument` — IDs are used directly.

## Usage

```bash
bind --document <mapping.yaml> [flags]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--document` | — | Path to gemara `MappingDocument` YAML **(required)** |
| `--source-catalog` | `""` | Path to source `ControlCatalog` YAML — enables title resolution and source-ID validation |
| `--target-catalog` | `""` | Path to target `ControlCatalog` YAML — enables title resolution |
| `--format` | `json` | Output format: `json`, `md`, or `csv` |
| `--filter` | `""` | Filter output: `mapped`, `unmapped`, or empty for all |
| `--program` | `""` | Program slug for provenance logging |
| `--version` | — | Print version and exit |

### Examples

```bash
# Resolve a mapping document — IDs only
bind --document mappings/iso27001-to-nist800-53.yaml

# Resolve with title resolution and source-ID validation
bind --document mappings/iso27001-to-nist800-53.yaml \
     --source-catalog catalogs/iso27001.yaml \
     --target-catalog catalogs/nist800-53.yaml

# Markdown table — readable in terminal or audit package
bind --document mappings/iso42001-to-iso27001.yaml --format md

# CSV export of unmapped controls only
bind --document mappings/iso42001-to-iso27001.yaml --filter unmapped --format csv
```

## Output

### JSON

```json
{
  "source": "iso27001",
  "target": "nist-sp-800-53-rev-5",
  "total": 114,
  "mapped": 108,
  "unmapped": 6,
  "entries": [
    {
      "source_id": "A.5.1",
      "source_title": "Policies for information security",
      "relationship": "subset-of",
      "targets": [
        {
          "target_id": "PM-1",
          "target_title": "Information Security Program Plan",
          "strength": 8,
          "rationale": "A.5.1 requires documented policy; PM-1 requires a documented program plan."
        }
      ]
    }
  ],
  "flags": []
}
```

`flags` lists data quality issues — source control IDs that appear in the `MappingDocument` but are absent from the source catalog.

### Markdown

```
# Cross-Framework Mapping: iso27001 → nist-sp-800-53-rev-5

**Total:** 114 | **Mapped:** 108 | **Unmapped:** 6

| Source | Relationship | Target Controls | Strength |
|---|---|---|---|
| A.5.1 — Policies for information security | subset-of | PM-1 — Information Security Program Plan | 8/10 |
```

### CSV

```
source_id,source_title,relationship,target_id,target_title,strength,rationale
A.5.1,Policies for information security,subset-of,PM-1,Information Security Program Plan,8,...
```

## Relationships

`bind` preserves the `relationship` value from the `MappingDocument` as-is. Typical values:

| Relationship | Meaning |
|---|---|
| `subset-of` | Source control is a subset of the target |
| `superset-of` | Source control is a superset of the target |
| `equivalent` | Controls are functionally equivalent |
| `related` | Partial overlap; not a direct substitution |
| `no-match` | No corresponding target control |

`no-match` entries count as unmapped and appear in `--filter unmapped` output.

## Data quality flags

When a source catalog is provided, `bind` validates that every source control ID in the `MappingDocument` exists in the catalog. Unknown IDs are flagged:

```json
"flags": [
  "[CONFLICT — VERIFY] source control \"A.99.1\" not found in source catalog \"iso27001\""
]
```

Flags appear in JSON and Markdown output. They do not cause a non-zero exit — they are data quality observations, not errors.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Mapping resolved successfully |
| `2` | Tool error — missing `--document`, unreadable file, or internal failure |

## Catalog acquisition

`bind` requires gemara `ControlCatalog` and `MappingDocument` YAML files. See [CATALOGS.md](https://github.com/Formulary-Labs/.github/blob/main/CATALOGS.md) for known catalog sources.

## Pipeline context

`bind` answers "what covers what" across frameworks. Use it when a program operates under multiple frameworks or when you need to understand control overlap before building a combined SOA.

```bash
# Identify unmapped gaps before building a cross-framework SOA
bind --document mappings/iso42001-to-iso27001.yaml \
     --source-catalog catalogs/iso42001.yaml \
     --target-catalog catalogs/iso27001.yaml \
     --filter unmapped --format csv > unmapped-gaps.csv

# Feed unmapped gaps into specimen as risks
specimen add --source coverage_gap --input unmapped-gaps.csv
```

See [CONTRIBUTING.md](https://github.com/Formulary-Labs/.github/blob/main/CONTRIBUTING.md) to contribute.

## License

Apache License 2.0
