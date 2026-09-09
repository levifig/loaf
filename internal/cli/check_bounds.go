package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
)

type powerBounds struct {
	minVal      float64
	maxVal      float64
	unit        string
	description string
}

var powerPhysicalBounds = map[string]powerBounds{
	"conductor_temp":  {-40, 250, "°C", "Conductor material limits"},
	"ambient_temp":    {-50, 60, "°C", "Operational temperature range"},
	"wind_speed":      {0, 50, "m/s", "Extreme weather limit"},
	"solar_radiation": {0, 1400, "W/m²", "Max solar constant"},
	"current":         {0, math.Inf(1), "A", "Must be non-negative"},
	"flux_density":    {0, 2, "T", "Steel saturation limit"},
	"resistance":      {0, math.Inf(1), "Ω/km", "Must be non-negative"},
	"sag":             {0, 500, "m", "Reasonable sag range"},
	"tension":         {0, 500000, "N", "Reasonable tension range"},
	"span_length":     {10, 2000, "m", "Typical span range"},
}

var powerBoundsTypeOrder = []string{
	"conductor_temp", "ambient_temp", "wind_speed", "solar_radiation",
	"current", "flux_density", "resistance", "sag", "tension", "span_length",
}

type checkBoundsOptions struct {
	paramType  string
	value      float64
	hasValue   bool
	unit       string
	checkAll   string
	listBounds bool
	jsonOutput bool
}

func writeCheckBoundsHelp(out io.Writer) {
	writeUsageHelp(out, "loaf check bounds (--type <type> --value <value> [--unit <unit>] | --check-all <file> | --list-bounds) [--json]", "Validate physics values against CIGRE/IEEE physical bounds. Temperature unit K is converted to Celsius before comparison.", "--type, -t        Parameter type", "--value, -v       Value to validate", "--unit, -u        Unit (C or K for temperatures)", "--check-all, -c   JSON object of values or {value, unit} objects", "--list-bounds, -l List known bounds", "--json            Output pass/fail, exit code, warnings, errors, and findings as JSON")
}

func (r Runner) runCheckBounds(args []string, out io.Writer, errOut io.Writer, runtimeRoot string) error {
	options, err := parseCheckBoundsArgs(args)
	if err != nil {
		return err
	}
	if options.listBounds {
		return writeCheckOperatorResult(out, errOut, "bounds", listPowerBoundsResult(), options.jsonOutput)
	}
	if options.checkAll != "" {
		result, err := validatePowerBoundsJSONFile(resolveCheckOperatorPath(runtimeRoot, options.checkAll))
		if err != nil {
			return err
		}
		return writeCheckOperatorResult(out, errOut, "bounds", result, options.jsonOutput)
	}
	if options.paramType == "" || !options.hasValue {
		return fmt.Errorf("either --type and --value, --check-all, or --list-bounds is required")
	}
	ok, message := validatePowerBoundValue(options.paramType, options.value, options.unit)
	result := checkResult{Passed: ok, Warnings: []string{}, Errors: []string{}, Findings: []string{message}}
	if !ok {
		result.Blocked = true
		result.Errors = append(result.Errors, message)
	}
	return writeCheckOperatorResult(out, errOut, "bounds", result, options.jsonOutput)
}

func parseCheckBoundsArgs(args []string) (checkBoundsOptions, error) {
	var options checkBoundsOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--json":
			options.jsonOutput = true
		case "--list-bounds", "-l":
			options.listBounds = true
		case "--type", "-t":
			value, err := consumeFlagValue(args, &i, arg)
			if err != nil {
				return options, err
			}
			options.paramType = value
		case "--value", "-v":
			raw, err := consumeFlagValue(args, &i, arg)
			if err != nil {
				return options, err
			}
			value, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return options, fmt.Errorf("malformed --value %q", raw)
			}
			options.value = value
			options.hasValue = true
		case "--unit", "-u":
			value, err := consumeFlagValue(args, &i, arg)
			if err != nil {
				return options, err
			}
			options.unit = value
		case "--check-all", "-c":
			value, err := consumeFlagValue(args, &i, arg)
			if err != nil {
				return options, err
			}
			options.checkAll = value
		default:
			return options, fmt.Errorf("unknown check option %q", arg)
		}
	}
	return options, nil
}

func validatePowerBoundValue(paramType string, value float64, unit string) (bool, string) {
	bounds, ok := powerPhysicalBounds[paramType]
	if !ok {
		return false, "Unknown parameter type: " + paramType + ". Valid types: " + strings.Join(powerBoundsTypeOrder, ", ")
	}
	normalizedUnit, err := normalizePowerBoundUnit(paramType, unit)
	if err != nil {
		return false, err.Error()
	}
	if (paramType == "conductor_temp" || paramType == "ambient_temp") && normalizedUnit == "K" {
		value = value - 273.15
	}
	if value < bounds.minVal {
		return false, fmt.Sprintf("%s = %s %s is below minimum (%s %s)", paramType, formatPowerNumber(value), bounds.unit, formatPowerNumber(bounds.minVal), bounds.unit)
	}
	if value > bounds.maxVal {
		return false, fmt.Sprintf("%s = %s %s exceeds maximum (%s %s)", paramType, formatPowerNumber(value), bounds.unit, formatPowerMax(bounds.maxVal), bounds.unit)
	}
	return true, fmt.Sprintf("%s = %s %s is within valid range [%s, %s]", paramType, formatPowerNumber(value), bounds.unit, formatPowerNumber(bounds.minVal), formatPowerMax(bounds.maxVal))
}

func validatePowerBoundsJSONFile(path string) (checkResult, error) {
	content, err := readCheckOperatorFile(path)
	if err != nil {
		return checkResult{}, fmt.Errorf("unreadable bounds file %s: %w", path, err)
	}
	return validatePowerBoundsJSON(content)
}

func validatePowerBoundsJSON(content string) (checkResult, error) {
	result := checkResult{Passed: true, Warnings: []string{}, Errors: []string{}, Findings: []string{}}
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &object); err != nil {
		return checkResult{}, fmt.Errorf("malformed bounds JSON: %w", err)
	}
	if object == nil {
		return checkResult{}, fmt.Errorf("malformed bounds JSON: expected a non-null object")
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value, unit, err := parsePowerBoundJSONValue(object[key])
		if err != nil {
			return checkResult{}, fmt.Errorf("malformed bounds JSON field %s: %w", key, err)
		}
		ok, message := validatePowerBoundValue(key, value, unit)
		result.Findings = append(result.Findings, message)
		if !ok {
			result.Passed = false
			result.Blocked = true
			result.Errors = append(result.Errors, message)
		}
	}
	return result, nil
}

func parsePowerBoundJSONValue(raw json.RawMessage) (float64, string, error) {
	raw = bytes.TrimSpace(raw)
	if jsonRawIsNull(raw) || len(raw) == 0 {
		return 0, "", fmt.Errorf("value must be a number or {value, unit} object")
	}
	if raw[0] == '{' {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil || object == nil {
			return 0, "", fmt.Errorf("value must be a number or {value, unit} object")
		}
		rawValue, ok := object["value"]
		if !ok {
			return 0, "", fmt.Errorf("missing value")
		}
		number, err := parseJSONFloat(rawValue)
		if err != nil {
			return 0, "", fmt.Errorf("value is not a number")
		}
		if rawUnit, exists := object["unit"]; exists {
			if jsonRawIsNull(rawUnit) {
				return 0, "", fmt.Errorf("unit is not a string")
			}
			var unit string
			if err := json.Unmarshal(rawUnit, &unit); err != nil {
				return 0, "", fmt.Errorf("unit is not a string")
			}
			return number, unit, nil
		}
		return number, "", nil
	}
	number, err := parseJSONFloat(raw)
	if err != nil {
		return 0, "", fmt.Errorf("value must be a number or {value, unit} object")
	}
	return number, "", nil
}

func normalizePowerBoundUnit(paramType, unit string) (string, error) {
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return "", nil
	}
	switch paramType {
	case "conductor_temp", "ambient_temp":
		switch strings.ToLower(unit) {
		case "c", "celsius", "°c", "degc":
			return "C", nil
		case "k", "kelvin", "°k", "degk":
			return "K", nil
		default:
			return "", fmt.Errorf("unsupported unit %q for %s", unit, paramType)
		}
	default:
		expected := powerPhysicalBounds[paramType].unit
		if unit != expected && !strings.EqualFold(unit, expected) {
			return "", fmt.Errorf("unsupported unit %q for %s", unit, paramType)
		}
		return unit, nil
	}
}

func parseJSONFloat(raw json.RawMessage) (float64, error) {
	raw = bytes.TrimSpace(raw)
	if jsonRawIsNull(raw) {
		return 0, fmt.Errorf("null")
	}
	if !jsonRawIsNumber(raw) {
		return 0, fmt.Errorf("not a number")
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err != nil {
		return 0, err
	}
	return number, nil
}

func jsonRawIsNumber(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	switch raw[0] {
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	default:
		return false
	}
}

func jsonRawIsNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func listPowerBoundsResult() checkResult {
	result := checkResult{Passed: true, Warnings: []string{}, Errors: []string{}, Findings: []string{}}
	for _, name := range powerBoundsTypeOrder {
		bounds := powerPhysicalBounds[name]
		result.Findings = append(result.Findings, fmt.Sprintf("%s %s %s %s %s", name, formatPowerNumber(bounds.minVal), formatPowerMax(bounds.maxVal), bounds.unit, bounds.description))
	}
	return result
}

func formatPowerNumber(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

func formatPowerMax(value float64) string {
	if math.IsInf(value, 1) {
		return "∞"
	}
	return formatPowerNumber(value)
}
