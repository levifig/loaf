# Go Concurrency Reference

## Contents
- Project Conventions
- Channel Types
- Preferred Patterns
- sync Package Usage
- Always/Never

Goroutine, channel, and synchronization conventions.

## Project Conventions

| Decision | Convention |
|----------|-----------|
| Coordination | Channels for communication; `sync.Mutex` for simple shared state |
| Cancellation | Always use `context.Context` |
| Goroutine lifecycle | Document ownership; never fire-and-forget |
| Buffered channels | Use when producer/consumer rates differ; unbuffered for synchronization |

## Channel Types

| Type | Declaration | Blocking |
|------|-------------|----------|
| Unbuffered | `make(chan int)` | Send blocks until receive |
| Buffered | `make(chan int, 10)` | Send blocks when full |

Use directional parameters: `chan<-` (send-only), `<-chan` (receive-only).

## Preferred Patterns

| Pattern | Use When |
|---------|----------|
| Worker pool | Fixed concurrency over a job queue |
| Semaphore (`chan struct{}`) | Limiting concurrent access to a resource |
| Select with context | Timeout and cancellation in concurrent operations |
| WaitGroup | Waiting for a known set of goroutines to finish |

## sync Package Usage

| Type | Use Case |
|------|----------|
| `sync.Mutex` | Protect shared state (prefer over channels for simple counters) |
| `sync.WaitGroup` | Wait for goroutines to complete |
| `sync.Once` | One-time initialization |

## Always/Never

| Always | Never |
|--------|-------|
| Close channels from sender | Close from receiver |
| Use context for cancellation | Let goroutines leak |
| Range over channels to detect close | Forget to close when done sending |
| Capture loop variables in closures | Share loop variable across goroutines |
| Use sync.Mutex for simple state | Use channels for simple counters |
| Document goroutine ownership | Create goroutines without clear lifecycle |
