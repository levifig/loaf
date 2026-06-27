package state

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateSpecStoresBodyAndAllocatesAlias(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	created, err := CreateSpec(context.Background(), root, PathResolver{StateHome: stateHome}, SpecCreateOptions{
		Slug:    "token-rotation",
		Title:   "Token Rotation",
		Body:    "# Token Rotation\n\nRotate tokens nightly.",
		SetBody: true,
	})
	if err != nil {
		t.Fatalf("CreateSpec() error = %v", err)
	}
	if created.Spec.Alias != "SPEC-001" || created.Spec.Title != "Token Rotation" || created.Spec.Status != "draft" || created.EventID == "" {
		t.Fatalf("created = %#v, want draft SPEC-001 metadata", created)
	}

	show, err := ShowSpec(context.Background(), root, PathResolver{StateHome: stateHome}, "SPEC-001")
	if err != nil {
		t.Fatalf("ShowSpec() error = %v", err)
	}
	if show.Spec.Body != "# Token Rotation\n\nRotate tokens nightly." {
		t.Fatalf("Body = %q, want byte-exact spec body", show.Spec.Body)
	}
	if !show.Spec.HasBody {
		t.Fatalf("HasBody = false, want true")
	}
	if show.Spec.Title != "Token Rotation" || show.Spec.Status != "draft" || show.Spec.Alias != "SPEC-001" {
		t.Fatalf("Spec = %#v, want created spec metadata", show.Spec)
	}

	specs, err := ListSpecs(context.Background(), root, PathResolver{StateHome: stateHome})
	if err != nil {
		t.Fatalf("ListSpecs() error = %v", err)
	}
	if _, ok := specs.Specs["SPEC-001"]; !ok {
		t.Fatalf("ListSpecs() = %#v, want SPEC-001 present", specs.Specs)
	}
}

func TestCreateSpecAutoAllocatesAcrossSQLiteAndFiles(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	first, err := CreateSpec(context.Background(), root, PathResolver{StateHome: stateHome}, SpecCreateOptions{Slug: "one"})
	if err != nil {
		t.Fatalf("CreateSpec(one) error = %v", err)
	}
	if first.Spec.Alias != "SPEC-001" {
		t.Fatalf("first alias = %q, want SPEC-001", first.Spec.Alias)
	}

	// A render file on disk for a higher id must lift the allocation watermark.
	specsDir := filepath.Join(root.Path(), ".agents", "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "SPEC-014-legacy.md"), []byte("legacy"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	next, err := CreateSpec(context.Background(), root, PathResolver{StateHome: stateHome}, SpecCreateOptions{Slug: "two"})
	if err != nil {
		t.Fatalf("CreateSpec(two) error = %v", err)
	}
	if next.Spec.Alias != "SPEC-015" {
		t.Fatalf("next alias = %q, want SPEC-015 (max of SQLite SPEC-001 and file SPEC-014, +1)", next.Spec.Alias)
	}
}

func TestCreateSpecExplicitIDAndDuplicateRejection(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	created, err := CreateSpec(context.Background(), root, PathResolver{StateHome: stateHome}, SpecCreateOptions{Slug: "explicit", ID: "SPEC-042"})
	if err != nil {
		t.Fatalf("CreateSpec(explicit) error = %v", err)
	}
	if created.Spec.Alias != "SPEC-042" {
		t.Fatalf("alias = %q, want SPEC-042", created.Spec.Alias)
	}

	if _, err := CreateSpec(context.Background(), root, PathResolver{StateHome: stateHome}, SpecCreateOptions{Slug: "dup", ID: "SPEC-042"}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateSpec(duplicate) error = %v, want already exists", err)
	}

	if _, err := CreateSpec(context.Background(), root, PathResolver{StateHome: stateHome}, SpecCreateOptions{Slug: "bad", ID: "SPEC-XYZ"}); err == nil || !strings.Contains(err.Error(), "invalid spec id") {
		t.Fatalf("CreateSpec(invalid id) error = %v, want invalid spec id", err)
	}
}

func TestCreateSpecFinalizeRendersSlugFilename(t *testing.T) {
	root := projectRoot(t)
	stateHome := t.TempDir()
	if _, err := Initialize(context.Background(), root, PathResolver{StateHome: stateHome}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	created, err := CreateSpec(context.Background(), root, PathResolver{StateHome: stateHome}, SpecCreateOptions{
		Slug:    "render-target",
		Title:   "Render Target",
		Body:    "Body for render.",
		SetBody: true,
	})
	if err != nil {
		t.Fatalf("CreateSpec() error = %v", err)
	}

	render, err := FinalizeDurableArtifact(context.Background(), root, PathResolver{StateHome: stateHome}, DurableFinalizeOptions{
		Kind: "spec",
		Ref:  created.Spec.Alias,
	})
	if err != nil {
		t.Fatalf("FinalizeDurableArtifact() error = %v", err)
	}
	wantRel := ".agents/specs/SPEC-001-render-target.md"
	if render.RelativePath != wantRel {
		t.Fatalf("RelativePath = %q, want %q", render.RelativePath, wantRel)
	}
	content, err := os.ReadFile(render.Path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", render.Path, err)
	}
	if !strings.Contains(string(content), "Body for render.") {
		t.Fatalf("rendered content missing body:\n%s", content)
	}
}
