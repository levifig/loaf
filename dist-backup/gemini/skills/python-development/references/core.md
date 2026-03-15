# Python Core

## Contents
- Python Stack
- Project Structure
- Dependency Management with uv
- pyproject.toml Conventions
- Python-specific Rules

Foundation for modern Python 3.12+ development.

## Python Stack

| Component | Default | Alternative |
|-----------|---------|-------------|
| Runtime | Python 3.12+ | - |
| Package Manager | uv | rye, poetry |
| Linter/Formatter | ruff | black + flake8 |
| Type Checker | mypy (strict) | pyright |
| Testing | pytest | unittest |

## Project Structure

Use `src/` layout: `src/{project}/` for source, `tests/` at root, `pyproject.toml` + `uv.lock` at root. Standard subdirectories: `models/`, `services/`, `utils/`.

## Dependency Management with uv

```bash
uv init my-project --python 3.12
uv add fastapi uvicorn pydantic-settings
uv add --dev pytest ruff mypy
uv run python -m my_project
uv sync
```

## pyproject.toml Conventions

Key tool settings (add to every project):

```toml
[tool.ruff]
target-version = "py312"
line-length = 100
select = ["E", "F", "I", "N", "W", "UP"]

[tool.mypy]
python_version = "3.12"
strict = true

[tool.pytest.ini_options]
testpaths = ["tests"]
addopts = "--cov=src --cov-report=term-missing"
```

Import order (enforced by ruff `I` rule): standard library, third-party, local.

## Python-specific Rules

### Always
- Use `pathlib.Path` for file operations (not `os.path`)
- Use f-strings for string formatting
- Use context managers (`with`) for resources
- Use `|` for type unions: `str | None` (not `Optional[str]`)

### Never
- Use mutable default arguments (`def foo(items=[])`)
- Use `from module import *`
- Mix tabs and spaces
