package devtool

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PackageOptions drive Package.
type PackageOptions struct {
	RootDir string
	Env     Env
	Stdout  io.Writer
}

// Package writes dist/release/loaf_<version>_<target>.tar.gz for every
// requested target plus checksums.txt, the assets GitHub Releases, Homebrew,
// and install.sh consume. Each archive is a complete distribution: the native
// binary at bin/<name>, the manifest, config, content, the authored vnext
// Flow content, the built dist and plugin trees, and the Claude Code
// marketplace manifest.
func Package(options PackageOptions) error {
	root := options.RootDir
	stdout := options.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	version, err := manifestVersion(root)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(root, "vnext", "content", "skills")); err != nil {
		return fmt.Errorf("missing tracker-native Flow content at vnext/content/skills.")
	}
	targets, err := ReleaseTargetsFromEnv(options.Env)
	if err != nil {
		return err
	}
	outDir := filepath.Join(root, "dist", "release")
	if err := os.RemoveAll(outDir); err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	var checksums []string
	for _, target := range targets {
		nativeSource := filepath.Join(root, "bin", "native", target.RuntimeID, target.BinaryName())
		if _, err := os.Stat(nativeSource); err != nil {
			return fmt.Errorf("missing native binary for %s: %s\nRun `make release` before packaging release archives.", target.RuntimeID, nativeSource)
		}
		packageName := "loaf_" + version + "_" + target.RuntimeID
		archivePath := filepath.Join(outDir, packageName+".tar.gz")
		digest, err := writeReleaseArchive(root, archivePath, packageName, version, target.RuntimeID, nativeSource, target.BinaryName())
		if err != nil {
			return err
		}
		checksums = append(checksums, digest+"  "+packageName+".tar.gz")
		fmt.Fprintf(stdout, "✓ Packaged %s.tar.gz\n", packageName)
	}
	checksumsPath := filepath.Join(outDir, "checksums.txt")
	if err := os.WriteFile(checksumsPath, []byte(strings.Join(checksums, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "✓ Wrote %s\n", checksumsPath)
	return nil
}

func manifestVersion(root string) (string, error) {
	body, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return "", fmt.Errorf("read package.json: %w", err)
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil {
		return "", fmt.Errorf("parse package.json: %w", err)
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return "", fmt.Errorf("package.json missing version.")
	}
	return manifest.Version, nil
}

// archiveEntries are the distribution parts, in the order they are written.
var archiveFiles = []string{"package.json", "README.md", "CHANGELOG.md"}
var archiveDirs = []string{"config", "content", "vnext/content", "dist", "plugins", ".claude-plugin"}

type releaseManifest struct {
	SchemaVersion  int                   `json:"schema_version"`
	PackageVersion string                `json:"package_version"`
	Target         string                `json:"target"`
	Files          []releaseManifestFile `json:"files"`
}

type releaseManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Mode   int64  `json:"mode"`
}

func buildReleaseManifest(root, version, target, nativeSource, binaryName string) ([]byte, error) {
	files := []releaseManifestFile{}
	add := func(source, archivePath string, mode int64) error {
		info, err := os.Lstat(source)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("release manifest source %s is not a regular file", source)
		}
		file, err := os.Open(source)
		if err != nil {
			return err
		}
		hasher := sha256.New()
		_, copyErr := io.Copy(hasher, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		files = append(files, releaseManifestFile{
			Path:   archivePath,
			SHA256: fmt.Sprintf("%x", hasher.Sum(nil)),
			Mode:   mode,
		})
		return nil
	}
	if err := add(nativeSource, "bin/"+binaryName, 0o755); err != nil {
		return nil, err
	}

	ampRoot := filepath.Join(root, "dist", "amp")
	info, err := os.Lstat(ampRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("missing Amp distribution at dist/amp")
		}
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("Amp distribution dist/amp is not a directory")
	}
	err = filepath.WalkDir(ampRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("Amp distribution entry %s is not a regular file", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		mode := int64(0o644)
		if info.Mode().Perm()&0o111 != 0 {
			mode = 0o755
		}
		return add(path, filepath.ToSlash(rel), mode)
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	manifest := releaseManifest{
		SchemaVersion:  1,
		PackageVersion: version,
		Target:         target,
		Files:          files,
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

func writeReleaseArchive(root, archivePath, packageName, version, target, nativeSource, binaryName string) (string, error) {
	manifest, err := buildReleaseManifest(root, version, target, nativeSource, binaryName)
	if err != nil {
		return "", err
	}
	file, err := os.Create(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	gz := gzip.NewWriter(io.MultiWriter(file, hasher))
	tw := tar.NewWriter(gz)

	addFile := func(source, name string, mode int64) error {
		info, err := os.Lstat(source)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("release archive source %s is not a regular file", source)
		}
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: info.Size(), ModTime: info.ModTime(), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		src, err := os.Open(source)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(tw, src)
		return err
	}
	addBytes := func(body []byte, name string, mode int64) error {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		_, err := tw.Write(body)
		return err
	}
	if err := addFile(nativeSource, packageName+"/bin/"+binaryName, 0o755); err != nil {
		return "", err
	}
	if err := addBytes(manifest, packageName+"/loaf-release-manifest.json", 0o644); err != nil {
		return "", err
	}
	for _, rel := range archiveFiles {
		source := filepath.Join(root, rel)
		if _, err := os.Stat(source); err != nil {
			continue
		}
		if err := addFile(source, packageName+"/"+rel, 0o644); err != nil {
			return "", err
		}
	}
	for _, dir := range archiveDirs {
		source := filepath.Join(root, filepath.FromSlash(dir))
		if _, err := os.Stat(source); err != nil {
			continue
		}
		var paths []string
		err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				// The release output itself never nests into an archive.
				if dir == "dist" && path == filepath.Join(source, "release") {
					return filepath.SkipDir
				}
				return nil
			}
			ampPrefix := filepath.Join(source, "amp") + string(os.PathSeparator)
			if entry.Name() == ".DS_Store" && (dir != "dist" || !strings.HasPrefix(path, ampPrefix)) {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
		if err != nil {
			return "", err
		}
		sort.Strings(paths)
		for _, path := range paths {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return "", err
			}
			info, err := os.Lstat(path)
			if err != nil {
				return "", err
			}
			mode := int64(0o644)
			if info.Mode().Perm()&0o111 != 0 {
				mode = 0o755
			}
			if err := addFile(path, packageName+"/"+filepath.ToSlash(rel), mode); err != nil {
				return "", err
			}
		}
	}
	if err := tw.Close(); err != nil {
		return "", err
	}
	if err := gz.Close(); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
