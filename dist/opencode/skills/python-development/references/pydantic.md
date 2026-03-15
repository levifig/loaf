# Pydantic Data Validation

## Contents
- Conventions
- Settings Pattern
- Critical Rules

Type-safe data validation and settings management with Pydantic v2.

## Conventions

- **Model config**: Always set `str_strip_whitespace`, `validate_assignment`, `from_attributes` on models used with ORMs
- **Field constraints**: Use `Field()` for validation (min/max length, gt/lt, pattern)
- **Specialized types**: Use `EmailStr`, `HttpUrl`, `SecretStr` over raw strings
- **Validators**: `@field_validator` for single-field, `@model_validator(mode="after")` for cross-field
- **Serialization**: Use `model_dump()` (not `.dict()`), `exclude=True` on sensitive fields
- **Computed fields**: Use `@computed_field` + `@property` for derived values
- **v2 only**: Never mix v1 patterns (`.dict()`, `@validator`, `Config` inner class)

## Settings Pattern

Standard env-based configuration using pydantic-settings:

```python
from pydantic_settings import BaseSettings, SettingsConfigDict

class Settings(BaseSettings):
    app_name: str = "My Application"
    debug: bool = False
    database_url: str = Field(..., validation_alias="DATABASE_URL")
    api_key: str = Field(..., validation_alias="API_KEY")

    model_config = SettingsConfigDict(
        env_file=".env",
        case_sensitive=False,
        extra="ignore",
    )
```

Cache with `@lru_cache` on getter function.

## Critical Rules

### Always
- Use `Field()` for constraints and metadata
- Use `computed_field` for calculated properties
- Use proper Pydantic types (`EmailStr`, `HttpUrl`)
- Use `model_dump()` instead of `.dict()`

### Never
- Bypass validation with `model_construct()`
- Use mutable defaults (use `default_factory`)
- Store sensitive data without `exclude=True`
- Mix Pydantic v1 and v2 patterns
