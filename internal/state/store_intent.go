package state

import (
	"context"
	"fmt"
	"os"

	"github.com/levifig/loaf/internal/project"
)

// projectStoreIntent describes how a project-scoped command may open the
// global state database. Registration is intentionally not an ordinary store
// intent: only Initialize and the explicit project identity APIs may create or
// update a project-path mapping.
type projectStoreIntent uint8

const (
	projectStoreReadExisting projectStoreIntent = iota + 1
	projectStoreMutateExisting
	projectStoreObserveExisting
)

// openProjectStore opens an already-existing project store for one of the
// ordinary intents. Every intent requires a durable database file and resolves
// the current project path before returning. A lookup failure closes the store
// so unknown checkouts cannot leave an open connection behind.
func openProjectStore(ctx context.Context, root project.Root, resolver PathResolver, intent projectStoreIntent) (*Store, error) {
	databasePath, err := resolver.DatabasePath(root)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(databasePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("SQLite state database is not initialized; run `loaf state init` or `loaf state migrate markdown --apply` first")
	} else if err != nil {
		return nil, fmt.Errorf("inspect state database: %w", err)
	}

	var store *Store
	switch intent {
	case projectStoreReadExisting:
		store, err = OpenStoreReadOnly(databasePath)
		if err == nil {
			err = store.RequireCurrentSchema(ctx)
		}
	case projectStoreMutateExisting:
		store, err = OpenStore(databasePath)
		if err == nil {
			err = store.RequireCurrentSchema(ctx)
		}
	case projectStoreObserveExisting:
		store, err = OpenStore(databasePath)
		if err == nil {
			// A long write elsewhere must not stall the tool call; skipping one
			// best-effort observation is cheaper than waiting it out.
			_, err = store.db.ExecContext(ctx, `PRAGMA busy_timeout = 100`)
		}
		if err == nil {
			err = store.RequireCurrentSchema(ctx)
		}
	default:
		return nil, fmt.Errorf("unsupported project store intent %d", intent)
	}
	if err != nil {
		if store != nil {
			if closeErr := store.Close(); closeErr != nil {
				return nil, fmt.Errorf("%w; close state database: %v", err, closeErr)
			}
		}
		return nil, err
	}

	if _, err := store.LookupProjectIdentityForRoot(ctx, root); err != nil {
		if closeErr := store.Close(); closeErr != nil {
			return nil, fmt.Errorf("%w; close state database: %v", err, closeErr)
		}
		return nil, err
	}
	return store, nil
}

// OpenProjectStoreForObservation opens an existing project store for a
// best-effort, machine-local observation write, such as a pre-tool gateway
// count. It applies the ordinary open check but waits at most 100 ms for
// another writer: observations run on every tool call, and their callers
// ignore write failures, so skipping one is cheaper than stalling the call.
func OpenProjectStoreForObservation(ctx context.Context, root project.Root, resolver PathResolver) (*Store, error) {
	return openProjectStore(ctx, root, resolver, projectStoreObserveExisting)
}

func openProjectStoreReadExisting(ctx context.Context, root project.Root, resolver PathResolver) (*Store, error) {
	return openProjectStore(ctx, root, resolver, projectStoreReadExisting)
}

func openProjectStoreMutateExisting(ctx context.Context, root project.Root, resolver PathResolver) (*Store, error) {
	return openProjectStore(ctx, root, resolver, projectStoreMutateExisting)
}

// openInitializedStore is retained for non-journal callers while they migrate
// to explicit store intents. It now mutates only existing schema state and
// never registers or refreshes a project identity.
func openInitializedStore(root project.Root, resolver PathResolver) (*Store, error) {
	return openProjectStoreMutateExisting(context.Background(), root, resolver)
}
