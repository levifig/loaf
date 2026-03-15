# GridSight Data Architecture — Greenfield Design

## Context

GridSight (GDS) is an on-premises power systems monitoring platform deployed at customer sites. It computes thermal ratings (CIGRE TB 601, IEEE 738), dynamic line rating, network state estimation, capacity analysis, fault location, and vegetation management. The digital twin is a stateless computation engine — same logic operates on live, historical, forecast, or scenario data.

**Everything is greenfield.** The existing codebase, database, and APIs are disposable. This plan captures the architectural decisions from two council sessions and extensive user interviews.

### System Boundaries

```
EMP (cloud, source of truth)
  ├── Versioned model snapshots (topology, equipment, conductor specs)
  ├── Weather observations (real-time)
  └── Weather forecasts
        ↓ one-way entrypoint (S3-compatible object storage or MQTT)
GDS (on-premises, customer-controlled K8s namespace)
  ├── Model loader (receives + validates EMP snapshots)
  ├── Ingest service (separate pod, connects to customer's electrical data sources)
  ├── Core (FastAPI, computation engine, API)
  ├── PostgreSQL + TimescaleDB + PostGIS
  └── Export queue (local alterations → EMP)
```

## Decided: Data Domains

Two fundamental domains, per user's mental model:

1. **The Model** — topology, equipment, conductor specs, weather stations. Slowly changing. Versioned snapshots from EMP. Stored in `model` schema.
2. **The State** — measurements (electrical + weather), calculated results, equipment status events. Time-series. Stored in `ts` schema.

## Decided: Database Architecture

### One database, three schemas

```
gridsight (database)
├── model     — topology, equipment catalog/inventory, versioned snapshots
├── ts        — time-series: measurements, weather, calculated results
└── public    — extensions, migrations metadata, shared functions/types
```

Single-tenant deployment. Cross-schema joins work natively. One backup target.

### Model Versioning

- Full snapshots (not diffs). Model data is small (hundreds to low-thousands of entities).
- `model.model_version` table: id, version_label, received_at, source_hash, status (pending/active/superseded).
- Unique partial index enforces exactly one active version.
- Every model table has `model_version_id` FK.
- New version arrives → insert with status='pending' → validate → atomic flip to 'active'.
- Old versions retained for result traceability.

### Local Alterations (Customer Overrides)

Stored as an **overlay**, not direct mutation to EMP-sourced model:
- `model.local_alteration`: target_table, target_id, alteration_type, field_overrides (JSONB), created_by, status.
- EMP baseline remains pristine. Overlay applied at query time.
- On new model version: auto-carry-forward compatible changes, flag conflicts for review.
- Export path: serialize overlay as JSON, send to EMP.

## Decided: Measurement Schema

### Wide Row Format

One row per device per timestamp (not EAV). Natural key: `(ts, equipment_id)`.

### Fixed Columns (~27) + JSONB Overflow

User chose Option B (JSONB overflow for rare/manufacturer-specific measurements).

**Column set:**

```
-- Identity
ts, equipment_id

-- Voltage (phase-to-neutral, kV)
voltage_a_kv, voltage_b_kv, voltage_c_kv

-- Current (RMS magnitude, Amperes)
current_a_a, current_b_a, current_c_a

-- Power
active_power_mw, reactive_power_mvar, power_factor

-- Frequency
frequency_hz, rocof_hz_per_s

-- Phasor angles (PMU only, nullable for RTU)
voltage_angle_a_deg, voltage_angle_b_deg, voltage_angle_c_deg
current_angle_a_deg, current_angle_b_deg, current_angle_c_deg

-- Line-mounted sensor data
conductor_temp_c, sag_m, inclination_deg, vibration_mm_s

-- Quality
quality (enum: good/suspect/bad/missing), quality_flags (bitmask)

-- Overflow
extended (JSONB, NULL for 95%+ of rows)
```

**Compression:** segment_by=equipment_id, order_by=ts DESC, compress_after=2 hours.
**Retention:** 90 days (configurable per customer).
**Continuous aggregates:** 5-min and 1-hour rollups.

### Separate Tables

| Data | Table | Why Separate |
|------|-------|-------------|
| Weather observations | `ts.weather_observation` | Different devices, columns, cadence |
| Weather forecasts | `ts.weather_forecast` | Additional dimensions (issued_at, horizon) |
| TPM results | `ts.tpm_result` | Computed, per-span, model_version_id |
| Capacity results | `ts.capacity_result` | Computed, per-line |
| NSE results | `ts.nse_result` | Computed, per-bus |
| Equipment status | `ts.equipment_status_event` | Discrete events (breaker/tap), not continuous |
| Fault oscillography | Separate domain (future) | kHz, event-driven, COMTRADE |

## Decided: Equipment Identity

Three-level hierarchy:

### Level 1: Catalog (the type)
`model.equipment_catalog` — manufacturer, model, device_type, specs. Conductor-specific columns for thermal calcs (r_dc_20c, diameter, emissivity, etc.). JSONB `specs` for the long tail.

### Level 2: Instance (the specific device)
`model.equipment` — FK to catalog, serial_number, external_id (customer's identifier), calibration_data (JSONB overrides), communication_config.

### Level 3: Installation (where + when)
`model.equipment_installation` — FK to equipment, polymorphic location (installed_on_type + installed_on_id), installed_at, removed_at. Tracks device swaps over time.

**Inheritance:** Instance calibration_data overrides catalog specs. The computation engine resolves `equipment.calibration_data[key] ?? catalog.specs[key]`.

**Measurement linkage:** `electrical_measurement_raw.equipment_id` → `model.equipment.id` (the instance). Join to `equipment_installation` for location. Join to `equipment_catalog` for specs.

## Decided: Model Entities

| Entity | Key Properties |
|--------|---------------|
| Substation | name, location (PostGIS), voltage levels |
| Bus | parent substation, nominal voltage |
| Line | name, from_bus, to_bus, voltage, conductor catalog ref |
| Tower | parent line, sequence number, location, height, type |
| Span | parent line, from_tower, to_tower, length, geometry (LineString), ruling span |
| Weather Station | location, elevation, source |

All model tables carry `model_version_id` FK.

## Decided: Ingest Service

### Architecture
- Separate K8s pod, same namespace as GDS core.
- Python (stack consistency, I/O-bound workload, shared Pydantic models).
- Plugin/adapter pattern: `customer.yaml` selects adapter (mqtt_iec61850, rest_json, csv_sftp, opc_ua, modbus_tcp) + field mapping + unit conversions.
- Different customers get different config via K8s ConfigMap. Same Docker image.

### Interface: Internal REST API

Ingest pod POSTs normalized JSON to GDS core's `/api/v1/ingest/electrical-readings` endpoint. Core handles validation, equipment resolution, and DB writes. Ingest is a pure protocol adapter with no domain logic.

Rationale: DB may be on different namespace/server. Cleaner boundary. Single audit/validation point. Ingest pod needs no DB credentials.

### Equipment Resolution
- Handled by GDS core, not the ingest pod.
- Core maintains in-memory cache of `external_id → equipment_id` mapping.
- Refresh via PostgreSQL LISTEN/NOTIFY on model version change.

## Decided: EMP-to-GDS Transport

S3-compatible object storage (MinIO on-prem) with structured paths:

```
gds-inbound/
├── models/v{version}/          — Parquet files + manifest.json
├── weather/observations/       — Time-partitioned Parquet
├── weather/forecasts/          — Timestamped Parquet
└── commands/                   — Operational commands
```

GDS pulls via scheduled sync (outbound HTTPS, fits corporate firewalls). Weather observations may use a lighter REST polling endpoint for near-real-time.

## Decided: Stateless Digital Twin

- Computation signature: `f(model, state, patches) → results`.
- Same engine operates on live, historical, forecast, or scenario data.
- `data_source` discriminator on measurement tables for bulk data origin: `'live'`, `'synthetic:demo-{name}'`, `'import:{source}'`.
- Results store `model_version_id` and optional `scenario_id` for traceability.

## Decided: Scenarios as Patches (Not Datasets)

Scenarios are **overlay definitions** applied at computation time, not pre-built datasets.

### Scenario Definition

```
model.scenario
  id, name, description, created_by, created_at
  base_source: 'live' | 'historical' | 'scenario:{id}'
  base_time_range: tstzrange (for historical base)
  state_patches: JSONB[]     -- override measurement/weather values
  model_patches: JSONB[]     -- override topology/equipment
  is_template: boolean       -- reusable event template (tree fall, fire, etc.)
  template_params: JSONB     -- parameterized fields (location, severity, etc.)
```

### Patch Types

| Type | Example | What It Patches |
|------|---------|----------------|
| Value override | "current at IED-042 = 800A" | State (measurement values) |
| Weather override | "wind = 200km/h in 5km radius of [lat,lon]" | State (weather values) |
| Event template | "tree fall at span-17" | Both model (clearance) and state (fault current) |
| Equipment change | "swap conductor on spans 12-15 to HTLS" | Model (conductor catalog ref) |
| Topology change | "open breaker B3" | Model (equipment status) |

### Computation Flow

```python
def compute_with_scenario(scenario, model, live_state):
    effective_model = apply_patches(model, scenario.model_patches)
    effective_state = apply_patches(live_state, scenario.state_patches)
    return compute_thermal_rating(effective_model, effective_state)
```

The engine doesn't know it's running a scenario. Same code, different inputs.

### Results

Result hypertables gain a nullable `scenario_id` column:
- `NULL` → live computation
- `UUID` → scenario computation, FK to `model.scenario`

Enables side-by-side comparison: live vs scenario results for the same span/time.

### Event Templates

Predefined scenarios with `is_template = true`. UI offers "Apply template → select location → creates scenario." Templates define abstract patches; location is filled in at application time.

### Synthetic Data (Demos/Tests)

Fully synthetic datasets stored in `electrical_measurement_raw` with `data_source = 'synthetic:demo-{name}'`. This is the only case where scenario data lives in the measurement table. Used for demos, internal testing, and integration tests — not for customer-facing scenario analysis.

## Key Domain Insight

CIGRE TB 601 / IEEE 738 thermal rating only needs **RMS current magnitude** from electrical measurements. Everything else is weather data and conductor properties. Voltage, power, angles, and frequency serve situational awareness, state estimation, and capacity analysis — not the core DLR calculation.

## Not Building Yet

- Message broker (Kafka/RabbitMQ) between ingest and core
- Multi-tenant partitioning
- Custom EMP↔GDS sync protocol
- Fault location oscillography storage (design when feature is specced)
- Event sourcing for model changes

## Next Steps

All major architectural decisions are resolved. To move forward:

1. **Formalize as ADR(s) in GDS project** — capture these decisions in `docs/decisions/` with the council reasoning
2. **Write concrete DDL** — create the greenfield schema (model + ts schemas, all tables, hypertables, compression, retention)
3. **Write SQLAlchemy models** — map the DDL to Python models with Pydantic schemas for the API layer
4. **Build ingest service** — plugin/adapter architecture with first adapter (likely MQTT), REST API interface to core
5. **Design EMP model export format** — Parquet + manifest structure for model snapshots
6. **Build scenario engine** — patch application logic, event templates, scenario management API

## Deliverables (for GDS project, not Loaf)

This plan will be used to create artifacts in the GDS codebase:
- ADRs in `docs/decisions/`
- DDL migrations
- SQLAlchemy models
- Ingest service scaffolding
- Scenario engine design
