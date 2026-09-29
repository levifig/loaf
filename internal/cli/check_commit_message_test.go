package cli

import "testing"

// A commit invocation frequently shares a Bash call with other commands, and
// some of those carry heredocs of their own. The extractor must attribute a
// heredoc to the command that opened it.
func TestExtractCommitMessageAttributesHeredocsToTheirOwnCommand(t *testing.T) {
	const marker = "EOF"
	heredoc := func(body string) string {
		return "\"$(cat <<'" + marker + "'\n" + body + "\n" + marker + "\n)\""
	}

	cases := []struct {
		name    string
		command string
		want    string
	}{
		{
			name:    "later command's heredoc is not the commit message",
			command: "git add -A\ngit commit -m \"chore: release the thing\"\ngh pr create --body " + heredoc("Release of five landed pull requests."),
			want:    "chore: release the thing",
		},
		{
			name:    "separator is &&",
			command: "git commit -m \"fix: correct the thing\" && gh pr create --body " + heredoc("Body prose."),
			want:    "fix: correct the thing",
		},
		{
			name:    "separator is a semicolon",
			command: "git commit -m \"docs: describe the thing\"; gh release create v1.2.3 --notes " + heredoc("Notes prose."),
			want:    "docs: describe the thing",
		},
		{
			name:    "a heredoc that really is the message still wins",
			command: "git commit -m " + heredoc("feat: add the thing\n\nA body paragraph."),
			want:    "feat: add the thing\n\nA body paragraph.",
		},
		{
			name:    "heredoc preserves quoted phrases",
			command: "git commit -m " + heredoc("fix: explain \"permission denied\" errors\n\nBody."),
			want:    "fix: explain \"permission denied\" errors\n\nBody.",
		},
		{
			name:    "heredoc preserves an unmatched quote",
			command: "git commit -m " + heredoc("fix: support 5\" disks"),
			want:    "fix: support 5\" disks",
		},
		{
			name:    "config heredoc does not replace the commit message",
			command: "git -c example.key=" + heredoc("configuration text") + " commit -m " + heredoc("fix: explain \"permission denied\" errors"),
			want:    "fix: explain \"permission denied\" errors",
		},
		{
			name:    "later message heredoc does not replace the subject",
			command: "git commit -m \"fix: subject\" -m " + heredoc("Body with \"quoted\" text."),
			want:    "fix: subject",
		},
		{
			name:    "all and message short flags",
			command: "git commit -am " + heredoc("feat: add the thing"),
			want:    "feat: add the thing",
		},
		{
			name:    "long message flag",
			command: "git commit --message " + heredoc("feat: add the thing"),
			want:    "feat: add the thing",
		},
		{
			name:    "assigned long message flag",
			command: "git commit --message=" + heredoc("feat: add the thing"),
			want:    "feat: add the thing",
		},
		{
			name:    "mixed message flags keep the subject",
			command: "git commit -am \"fix: subject\" --message " + heredoc("Body with \"quoted\" text."),
			want:    "fix: subject",
		},
		{
			name:    "quiet and message short flags",
			command: "git commit -qm " + heredoc("feat: add the thing"),
			want:    "feat: add the thing",
		},
		{
			name:    "combined boolean short flags",
			command: "git commit -qasm " + heredoc("feat: add the thing"),
			want:    "feat: add the thing",
		},
		{
			name:    "attached combined message flag",
			command: "git commit -am" + heredoc("feat: add the thing"),
			want:    "feat: add the thing",
		},
		{
			name:    "attached single message flag",
			command: "git commit -m" + heredoc("feat: add the thing"),
			want:    "feat: add the thing",
		},
		{
			name:    "signing key is not a message flag",
			command: "git commit -Smy-key -m \"fix: subject\"",
			want:    "fix: subject",
		},
		{
			name:    "plain single flag",
			command: "git commit -m \"docs: plain subject\"",
			want:    "docs: plain subject",
		},
		{
			name:    "repeated flags take the subject",
			command: "git commit -m \"chore: the subject\" -m \"the body paragraph\"",
			want:    "chore: the subject",
		},
		{
			name:    "text before the commit is ignored",
			command: "cat notes.md\ngit commit -m \"refactor: rename the thing\"",
			want:    "refactor: rename the thing",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractCommitMessage(tc.command); got != tc.want {
				t.Fatalf("extractCommitMessage()\n  got  %q\n  want %q", got, tc.want)
			}
		})
	}
}
