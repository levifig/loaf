package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func writeCheckUnitsHelp(out io.Writer) {
	writeUsageHelp(out, "loaf check units <value> <from-unit> <to-unit> [--json]", "Convert power-system units (temperature, length, resistance, power, speed). Unknown conversions fail.", "--list, -l  List available conversions", "--json      Output pass/fail, exit code, warnings, errors, and findings as JSON")
}

func (r Runner) runCheckUnits(args []string, out io.Writer, errOut io.Writer, _ string) error {
	options, err := parseCheckUnitsArgs(args)
	if err != nil {
		return err
	}
	if options.list {
		return writeCheckOperatorResult(out, errOut, "units", listPowerUnitConversions(), options.jsonOutput)
	}
	if !options.hasValue || options.from == "" || options.to == "" {
		return fmt.Errorf("value, from-unit, and to-unit are required")
	}
	result, message := convertPowerUnits(options.value, options.from, options.to)
	output := checkResult{Passed: true, Warnings: []string{}, Errors: []string{}, Findings: []string{message}}
	if !result {
		output.Passed = false
		output.Blocked = true
		output.Errors = append(output.Errors, message)
	}
	return writeCheckOperatorResult(out, errOut, "units", output, options.jsonOutput)
}

type checkUnitsOptions struct {
	value      float64
	hasValue   bool
	from       string
	to         string
	list       bool
	jsonOutput bool
}

func parseCheckUnitsArgs(args []string) (checkUnitsOptions, error) {
	var options checkUnitsOptions
	var positionals []string
	positionalOnly := false
	for _, arg := range args {
		if positionalOnly {
			positionals = append(positionals, arg)
			continue
		}
		switch arg {
		case "--json":
			options.jsonOutput = true
		case "--list", "-l":
			options.list = true
		case "--":
			positionalOnly = true
		default:
			if strings.HasPrefix(arg, "-") && arg != "-" {
				if _, err := strconv.ParseFloat(arg, 64); err != nil {
					return options, fmt.Errorf("unknown check option %q", arg)
				}
			}
			positionals = append(positionals, arg)
		}
	}
	if len(positionals) == 0 {
		return options, nil
	}
	if len(positionals) != 3 {
		return options, fmt.Errorf("expected <value> <from-unit> <to-unit>")
	}
	value, err := strconv.ParseFloat(positionals[0], 64)
	if err != nil {
		return options, fmt.Errorf("malformed value %q", positionals[0])
	}
	options.value = value
	options.hasValue = true
	options.from = positionals[1]
	options.to = positionals[2]
	return options, nil
}

func convertPowerUnits(value float64, fromUnit, toUnit string) (bool, string) {
	fromUnit = normalizePowerUnit(fromUnit)
	toUnit = normalizePowerUnit(toUnit)
	if fromUnit == toUnit {
		return true, fmt.Sprintf("%s %s = %s %s (same unit)", formatPowerNumber(value), fromUnit, formatPowerNumber(value), toUnit)
	}
	fn, ok := powerUnitConversions()[fromUnit+"\x00"+toUnit]
	if !ok {
		return false, "Unknown conversion: " + fromUnit + " -> " + toUnit
	}
	return true, fmt.Sprintf("%s %s = %s %s", formatPowerNumber(value), fromUnit, formatPowerSignificant(fn(value)), toUnit)
}

func normalizePowerUnit(unit string) string {
	unit = strings.TrimSpace(unit)
	aliases := map[string]string{
		"celsius": "C", "°c": "C", "degc": "C",
		"kelvin": "K", "°k": "K", "degk": "K",
		"fahrenheit": "F", "°f": "F", "degf": "F",
		"meter": "m", "meters": "m", "metre": "m",
		"feet": "ft", "foot": "ft",
		"kilometer": "km", "kilometers": "km",
		"mile": "mi", "miles": "mi",
		"watt": "W", "watts": "W",
		"kilowatt": "kW", "kilowatts": "kW",
		"megawatt": "MW", "megawatts": "MW",
		"horsepower": "hp",
	}
	if mapped, ok := aliases[strings.ToLower(unit)]; ok {
		return mapped
	}
	return unit
}

func powerUnitConversions() map[string]func(float64) float64 {
	cToK := func(c float64) float64 { return c + 273.15 }
	kToC := func(k float64) float64 { return k - 273.15 }
	cToF := func(c float64) float64 { return c*9/5 + 32 }
	fToC := func(f float64) float64 { return (f - 32) * 5 / 9 }
	mToFt := func(m float64) float64 { return m * 3.28084 }
	ftToM := func(ft float64) float64 { return ft / 3.28084 }
	mToKm := func(m float64) float64 { return m / 1000 }
	kmToM := func(km float64) float64 { return km * 1000 }
	kmToMi := func(km float64) float64 { return km * 0.621371 }
	miToKm := func(mi float64) float64 { return mi / 0.621371 }
	ohmKmToOhmMi := func(r float64) float64 { return r * 1.60934 }
	ohmMiToOhmKm := func(r float64) float64 { return r / 1.60934 }
	ohmKmToOhmKft := func(r float64) float64 { return r * 0.3048 }
	ohmKftToOhmKm := func(r float64) float64 { return r / 0.3048 }
	wToKw := func(w float64) float64 { return w / 1000 }
	kwToW := func(kw float64) float64 { return kw * 1000 }
	kwToMw := func(kw float64) float64 { return kw / 1000 }
	mwToKw := func(mw float64) float64 { return mw * 1000 }
	kwToHp := func(kw float64) float64 { return kw * 1.34102 }
	hpToKw := func(hp float64) float64 { return hp / 1.34102 }
	msToFts := func(ms float64) float64 { return ms * 3.28084 }
	ftsToMs := func(fts float64) float64 { return fts / 3.28084 }
	msToKmh := func(ms float64) float64 { return ms * 3.6 }
	kmhToMs := func(kmh float64) float64 { return kmh / 3.6 }
	kmhToMph := func(kmh float64) float64 { return kmh * 0.621371 }
	mphToKmh := func(mph float64) float64 { return mph / 0.621371 }
	return map[string]func(float64) float64{
		pairUnit("C", "K"):            cToK,
		pairUnit("K", "C"):            kToC,
		pairUnit("C", "F"):            cToF,
		pairUnit("F", "C"):            fToC,
		pairUnit("K", "F"):            func(k float64) float64 { return cToF(kToC(k)) },
		pairUnit("F", "K"):            func(f float64) float64 { return cToK(fToC(f)) },
		pairUnit("m", "ft"):           mToFt,
		pairUnit("ft", "m"):           ftToM,
		pairUnit("m", "km"):           mToKm,
		pairUnit("km", "m"):           kmToM,
		pairUnit("km", "mi"):          kmToMi,
		pairUnit("mi", "km"):          miToKm,
		pairUnit("m", "mi"):           func(m float64) float64 { return kmToMi(mToKm(m)) },
		pairUnit("mi", "m"):           func(mi float64) float64 { return kmToM(miToKm(mi)) },
		pairUnit("ohm/km", "ohm/mi"):  ohmKmToOhmMi,
		pairUnit("ohm/mi", "ohm/km"):  ohmMiToOhmKm,
		pairUnit("ohm/km", "ohm/kft"): ohmKmToOhmKft,
		pairUnit("ohm/kft", "ohm/km"): ohmKftToOhmKm,
		pairUnit("W", "kW"):           wToKw,
		pairUnit("kW", "W"):           kwToW,
		pairUnit("kW", "MW"):          kwToMw,
		pairUnit("MW", "kW"):          mwToKw,
		pairUnit("kW", "hp"):          kwToHp,
		pairUnit("hp", "kW"):          hpToKw,
		pairUnit("W", "MW"):           func(w float64) float64 { return kwToMw(wToKw(w)) },
		pairUnit("MW", "W"):           func(mw float64) float64 { return kwToW(mwToKw(mw)) },
		pairUnit("m/s", "ft/s"):       msToFts,
		pairUnit("ft/s", "m/s"):       ftsToMs,
		pairUnit("m/s", "km/h"):       msToKmh,
		pairUnit("km/h", "m/s"):       kmhToMs,
		pairUnit("km/h", "mph"):       kmhToMph,
		pairUnit("mph", "km/h"):       mphToKmh,
		pairUnit("m/s", "mph"):        func(ms float64) float64 { return kmhToMph(msToKmh(ms)) },
		pairUnit("mph", "m/s"):        func(mph float64) float64 { return kmhToMs(mphToKmh(mph)) },
	}
}

func pairUnit(from, to string) string {
	return from + "\x00" + to
}

func listPowerUnitConversions() checkResult {
	return checkResult{
		Passed:   true,
		Warnings: []string{},
		Errors:   []string{},
		Findings: []string{
			"Temperature: C, K, F",
			"Length: m, ft, km, mi",
			"Resistance: ohm/km, ohm/mi, ohm/kft",
			"Power: W, kW, MW, hp",
			"Speed: m/s, ft/s, km/h, mph",
		},
	}
}

func formatPowerSignificant(value float64) string {
	return strconv.FormatFloat(value, 'g', 6, 64)
}
