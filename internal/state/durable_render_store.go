package state

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/levifig/loaf/internal/project"
)

// DurableRenderOptions describes a scratch durable render request.
type DurableRenderOptions struct {
	Kind   string
	Ref    string
	Branch string
}

// DurableRenderResult describes a rendered scratch durable document.
type DurableRenderResult struct {
	ContractVersion    int    `json:"contract_version,omitempty"`
	DatabaseScope      string `json:"database_scope,omitempty"`
	DatabasePath       string `json:"database_path,omitempty"`
	ProjectID          string `json:"project_id,omitempty"`
	ProjectName        string `json:"project_name,omitempty"`
	ProjectCurrentPath string `json:"project_current_path,omitempty"`
	Kind               string `json:"kind"`
	Ref                string `json:"ref"`
	Title              string `json:"title,omitempty"`
	Branch             string `json:"branch"`
	Path               string `json:"path"`
	ContentHash        string `json:"content_hash"`
	Contract           string `json:"contract"`
}

// RenderDurableArtifact writes a deterministic scratch render under XDG cache.
func RenderDurableArtifact(ctx context.Context, root project.Root, resolver PathResolver, options DurableRenderOptions) (DurableRenderResult, error) {
	store, err := openInitializedStore(root, resolver)
	if err != nil {
		return DurableRenderResult{}, err
	}
	defer store.Close()
	return store.RenderDurableArtifact(ctx, root, resolver, options)
}

// RenderDurableArtifact writes a deterministic scratch render using an open store.
func (s *Store) RenderDurableArtifact(ctx context.Context, root project.Root, resolver PathResolver, options DurableRenderOptions) (DurableRenderResult, error) {
	projectID, err := s.projectID(ctx, root)
	if err != nil {
		return DurableRenderResult{}, err
	}
	identity, err := s.projectIdentity(ctx, projectID)
	if err != nil {
		return DurableRenderResult{}, err
	}
	kind := strings.TrimSpace(options.Kind)
	ref := strings.TrimSpace(options.Ref)
	if ref == "" {
		return DurableRenderResult{}, fmt.Errorf("durable render requires a ref")
	}

	var doc DurableRenderDocument
	title := ""
	switch kind {
	case "spec":
		result, err := s.ShowSpec(ctx, root, ref)
		if err != nil {
			return DurableRenderResult{}, err
		}
		doc = DurableSpecRenderDocument(result.Spec)
		ref = firstNonEmpty(result.Spec.Alias, ref)
		title = result.Spec.Title
	case "report":
		result, err := s.ShowReport(ctx, root, ref)
		if err != nil {
			return DurableRenderResult{}, err
		}
		doc = DurableReportRenderDocument(result.Report)
		ref = firstNonEmpty(result.Report.Alias, ref)
		title = result.Report.Title
	default:
		return DurableRenderResult{}, fmt.Errorf("durable render kind %q is not supported", kind)
	}

	content, err := RenderDurableDocument(doc)
	if err != nil {
		return DurableRenderResult{}, err
	}
	branch := firstNonEmpty(strings.TrimSpace(options.Branch), ObservedGitBranch(root.Path()), "detached")
	path, err := durableRenderCacheFile(root, resolver, identity.ID, branch, kind, ref)
	if err != nil {
		return DurableRenderResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return DurableRenderResult{}, fmt.Errorf("create durable render cache directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return DurableRenderResult{}, fmt.Errorf("write durable render %s: %w", path, err)
	}

	return DurableRenderResult{
		ContractVersion:    StateJSONContractVersion,
		DatabaseScope:      identity.DatabaseScope,
		DatabasePath:       identity.DatabasePath,
		ProjectID:          identity.ID,
		ProjectName:        identity.FriendlyName,
		ProjectCurrentPath: identity.CurrentPath,
		Kind:               kind,
		Ref:                ref,
		Title:              title,
		Branch:             branch,
		Path:               path,
		ContentHash:        artifactBodyHash(content),
		Contract:           DurableRenderContract,
	}, nil
}

func durableRenderCacheFile(root project.Root, resolver PathResolver, projectID string, branch string, kind string, ref string) (string, error) {
	return resolver.RenderCachePath(
		root,
		safeDurableRenderSegment(projectID),
		safeDurableRenderSegment(branch),
		safeDurableRenderSegment(kind)+"-"+safeDurableRenderSegment(ref)+".md",
	)
}

func safeDurableRenderSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	cleaned := strings.Trim(b.String(), "-")
	if cleaned == "" {
		return "unknown"
	}
	return cleaned
}
