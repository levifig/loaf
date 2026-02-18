# Python Data Engineering

## Contents
- Stack Decisions
- ETL Pipeline Pattern
- Critical Rules

Modern data processing conventions.

## Stack Decisions

| Component | Choice | Why |
|-----------|--------|-----|
| DataFrames | Polars | Lazy evaluation, Rust performance, no GIL |
| Serialization | Parquet | Columnar, compressed, schema-preserving |
| Validation | Pydantic | Type-safe schema definition |

Prefer Polars over Pandas for all new projects.

## ETL Pipeline Pattern

Use Protocol-based sources and sinks for composable pipelines:

- `DataSource` protocol: `def read(self) -> pl.LazyFrame`
- `DataSink` protocol: `def write(self, df: pl.DataFrame) -> None`
- `ETLPipeline`: chains source, N transforms, sink; calls `collect()` at sink boundary

Key conventions:
- Always use lazy evaluation (`scan_*`) for large datasets
- Define schemas explicitly (Polars schema dict or Pydantic model)
- Use `read_csv_batched` / `sink_parquet` for streaming large files

## Critical Rules

### Always
- Use lazy evaluation (scan_*) for large datasets
- Define schemas explicitly
- Use streaming for memory efficiency
- Prefer Polars over Pandas for new projects

### Never
- Load entire dataset into memory unnecessarily
- Skip schema validation
- Use `iterrows()` (use `iter_rows()`)
- Forget to `collect()` lazy frames
