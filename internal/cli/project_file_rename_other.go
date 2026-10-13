//go:build !darwin && !linux

package cli

import (
	"fmt"
	"os"
)

func exclusiveRename(from *os.File, oldname string, to *os.File, newname string) error {
	return fmt.Errorf("atomic no-replace project repair is unavailable on this platform")
}
