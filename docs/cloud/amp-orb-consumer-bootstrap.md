# Amp Orb Consumer Bootstrap

This guide installs a pinned Loaf release into an Amp Orb for an external consumer repository. It does not build Loaf, depend on a Loaf checkout, or place release bytes in the consumer repository.

## Contents

- Consumer files
- Configure the pin
- Setup and resume
- Upgrade a project
- Provenance and secrets boundary
- Preserve consumer-owned instructions
- Disposable clean-room proof

## Consumer files

Copy the contents of [`content/templates/amp-orb/.agents/`](../../content/templates/amp-orb/.agents/) into the consumer repository's `.agents/` directory:

```text
.agents/
├── loaf-orb.pin
├── loaf-orb-bootstrap.sh
├── resume
└── setup
```

Commit those four project-owned files and make the three scripts executable. Do not copy a Loaf binary, source checkout, credential, user home, or machine-specific path into the repository.

The bootstrap installs release bytes outside the checkout. Its default per-project user prefix is `${XDG_DATA_HOME:-$HOME/.local/share}/loaf/orb/<project.id>`. `LOAF_ORB_HOME` may select a different absolute Orb-local user prefix. The exact pinned release directory is `<prefix>/releases/<version>`.

Amp starts the later agent process with project environment configuration rather than the transient environment exported by setup or resume. Configure that environment's `PATH` to include `${LOAF_BIN_DIR:-$HOME/.local/bin}`. The bootstrap safely maintains `<LOAF_BIN_DIR>/loaf` as a symlink to the exact pinned release binary, prepends that user bin for its own process, and proves ordinary `loaf` resolution reaches the pinned executable. It refuses to replace a regular file or a symlink owned by another installation; only a prior link into this project's Orb-local prefix is replaceable.

## Configure the pin

`.agents/loaf-orb.pin` is an eight-line, closed-format record. Replace every placeholder and retain exactly these keys, once each:

```text
schema=1
version=2.0.0-alpha.1
archive.linux-x64.sha256=<64 lowercase hexadecimal characters>
archive.linux-arm64.sha256=<64 lowercase hexadecimal characters>
project.id=<durable Loaf project ID>
git.remote=owner/repository
tracker.provider=github
tracker.scope=owner/repository#board-name
```

Take each archive checksum from the official GitHub release and review it as part of the pin change. `git.remote` is the unauthenticated `owner/repository` identity, not an HTTPS or SSH URL. Use a provider-native, non-secret tracker scope. The parser rejects missing, duplicate, unknown, empty, or malformed lines before using any value.

The pin must not contain credentials, authenticated URLs, access tokens, machine paths, user names, email addresses, or other operator identity. Project and tracker identities are repository metadata; authentication stays in the Orb's secret store and harness-owned connection.

## Setup and resume

Amp invokes `.agents/setup` for a fresh Orb and `.agents/resume` when waking an existing Orb. Both are small POSIX entry points into `.agents/loaf-orb-bootstrap.sh`.

Setup performs this fail-closed sequence:

1. Parse and strictly validate all pin fields.
2. Select the pinned `linux-x64` or `linux-arm64` checksum from the Orb architecture.
3. Download `loaf_<version>_<platform>.tar.gz` from the official `levifig/loaf` GitHub release with bounded connect and total timeouts.
4. Verify the archive with `shasum -a 256` or `sha256sum` before extraction.
5. Extract while preserving the archive's reviewed file modes, then require the expected release directory, executable `bin/loaf`, and `loaf-release-manifest.json`.
6. Activate the release under the Orb-local user prefix, safely link it through the configured user bin, and prove `command -v loaf` resolves through that link to the exact pinned binary.
7. Run `LOAF_PROJECT_ENV=1 <absolute-loaf> install --to amp --yes`.
8. Run `<absolute-loaf> harness readiness --target amp --pin .agents/loaf-orb.pin --archive-sha256 <verified-checksum>`.

Both setup and resume run full readiness first without taking a lock or changing files. A ready resume is entirely local and exits without a download, which keeps the normal path within Amp's roughly ten-second blocking resume window. A failed check acquires the project prefix's atomic `.bootstrap.lock`, records its PID owner, and reruns readiness once under the lock so a concurrent finisher avoids a duplicate download. A live owner is refused. A dead PID proves an orphaned lock, which the next process retires narrowly before acquiring its own lock; malformed or unexpectedly populated lock directories are refused rather than removed broadly. Only the current lock holder may reacquire, publish, activate, or install. Missing or failed locked readiness unconditionally reacquires the official archive, verifies its pinned checksum, atomically publishes the staged release, and only then runs install plus final readiness. That repair can exceed the blocking window; bounded download failure or any other failure must remain nonzero, and the setup/resume logs are authoritative. A receipt match alone never authorizes repair from existing bytes, and there is no unpinned repair path.

The release URL can be overridden only by setting both `LOAF_ORB_ISOLATED_TESTING=1` and `LOAF_ORB_TEST_RELEASE_BASE_URL` in a disposable test environment. Never set those variables on a real Orb.

## Upgrade a project

Upgrade by one reviewed pin commit:

1. Change `version` and both architecture checksums together.
2. Confirm the checksums against the assets of that exact official release.
3. Run `.agents/setup`, or let the next `.agents/resume` repair readiness to the new pin.
4. Commit any reviewed project-owned managed-content changes produced by `loaf install` together with the pin update.

The versioned prefix leaves the prior release available for rollback. Reverting the pin selects that version again; if its verified receipt is absent, setup reacquires and verifies it from the official release.

## Provenance and secrets boundary

The committed pin is the project's reviewable provenance root. The bootstrap uses a fixed official release origin, verifies the selected archive digest before extraction, requires `loaf-release-manifest.json`, writes the verified lowercase digest plus a newline to `.loaf-archive.sha256` beside the installed release, and reads that sidecar back when invoking readiness. The sidecar must be a non-symlink regular file containing exactly one lowercase digest line. A missing, malformed, pin-mismatched, or readiness-rejected sidecar forces fresh checksum-verified acquisition before install. The absolute executable and its installed distribution, not `PATH` discovery or the checkout, supply Loaf content.

Inside Amp's setup or resume lifecycle, readiness asks `amp orb id-token` for a short-lived project-scoped token and reads only its Amp `project_id` and optional `workspace_id` claims. It never persists or prints the token, and it ignores user, thread, and email claims. Outside an Orb lifecycle environment those Amp IDs are omitted with an explicit limitation; no identity is guessed. The committed `project.id` remains Loaf's continuity project identity and is intentionally distinct from Amp's project identifier.

Neither the pin nor the scripts authenticate to GitHub or the selected tracker. Keep provider tokens, GitHub credentials, Amp credentials, and any Loaf connection material in the Orb secret store or harness-managed connection. Do not embed credentials in `git.remote`, `tracker.scope`, `LOAF_ORB_HOME`, or the release URL override.

## Preserve consumer-owned instructions

The bootstrap never writes `AGENTS.md` itself. The absolute `loaf install --to amp --yes` invocation owns only Loaf's existing fenced section:

```text
<!-- loaf:managed:start -->
...
<!-- loaf:managed:end -->
```

Consumer prose outside that fence remains consumer-owned and must be preserved byte-for-byte. A single well-formed managed fence may be updated in place. Duplicate managed fences, malformed fence boundaries, or ambiguous legacy instruction sources must refuse rather than overwrite or choose a source silently. Resolve the ambiguity in the consumer repository, then rerun setup.

## Disposable clean-room proof

Run the proof in a temporary consumer repository, never in the Loaf source checkout:

```sh
proof_root=$(mktemp -d)
consumer=$proof_root/consumer
orb_home=$proof_root/orb-home
proof_home=$proof_root/home
mkdir -p "$consumer/.agents"
cp content/templates/amp-orb/.agents/loaf-orb.pin "$consumer/.agents/"
cp content/templates/amp-orb/.agents/loaf-orb-bootstrap.sh "$consumer/.agents/"
cp content/templates/amp-orb/.agents/setup "$consumer/.agents/"
cp content/templates/amp-orb/.agents/resume "$consumer/.agents/"
chmod +x "$consumer/.agents/setup" "$consumer/.agents/resume" "$consumer/.agents/loaf-orb-bootstrap.sh"
cd "$consumer"
git init
git remote add origin https://github.com/owner/repository.git
```

Fill the copy of `loaf-orb.pin` with a real release, both official checksums, a disposable durable project ID, `git.remote=owner/repository`, and non-secret tracker identity. The pin and origin must identify the same repository. Add a sentinel and commit the consumer-owned starting state before first setup; the synthetic local commit identity is disposable proof metadata and does not belong in the pin:

```sh
printf '%s\n' 'consumer instructions: preserve this line' > "$consumer/AGENTS.md"
cd "$consumer"
git config user.name 'Loaf clean-room proof'
git config user.email 'loaf-clean-room@example.invalid'
git add .agents/loaf-orb.pin .agents/loaf-orb-bootstrap.sh .agents/setup .agents/resume AGENTS.md
git commit -m 'Add pinned Amp Orb bootstrap'
HOME="$proof_home" XDG_CONFIG_HOME="$proof_home/.config" LOAF_ORB_HOME="$orb_home" LOAF_BIN_DIR="$proof_root/user-bin" .agents/setup
grep -F 'consumer instructions: preserve this line' AGENTS.md
HOME="$proof_home" XDG_CONFIG_HOME="$proof_home/.config" LOAF_ORB_HOME="$orb_home" LOAF_BIN_DIR="$proof_root/user-bin" .agents/resume
```

Expected evidence:

- Setup downloads one official archive, rejects any checksum mismatch before extraction, requires the manifest, installs beneath `$orb_home/releases/<version>`, activates `$proof_root/user-bin/loaf` to the exact pinned binary, preserves the sentinel, and reports readiness.
- Resume invokes readiness first and performs no download when ready. An offline second resume is a useful proof of that property.
- Changing one checksum to another valid-looking 64-character digest makes fresh setup fail before extraction.
- Adding a second managed fence or breaking one fence boundary makes install fail loudly without replacing consumer prose. Readiness also refuses a legacy name-only project skill ownership manifest, a foreign project-local Loaf plugin, or any duplicate effective Loaf skill/plugin source under the disposable global homes (`$HOME/.agents/skills`, `$HOME/.config/agents/skills`, `$XDG_CONFIG_HOME/amp/plugins`, or `$HOME/.amp/plugins`). Remove the legacy or duplicate source before continuing.
- Replacing the pin with an unknown key, duplicate key, authenticated URL, path-shaped identity, uppercase checksum, or placeholder makes bootstrap fail before network access.

For a local archive-server fixture, use the two isolated-testing variables and reproduce the official `/download/v<version>/<archive>` path beneath the fixture base URL. That override is test-only evidence and must never appear in committed consumer configuration.
