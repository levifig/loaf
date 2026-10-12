package vnextflowcontract

import (
	"io/fs"
	"os"
	"path"
	"strings"
	"testing"
	"testing/fstest"
)

// Flow's closed portable contract is a subset of the shared corpus. Target
// sidecars are checked by the CLI's sidecar-ownership and build-invariance tests;
// unrelated domain skills are not declarations in the Flow manifest.
func repositoryFlowContent(t *testing.T) fs.FS {
	t.Helper()
	source := os.DirFS("../../content")
	var manifest flowManifest
	if err := decodeStrictJSON(source, flowManifestPath, &manifest); err != nil {
		t.Fatal(err)
	}
	roots := []string{flowManifestPath, "agents/background-runner.md", "contracts"}
	for _, skill := range manifest.Skills {
		roots = append(roots, path.Dir(skill.Path))
	}
	entries, err := fs.ReadDir(source, "skills")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() && providerManifestExists(source, entry.Name()) {
			roots = append(roots, "skills/"+entry.Name())
		}
	}
	for _, template := range manifest.Templates {
		roots = append(roots, template.Path)
	}
	content := fstest.MapFS{}
	for _, root := range roots {
		err := fs.WalkDir(source, root, func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if strings.HasPrefix(entry.Name(), "SKILL.") && strings.HasSuffix(entry.Name(), ".yaml") {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			file := &fstest.MapFile{Mode: info.Mode()}
			if info.Mode().IsRegular() {
				file.Data, err = fs.ReadFile(source, name)
				if err != nil {
					return err
				}
			}
			content[name] = file
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return content
}
