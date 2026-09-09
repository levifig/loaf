package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerCheckDockerfileValidatesBestPracticesWithoutInterpreter(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	good := "FROM golang:1.22 AS build\nFROM alpine:3.20\nUSER 1000\nHEALTHCHECK CMD true\nENV PYTHONUNBUFFERED=1\nRUN apt-get install -y curl && rm -rf /var/lib/apt/lists\n"
	writeFile(t, filepath.Join(root, "Dockerfile"), good)
	if err := (Runner{WorkingDir: root}).Run([]string{"check", "dockerfile"}); err != nil {
		t.Fatalf("good default Dockerfile: %v", err)
	}

	bad := "FROM ubuntu:latest\nRUN apt-get install -y python\nENV " + "PASS" + "WORD" + "=hunter2\n"
	writeFile(t, filepath.Join(root, "bad.Dockerfile"), bad)
	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "dockerfile", "bad.Dockerfile", "--json"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("bad: %v", err)
	}
	var output checkOperatorJSON
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(output.Findings, "\n")
	for _, want := range []string{"multi-stage", "USER", "HEALTHCHECK", ":latest", "apt-get", "PYTHONUNBUFFERED", "assignment leak"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("findings=%#v, want %q", output.Findings, want)
		}
	}
}

func TestRunnerCheckDockerfileFailsOnMissingOrUnreadable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	err := (Runner{WorkingDir: root}).Run([]string{"check", "dockerfile", "missing"})
	if err == nil || !strings.Contains(err.Error(), "unreadable Dockerfile") {
		t.Fatalf("missing: %v", err)
	}

	skipWithoutEnforcedPermissions(t)
	path := filepath.Join(root, "locked")
	writeFile(t, path, "FROM alpine:3.20\n")
	chmodForTest(t, path, 0o000)
	err = (Runner{WorkingDir: root}).Run([]string{"check", "dockerfile", "locked"})
	if err == nil || !strings.Contains(err.Error(), "unreadable Dockerfile") {
		t.Fatalf("unreadable: %v", err)
	}
}

func TestRunnerCheckK8sManifestRegexProbesWithoutYAMLParser(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	good := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  template:
    spec:
      runAsNonRoot: true
      containers:
        - name: app
          image: ghcr.io/example/web:1.2.3
          resources:
            requests:
              cpu: 10m
            limits:
              cpu: 100m
          livenessProbe:
            httpGet:
              path: /health
          readinessProbe:
            httpGet:
              path: /ready
          securityContext:
            allowPrivilegeEscalation: false
`
	writeFile(t, filepath.Join(root, "good.yaml"), good)
	if err := (Runner{WorkingDir: root}).Run([]string{"check", "k8s-manifest", "good.yaml"}); err != nil {
		t.Fatalf("good: %v", err)
	}

	bad := `
kind: Deployment
metadata:
  name: web
spec:
  containers:
        - name: app
          image: nginx
---
kind: Secret
metadata:
  name: db
stringData:
  ` + "DB_" + "PASS" + "WORD" + `: hunter2
`
	writeFile(t, filepath.Join(root, "bad.yaml"), bad)
	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "k8s-manifest", "bad.yaml", "--json"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("bad: %v", err)
	}
	var output checkOperatorJSON
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(output.Findings, "\n")
	for _, want := range []string{"runAsNonRoot", "Using :latest or no tag", "allowPrivilegeEscalation", "stringData"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("findings=%#v, want %q", output.Findings, want)
		}
	}
}

func TestRunnerCheckK8sManifestFailsOnMissingOrUnreadable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	err := (Runner{WorkingDir: root}).Run([]string{"check", "k8s-manifest", "missing.yaml"})
	if err == nil || !strings.Contains(err.Error(), "unreadable manifest") {
		t.Fatalf("missing: %v", err)
	}
	skipWithoutEnforcedPermissions(t)
	path := filepath.Join(root, "locked.yaml")
	writeFile(t, path, "kind: Pod\n")
	chmodForTest(t, path, 0o000)
	err = (Runner{WorkingDir: root}).Run([]string{"check", "k8s-manifest", "locked.yaml"})
	if err == nil || !strings.Contains(err.Error(), "unreadable manifest") {
		t.Fatalf("unreadable: %v", err)
	}
}

func TestRunnerCheckK8sManifestEmptyDocumentsPass(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	writeFile(t, filepath.Join(root, "empty.yaml"), "\n---\n\n")
	if err := (Runner{WorkingDir: root}).Run([]string{"check", "k8s-manifest", "empty.yaml"}); err != nil {
		t.Fatalf("empty docs: %v", err)
	}
}

func TestRunnerCheckDockerfileDefaultMissingIsUnreadable(t *testing.T) {
	err := (Runner{WorkingDir: t.TempDir()}).Run([]string{"check", "dockerfile"})
	if err == nil {
		t.Fatal("missing default Dockerfile passed")
	}
}
