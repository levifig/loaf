const command = `git commit -m "fix: 'bug'"`;
const msgMatch = command.match(/-m(?:\s+|=)(?:["']([^"']+)["']|([^\s"']+))/);
const message = msgMatch ? (msgMatch[1] || msgMatch[2]) : "no match";
console.log(`Extracted: [${message}]`);

const conventionalCommitRegex = /^(feat|fix|docs|style|refactor|perf|test|chore|ci|build|revert)(\(.+\))?!?: .+/;
console.log(`Valid: ${conventionalCommitRegex.test(message)}`);
