---
name: power-systems-modeling
description: >-
  Covers thermal rating models (CIGRE TB 601, IEEE 738), conductor physics, and
  electrical properties.
---

# Power Systems Reference

Follows [foundations principles](../foundations/SKILL.md).

Domain knowledge for overhead transmission line physics, thermal ratings, and mechanical analysis.

## Key Reference Files

| File | Content |
|------|---------|
| `references/thermal-models.md` | CIGRE TB 601, IEEE 738 thermal rating implementations |
| `references/conductor-limits.md` | Physical validation bounds and parameter constraints |
| `references/electrical-properties.md` | Resistance, sag, catenary formulas and calculations |
| `references/standards-reference.md` | Industry standards summary (CIGRE, IEEE, IEC, EN) |

## Available Scripts

| Script | Usage | Description |
|--------|-------|-------------|
| `scripts/validate-bounds.py` | `validate-bounds.py -t conductor_temp -v 85` | Validate physics values against bounds |
| `scripts/convert-units.py` | `convert-units.py 25 C K` | Convert between units (temp, length, power, speed) |
| `scripts/check-standard-refs.sh` | `check-standard-refs.sh <dir>` | Check for proper CIGRE/IEEE citations in code |

## Default Standard

**CIGRE TB 601** is the default thermal rating standard. Document any deviations in code comments. IEEE 738 differs in treatment of low wind speeds.

## Physical Bounds

| Parameter | Valid Range |
|-----------|-------------|
| Conductor temperature | -40°C to 250°C |
| Ambient temperature | -50°C to 60°C |
| Wind speed | 0 to 50 m/s |
| Solar radiation | 0 to 1400 W/m² |

Always validate physical parameters at function boundaries. Raise `ValueError` for out-of-range values.

## Unit Conventions

- **Internal**: SI units, temperatures in Kelvin
- **Display**: Celsius for temperatures
- **Document**: Always include units in variable names or docstrings

## Standard Citations

Always cite standard section numbers in code comments:

```python
# CIGRE TB 601, Section 4.2.3: Natural convection heat loss
# Formula: P_n = pi * D * lambda * Nu * (T_s - T_a)
```

## Testing Tolerances

Thermal calculations: use `pytest.approx()` with `rel=1e-3` (0.1% accuracy).

## Related Skills

- `database-patterns` — For persisting physics results
- [foundations/code-style](../foundations/references/code-style.md) — For Python conventions in physics code
