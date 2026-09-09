package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRunnerCheckBoundsBoundariesAndUnitsWithoutInterpreter(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	cases := []struct {
		name string
		args []string
		fail bool
		want string
	}{
		{name: "conductor min", args: []string{"--type", "conductor_temp", "--value", "-40"}, want: "within valid range"},
		{name: "conductor max", args: []string{"-t", "conductor_temp", "-v", "250"}, want: "within valid range"},
		{name: "conductor below", args: []string{"-t", "conductor_temp", "-v", "-40.1"}, fail: true, want: "below minimum"},
		{name: "conductor above", args: []string{"-t", "conductor_temp", "-v", "250.1"}, fail: true, want: "exceeds maximum"},
		{name: "kelvin inside", args: []string{"-t", "conductor_temp", "-v", "300", "-u", "K"}, want: "26.85"},
		{name: "kelvin below", args: []string{"-t", "conductor_temp", "-v", "200", "-u", "K"}, fail: true, want: "below minimum"},
		{name: "ambient max", args: []string{"-t", "ambient_temp", "-v", "60"}, want: "within valid range"},
		{name: "current zero", args: []string{"-t", "current", "-v", "0"}, want: "within valid range"},
		{name: "current negative", args: []string{"-t", "current", "-v", "-0.1"}, fail: true, want: "below minimum"},
		{name: "current huge", args: []string{"-t", "current", "-v", "1e20"}, want: "within valid range"},
		{name: "flux max", args: []string{"-t", "flux_density", "-v", "2"}, want: "within valid range"},
		{name: "flux above", args: []string{"-t", "flux_density", "-v", "2.0001"}, fail: true, want: "exceeds maximum"},
		{name: "unknown type", args: []string{"-t", "humidity", "-v", "1"}, fail: true, want: "Unknown parameter type"},
		{name: "unsupported fahrenheit", args: []string{"-t", "conductor_temp", "-v", "70", "-u", "F"}, fail: true, want: "unsupported unit"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			args := append([]string{"check", "bounds"}, tc.args...)
			args = append(args, "--json")
			err := (Runner{Stdout: &stdout, WorkingDir: root}).Run(args)
			var output checkOperatorJSON
			if unmarshalErr := json.Unmarshal(stdout.Bytes(), &output); unmarshalErr != nil && stdout.Len() > 0 {
				t.Fatalf("json: %v stdout=%s", unmarshalErr, stdout.String())
			}
			if tc.fail {
				var exitErr ExitError
				if !errors.As(err, &exitErr) || exitErr.Code != 1 {
					t.Fatalf("err=%v stdout=%s, want fail", err, stdout.String())
				}
				joined := strings.Join(append(output.Errors, output.Findings...), "\n")
				if !strings.Contains(joined, tc.want) {
					t.Fatalf("output=%#v, want %q", output, tc.want)
				}
				return
			}
			if err != nil {
				t.Fatalf("err=%v stdout=%s", err, stdout.String())
			}
			if !strings.Contains(strings.Join(output.Findings, "\n"), tc.want) {
				t.Fatalf("findings=%#v, want %q", output.Findings, tc.want)
			}
		})
	}
}

func TestRunnerCheckBoundsJSONFileAndMalformedInputs(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	writeFile(t, filepath.Join(root, "ok.json"), `{"conductor_temp": 85, "ambient_temp": {"value": 300, "unit": "K"}}`+"\n")
	if err := (Runner{WorkingDir: root}).Run([]string{"check", "bounds", "--check-all", "ok.json"}); err != nil {
		t.Fatalf("ok json: %v", err)
	}
	writeFile(t, filepath.Join(root, "bad.json"), `{"conductor_temp": 900}`+"\n")
	err := (Runner{WorkingDir: root}).Run([]string{"check", "bounds", "--check-all", "bad.json"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("out of range json: %v", err)
	}
	err = (Runner{WorkingDir: root}).Run([]string{"check", "bounds", "--check-all", "missing.json"})
	if err == nil || !strings.Contains(err.Error(), "unreadable bounds file") {
		t.Fatalf("missing: %v", err)
	}
	writeFile(t, filepath.Join(root, "broken.json"), `{"conductor_temp":`)
	err = (Runner{WorkingDir: root}).Run([]string{"check", "bounds", "--check-all", "broken.json"})
	if err == nil || !strings.Contains(err.Error(), "malformed bounds JSON") {
		t.Fatalf("broken: %v", err)
	}
	writeFile(t, filepath.Join(root, "array.json"), `[]`)
	err = (Runner{WorkingDir: root}).Run([]string{"check", "bounds", "--check-all", "array.json"})
	if err == nil || !strings.Contains(err.Error(), "malformed bounds JSON") {
		t.Fatalf("array: %v", err)
	}
	writeFile(t, filepath.Join(root, "nonstr.json"), `{"conductor_temp": {"unit": "C"}}`)
	err = (Runner{WorkingDir: root}).Run([]string{"check", "bounds", "--check-all", "nonstr.json"})
	if err == nil || !strings.Contains(err.Error(), "malformed bounds JSON") {
		t.Fatalf("missing value: %v", err)
	}
	err = (Runner{WorkingDir: root}).Run([]string{"check", "bounds", "--value", "hot", "--type", "conductor_temp"})
	if err == nil || !strings.Contains(err.Error(), "malformed --value") {
		t.Fatalf("non-numeric value: %v", err)
	}
	for _, tc := range []struct {
		name    string
		content string
		want    string
		coded   bool
	}{
		{name: "null field", content: "{\"conductor_temp\":null}\n", want: "malformed bounds JSON field conductor_temp"},
		{name: "document null", content: "null\n", want: "expected a non-null object"},
		{name: "null value object", content: "{\"conductor_temp\":{\"value\":null,\"unit\":\"C\"}}\n", want: "value is not a number"},
		{name: "null unit", content: "{\"conductor_temp\":{\"value\":85,\"unit\":null}}\n", want: "unit is not a string"},
		{name: "numeric unit", content: "{\"conductor_temp\":{\"value\":85,\"unit\":1}}\n", want: "unit is not a string"},
		{name: "unsupported unit", content: "{\"conductor_temp\":{\"value\":85,\"unit\":\"F\"}}\n", want: "unsupported unit", coded: true},
		{name: "trailing delimiter", content: "{\"conductor_temp\":85}}\n", want: "malformed bounds JSON"},
		{name: "trailing content", content: "{\"conductor_temp\":85} true\n", want: "malformed bounds JSON"},
		{name: "numeric string", content: "{\"conductor_temp\":\"85\"}\n", want: "malformed bounds JSON field conductor_temp"},
		{name: "nested numeric string", content: "{\"conductor_temp\":{\"value\":\"85\",\"unit\":\"C\"}}\n", want: "value is not a number"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writeFile(t, filepath.Join(root, tc.name+".json"), tc.content)
			args := []string{"check", "bounds", "--check-all", tc.name + ".json"}
			if tc.coded {
				args = append(args, "--json")
			}
			var stdout bytes.Buffer
			err := (Runner{Stdout: &stdout, WorkingDir: root}).Run(args)
			if tc.coded {
				var exitErr ExitError
				if !errors.As(err, &exitErr) || exitErr.Code != 1 {
					t.Fatalf("err=%v stdout=%s, want exit 1", err, stdout.String())
				}
				if !strings.Contains(stdout.String(), tc.want) {
					t.Fatalf("stdout=%s, want %q", stdout.String(), tc.want)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, want %q", err, tc.want)
			}
		})
	}
}

func TestRunnerCheckUnitsNumericalParityWithoutInterpreter(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	mph := strconv.FormatFloat(3.6*0.621371, 'g', 6, 64)
	cases := []struct {
		name string
		args []string
		fail bool
		want string
	}{
		{name: "C to K", args: []string{"25", "C", "K"}, want: "25 C = 298.15 K"},
		{name: "0 C to F", args: []string{"0", "C", "F"}, want: "0 C = 32 F"},
		{name: "100 C to F", args: []string{"100", "C", "F"}, want: "100 C = 212 F"},
		{name: "-40 C to F", args: []string{"-40", "C", "F"}, want: "-40 C = -40 F"},
		{name: "celsius alias", args: []string{"25", "celsius", "K"}, want: "25 C = 298.15 K"},
		{name: "meters to feet", args: []string{"1", "m", "ft"}, want: "1 m = 3.28084 ft"},
		{name: "km to mi", args: []string{"1", "km", "mi"}, want: "1 km = 0.621371 mi"},
		{name: "ohm/km to ohm/mi", args: []string{"1", "ohm/km", "ohm/mi"}, want: "1 ohm/km = 1.60934 ohm/mi"},
		{name: "W to kW", args: []string{"1000", "W", "kW"}, want: "1000 W = 1 kW"},
		{name: "kW to hp", args: []string{"1", "kW", "hp"}, want: "1 kW = 1.34102 hp"},
		{name: "m/s to km/h", args: []string{"1", "m/s", "km/h"}, want: "1 m/s = 3.6 km/h"},
		{name: "m/s to mph", args: []string{"1", "m/s", "mph"}, want: "1 m/s = " + mph + " mph"},
		{name: "same unit", args: []string{"25", "C", "C"}, want: "25 C = 25 C (same unit)"},
		{name: "unknown", args: []string{"1", "C", "m"}, fail: true, want: "Unknown conversion"},
		{name: "bare c is not alias", args: []string{"25", "c", "K"}, fail: true, want: "Unknown conversion"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			args := append([]string{"check", "units"}, tc.args...)
			args = append(args, "--json")
			err := (Runner{Stdout: &stdout, WorkingDir: root}).Run(args)
			var output checkOperatorJSON
			if unmarshalErr := json.Unmarshal(stdout.Bytes(), &output); unmarshalErr != nil {
				t.Fatalf("json: %v stdout=%s", unmarshalErr, stdout.String())
			}
			joined := strings.Join(append(output.Errors, output.Findings...), "\n")
			if tc.fail {
				var exitErr ExitError
				if !errors.As(err, &exitErr) || exitErr.Code != 1 {
					t.Fatalf("err=%v, want fail", err)
				}
			} else if err != nil {
				t.Fatalf("err=%v stdout=%s", err, stdout.String())
			}
			if !strings.Contains(joined, tc.want) {
				t.Fatalf("output=%#v, want %q", output, tc.want)
			}
		})
	}
	err := (Runner{WorkingDir: root}).Run([]string{"check", "units", "hot", "C", "K"})
	if err == nil || !strings.Contains(err.Error(), "malformed value") {
		t.Fatalf("non-numeric: %v", err)
	}
}

func TestRunnerCheckStandardRefsRequiresCitations(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	mkdirAll(t, filepath.Join(root, "src"))
	writeFile(t, filepath.Join(root, "src", "ok.py"), "# CIGRE TB 601, Section 4.2.3: Natural convection heat loss\ndef convection():\n    return 1\n")
	if err := (Runner{WorkingDir: root}).Run([]string{"check", "standard-refs", "src"}); err != nil {
		t.Fatalf("cited physics: %v", err)
	}
	writeFile(t, filepath.Join(root, "src", "missing.py"), "def convection():\n    return 1\n")
	var stdout bytes.Buffer
	err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "standard-refs", "src", "--json"})
	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("missing cite: %v", err)
	}
	var output checkOperatorJSON
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(output.Findings, "\n"), "missing.py") {
		t.Fatalf("findings=%#v", output.Findings)
	}
	writeFile(t, filepath.Join(root, "src", "nosection.py"), "# IEEE 738-2012 thermal rating\ndef convection():\n    return 1\n")
	stdout.Reset()
	err = (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "standard-refs", filepath.Join("src", "nosection.py"), "--json"})
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("no section: %v", err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(output.Findings, "\n"), "section/chapter") {
		t.Fatalf("findings=%#v", output.Findings)
	}
}

func TestRunnerCheckStandardRefsFailsClosedOnUnreadable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	err := (Runner{WorkingDir: root}).Run([]string{"check", "standard-refs", "missing"})
	if err == nil || !strings.Contains(err.Error(), "unreadable path") {
		t.Fatalf("missing: %v", err)
	}
	skipWithoutEnforcedPermissions(t)
	locked := filepath.Join(root, "locked.py")
	writeFile(t, locked, "def convection():\n    return 1\n")
	chmodForTest(t, locked, 0o000)
	err = (Runner{WorkingDir: root}).Run([]string{"check", "standard-refs", "locked.py"})
	if err == nil || !strings.Contains(err.Error(), "unreadable file") {
		t.Fatalf("unreadable file: %v", err)
	}
	denied := filepath.Join(root, "denied")
	mkdirAll(t, denied)
	writeFile(t, filepath.Join(denied, "x.py"), "def convection():\n    return 1\n")
	chmodForTest(t, denied, 0o000)
	t.Cleanup(func() { _ = os.Chmod(denied, 0o755) })
	err = (Runner{WorkingDir: root}).Run([]string{"check", "standard-refs", "denied"})
	if err == nil || !strings.Contains(err.Error(), "unreadable path") {
		t.Fatalf("unreadable dir: %v", err)
	}
}

func TestRunnerCheckBoundsAndUnitsList(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	var stdout bytes.Buffer
	if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "bounds", "--list-bounds", "--json"}); err != nil {
		t.Fatalf("list bounds: %v", err)
	}
	if !strings.Contains(stdout.String(), "conductor_temp") || !strings.Contains(stdout.String(), "∞") {
		t.Fatalf("list bounds stdout=%s", stdout.String())
	}
	stdout.Reset()
	if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run([]string{"check", "units", "--list", "--json"}); err != nil {
		t.Fatalf("list units: %v", err)
	}
	if !strings.Contains(stdout.String(), "ohm/km") {
		t.Fatalf("list units stdout=%s", stdout.String())
	}
}

func TestRunnerCheckUnitsAndBoundsTextOutput(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{name: "convert C to K", args: []string{"check", "units", "25", "C", "K"}, want: []string{"ok units: passed", "25 C = 298.15 K"}},
		{name: "list units", args: []string{"check", "units", "--list"}, want: []string{"ok units: passed", "Temperature: C, K, F", "ohm/km"}},
		{name: "list bounds", args: []string{"check", "bounds", "--list-bounds"}, want: []string{"ok bounds: passed", "conductor_temp", "∞"}},
		{name: "in-range bound", args: []string{"check", "bounds", "--type", "conductor_temp", "--value", "85"}, want: []string{"ok bounds: passed", "within valid range"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			if err := (Runner{Stdout: &stdout, WorkingDir: root}).Run(tc.args); err != nil {
				t.Fatalf("err=%v stdout=%s", err, stdout.String())
			}
			got := stripANSI(stdout.String())
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Fatalf("stdout=%q, want %q", got, want)
				}
			}
		})
	}
}
