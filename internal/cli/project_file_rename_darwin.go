package cli

import (
	"golang.org/x/sys/unix"
	"os"
)

func exclusiveRename(from *os.File, oldname string, to *os.File, newname string) error {
	return unix.RenameatxNp(int(from.Fd()), oldname, int(to.Fd()), newname, unix.RENAME_EXCL)
}
