package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

func (r Runner) runCheckCommitMsg(args []string, out io.Writer, errOut io.Writer, runtimeRoot string) error {
	options, err := parseCheckOperatorArgs(args, true)
	if err != nil {
		return err
	}
	message, err := r.readCommitMessageInput(options.path, runtimeRoot)
	if err != nil {
		return err
	}
	return writeCheckOperatorResult(out, errOut, "commit-msg", evaluateCommitMessage(message, runtimeRoot), options.jsonOutput)
}

func (r Runner) readCommitMessageInput(path, runtimeRoot string) (string, error) {
	if path == "-" {
		reader := firstReader(r.Stdin, os.Stdin)
		if reader == nil {
			return "", fmt.Errorf("stdin is unavailable")
		}
		body, err := io.ReadAll(io.LimitReader(reader, projectFileReadLimit+1))
		if err != nil {
			return "", err
		}
		if int64(len(body)) > projectFileReadLimit {
			return "", fmt.Errorf("commit message exceeds the project read limit")
		}
		if !utf8.Valid(body) {
			return "", fmt.Errorf("commit message is not valid UTF-8")
		}
		return strings.ReplaceAll(string(body), "\r\n", "\n"), nil
	}
	return readCheckOperatorFile(resolveCheckOperatorPath(runtimeRoot, path))
}
