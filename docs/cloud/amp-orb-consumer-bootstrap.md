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

Amp starts the later agent process from project settings, not from exports made inside setup or resume. Configure this non-secret project environment value literally; Amp project settings do not promise shell expansion:

```text
LOAF_PROJECT_ENV=1
```

Inspect `PATH` inside the real project Orb. If it already contains `/home/user/.local/bin`, leave the project `PATH` setting unset so Amp preserves its full inherited toolchain path. Otherwise, copy every existing entry exactly and prefix the literal user-bin directory; for example, if the observed value is `<existing-path>`, set `PATH=/home/user/.local/bin:<existing-path>` with the actual entries substituted literally. Do not replace it with a minimal system-only path, and do not use `$HOME` or another shell expression. Omit `LOAF_BIN_DIR` for this default. Set it only when the Orb needs a different absolute user-bin directory, and prefix that same literal directory to the preserved `PATH`.

The bootstrap safely maintains the selected user-bin `loaf` as a symlink to the exact pinned release binary. It refuses to replace a regular file or a symlink owned by another installation; only a prior link into this project's Orb-local prefix is replaceable.

## Configure the pin

`.agents/loaf-orb.pin` is an eight-line, closed-format record. Replace every placeholder and retain exactly these keys, once each:

```text
schema=1
version=0.6.0-rc.1
archive.linux-x64.sha256=<64 lowercase hexadecimal characters>
archive.linux-arm64.sha256=<64 lowercase hexadecimal characters>
project.id=<durable Loaf project ID>
git.remote=owner/repository
tracker.provider=github
tracker.scope=owner/repository#board-name
```

Take each archive checksum from the immutable official `v0.6.0-rc.1` GitHub release and review it as part of the pin change. The tag carries the leading `v`; the pin's strict SemVer value does not. `git.remote` is the unauthenticated `owner/repository` identity, not an HTTPS or SSH URL. Use a provider-native, non-secret tracker scope. The parser rejects missing, duplicate, unknown, empty, unsafe, uncommitted, or malformed pin data before network access.

The pin must not contain credentials, authenticated URLs, access tokens, machine paths, user names, email addresses, or other operator identity. Project and tracker identities are repository metadata; authentication stays in the Orb's secret store and harness-owned connection.

## Setup and resume

Amp invokes `.agents/setup` for a fresh Orb and `.agents/resume` when waking an existing Orb. Both are small POSIX entry points into `.agents/loaf-orb-bootstrap.sh`. The managed project plugin invokes `.agents/loaf-orb-bootstrap.sh agent-check` before agent tools become available.

Setup performs this fail-closed sequence:

1. Parse and strictly validate all pin fields.
2. Select the pinned `linux-x64` or `linux-arm64` checksum from the Orb architecture.
3. Download `loaf_<version>_<platform>.tar.gz` from the official `levifig/loaf` GitHub release with bounded connect and total timeouts.
4. Verify the archive with `shasum -a 256` or `sha256sum` before extraction.
5. Extract while preserving the archive's reviewed file modes, then require the expected release directory, executable `bin/loaf`, and `loaf-release-manifest.json`.
6. Activate the release under the Orb-local user prefix, safely link it through the configured user bin, and prove `command -v loaf` resolves through that link to the exact pinned binary.
7. Run `LOAF_PROJECT_ENV=1 <absolute-loaf> install --to amp --yes`.
8. Run `<absolute-loaf> harness readiness --target amp --pin .agents/loaf-orb.pin --archive-sha256 <verified-checksum>`.

Setup and resume capture their inherited `PATH` and `LOAF_PROJECT_ENV` before changing anything. They first prove internal state using the absolute pinned executable and a controlled subprocess environment. If internal state is healthy but the captured Amp project environment is wrong, they fail with an actionable environment error and do not download. If runtime or installed content is missing or damaged, they take the PID-owned lock, repair only from the verified archive, run the deterministic install, prove internal readiness, and finally prove the originally inherited agent environment. Lifecycle overlap can therefore never masquerade as agent readiness.

`agent-check` is the plugin barrier: it is read-only, takes no lock, performs no download or install, and never repairs. It requires the committed safe pin, verified receipt/platform/manifest, the exact project-owned user-bin symlink, `LOAF_PROJECT_ENV=1`, and `command -v loaf` resolving through the inherited `PATH` to the pinned binary before it invokes absolute `loaf harness readiness` with the verified digest. Its refusals avoid printing machine paths or captured command output.

A ready resume is local, but Amp gives resume only an approximately ten-second courtesy window; resume is not a barrier and may finish after the agent starts. The plugin's `agent-check`, not resume completion, gates tools. Snapshot restore may skip `.agents/setup` entirely, and edits to setup or the bootstrap do not affect existing snapshots until the operator deletes and recreates them. Keep the Amp project's optional pre-clone and pre-setup command fields empty; `.agents/setup` remains the committed lifecycle setup entry point. Amp Desktop does not use the Orb resume lifecycle and is irrelevant to this contract.

A failed internal check acquires the project prefix's atomic `.bootstrap.lock`, records its PID owner, and reruns readiness once under the lock so a concurrent finisher avoids a duplicate download. A live owner is refused. A dead PID proves an orphaned lock, which the next process retires narrowly before acquiring its own lock; malformed or unexpectedly populated lock directories are refused rather than removed broadly. Only the current lock holder may reacquire, publish, activate, or install. A receipt match alone never authorizes repair from existing bytes, and there is no unpinned repair path.

The release URL can be overridden only by setting both `LOAF_ORB_ISOLATED_TESTING=1` and `LOAF_ORB_TEST_RELEASE_BASE_URL` in a disposable test environment. Never set those variables on a real Orb.

## Upgrade a project

Upgrade by one reviewed pin and generated-content commit:

1. Change `version` and both architecture checksums together.
2. Confirm the checksums against the assets of that exact official release.
3. Invoke the new pinned absolute executable with `LOAF_PROJECT_ENV=1` and `install --to amp --yes` in a clean materialization environment.
4. Review and commit the resulting project-local plugin, skills, managed `AGENTS.md` fence, skill/target ownership manifests, and `.agents/loaf/install-targets/amp.json` together with the pin update.
5. Require `git status --porcelain` to be empty before deleting/recreating Orb snapshots or starting dogfood.

The versioned prefix leaves the prior release available for rollback. Reverting the pin selects that version again; if its verified receipt is absent, setup reacquires and verifies it from the official release.

## Provenance and secrets boundary

The committed pin is the project's reviewable provenance root. The bootstrap uses a fixed official release origin, verifies the selected archive digest before extraction, requires `loaf-release-manifest.json`, writes the verified lowercase digest plus a newline to `.loaf-archive.sha256` beside the installed release, and reads that sidecar back when invoking readiness. The sidecar must be a non-symlink regular file containing exactly one lowercase digest line. A missing, malformed, pin-mismatched, or readiness-rejected sidecar forces fresh checksum-verified acquisition before install. The absolute executable and its installed distribution, not `PATH` discovery or the checkout, supply Loaf content.

Inside Amp's setup or resume lifecycle, readiness asks `amp orb id-token` for a short-lived project-scoped token and reads only its Amp `project_id` and optional `workspace_id` claims. It never persists or prints the token, and it ignores user, thread, and email claims. Outside an Orb lifecycle environment those Amp IDs are omitted with an explicit limitation; no identity is guessed. The committed `project.id` remains Loaf's continuity project identity and is intentionally distinct from Amp's project identifier.

Neither the pin nor the scripts authenticate to GitHub or the selected tracker. Keep provider tokens, GitHub credentials, Amp credentials, and any Loaf connection material in the Orb secret store or harness-managed connection. Do not embed credentials in `git.remote`, `tracker.scope`, `LOAF_ORB_HOME`, or the release URL override.

Private synchronization and attach are outside this bootstrap/readiness contract. Do not add sync credentials or attach behavior to these lifecycle scripts.

## Preserve consumer-owned instructions

The bootstrap never writes `AGENTS.md` itself. The absolute `loaf install --to amp --yes` invocation owns only Loaf's existing fenced section:

```text
<!-- loaf:managed:start -->
...
<!-- loaf:managed:end -->
```

Consumer prose outside that fence remains consumer-owned and must be preserved byte-for-byte. A single well-formed managed fence may be updated in place. Duplicate managed fences, malformed fence boundaries, or ambiguous legacy instruction sources must refuse rather than overwrite or choose a source silently. Resolve the ambiguity in the consumer repository, then rerun setup.

The complete deterministic install output is reviewed source in the consumer repository: `.amp/plugins/`, `.amp/.loaf-managed-target.json`, `.agents/skills/`, `.agents/skills/.loaf-managed-skills.json`, `.agents/loaf/install-targets/amp.json`, and the managed `AGENTS.md` fence. Commit it before Orb dogfood and start the Orb from a clean Git status. These bytes are derived from the pinned release and prove what Amp may load; they are not a tracker, runtime authority, or second work record.

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
HOME="$proof_home" XDG_CONFIG_HOME="$proof_home/.config" LOAF_ORB_HOME="$orb_home" LOAF_BIN_DIR="$proof_root/user-bin" LOAF_PROJECT_ENV=1 PATH="$proof_root/user-bin:$PATH" .agents/setup
grep -F 'consumer instructions: preserve this line' AGENTS.md
git add AGENTS.md .amp .agents/skills .agents/loaf/install-targets/amp.json
git commit -m 'Materialize pinned Loaf Amp integration'
test -z "$(git status --porcelain)"
HOME="$proof_home" XDG_CONFIG_HOME="$proof_home/.config" LOAF_ORB_HOME="$orb_home" LOAF_BIN_DIR="$proof_root/user-bin" LOAF_PROJECT_ENV=1 PATH="$proof_root/user-bin:$PATH" .agents/resume
HOME="$proof_home" XDG_CONFIG_HOME="$proof_home/.config" LOAF_ORB_HOME="$orb_home" LOAF_BIN_DIR="$proof_root/user-bin" LOAF_PROJECT_ENV=1 PATH="$proof_root/user-bin:$PATH" .agents/loaf-orb-bootstrap.sh agent-check
test -z "$(git status --porcelain)"
```

Expected evidence:

- Setup downloads one official archive, rejects any checksum mismatch before extraction, requires the manifest, installs beneath `$orb_home/releases/<version>`, activates `$proof_root/user-bin/loaf` to the exact pinned binary, preserves the sentinel, and proves internal plus inherited-agent readiness.
- The first setup materializes deterministic project integration for review and commit; it is preparation, not clean-Orb dogfood. Dogfood begins only from the clean commit containing those generated bytes.
- Resume invokes internal readiness first and performs no download when ready, but `agent-check` remains the tool barrier. An offline second resume plus `agent-check` is a useful proof of that property.
- Changing one checksum to another valid-looking 64-character digest makes fresh setup fail before extraction.
- Adding a second managed fence or breaking one fence boundary makes install fail loudly without replacing consumer prose. Readiness also refuses a legacy name-only project skill ownership manifest, a foreign project-local Loaf plugin, or any duplicate effective Loaf skill/plugin source under the disposable global homes (`$HOME/.agents/skills`, `$HOME/.config/agents/skills`, `$XDG_CONFIG_HOME/amp/plugins`, or `$HOME/.amp/plugins`). Remove the legacy or duplicate source before continuing.
- Replacing the pin with an unknown key, duplicate key, authenticated URL, path-shaped identity, uppercase checksum, or placeholder makes bootstrap fail before network access.

For a local archive-server fixture, use the two isolated-testing variables and reproduce the official `/download/v<version>/<archive>` path beneath the fixture base URL. That override is test-only evidence and must never appear in committed consumer configuration.
