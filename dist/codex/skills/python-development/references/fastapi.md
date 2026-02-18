# FastAPI Development

## Contents
- FastAPI Stack
- Conventions
- Critical Rules

Conventions for building REST APIs with FastAPI.

## FastAPI Stack

| Component | Default |
|-----------|---------|
| Server | Uvicorn |
| Validation | Pydantic v2 |
| Auth | OAuth2/JWT |
| Testing | httpx (AsyncClient) |
| OpenAPI | Built-in (`/api/docs`) |

## Conventions

- **Route handlers**: Always `async def`
- **Response models**: Always specify `response_model` for type safety and OpenAPI docs
- **Input validation**: Pydantic models for request bodies; `Field()` for constraints
- **Dependency injection**: Use `Depends()` for shared logic (DB sessions, auth, settings)
- **Type annotations**: Use `Annotated[T, Depends(...)]` pattern (PEP 593)
- **Router organization**: Group by domain with `APIRouter(prefix="/api/v1/{resource}", tags=[...])`
- **Error handling**: Register `@app.exception_handler` for domain exceptions; never expose internal errors
- **Background tasks**: Use `BackgroundTasks` parameter for non-blocking post-response work
- **ORM boundary**: Never return raw database objects from endpoints; always map to response models

## Critical Rules

### Always
- Use `async def` for route handlers
- Define `response_model` for type safety
- Use dependency injection for shared logic
- Include proper status codes

### Never
- Block the event loop with sync I/O
- Return raw database objects
- Skip input validation
- Expose internal errors to clients
