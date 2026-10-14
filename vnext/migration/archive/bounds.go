package archive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Inspect structural amplification before allocating typed slices. The outer
// byte cap alone does not bound millions of tiny objects or recursive values.
func inspectJSONBounds(encoded []byte) error {
	if !utf8.Valid(encoded) {
		return fmt.Errorf("archive JSON contains invalid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	var walk func(int, string) error
	tokens := 0
	walk = func(depth int, key string) error {
		tokens++
		if depth > 32 || tokens > 2_000_000 {
			return fmt.Errorf("archive JSON structure exceeds bounds")
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				field, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := field.(string)
				if !ok {
					return fmt.Errorf("invalid JSON field")
				}
				folded := strings.ToLower(name)
				if seen[folded] {
					return fmt.Errorf("duplicate JSON field %q", name)
				}
				seen[folded] = true
				if err := walk(depth+1, name); err != nil {
					return err
				}
			}
		case '[':
			limit := maxRecords
			if key == "Items" || key == "suggested_skills" || key == "kinds" {
				limit = 64
			}
			n := 0
			for decoder.More() {
				n++
				if n > limit {
					if key == "records" {
						return fmt.Errorf("archive record count exceeds %d", limit)
					}
					return fmt.Errorf("archive %s array exceeds %d items", key, limit)
				}
				if err := walk(depth+1, ""); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("invalid JSON delimiter")
		}
		_, err = decoder.Token()
		return err
	}
	if err := walk(0, ""); err != nil {
		return err
	}
	return requireJSONEOFV1(decoder)
}
