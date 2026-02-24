---
description: How to create a new release for Claude Telex
---

## Release Workflow

### 1. Ensure all changes are committed and pushed to `main`

// turbo
```
git status
```

### 2. Write the changelog

Write the changelog following [Keep a Changelog](https://keepachangelog.com/) format. Only include functional changes — no doc-only updates.

Categories (use only the ones that apply):

- **Added** — New features
- **Changed** — Changes to existing functionality
- **Fixed** — Bug fixes
- **Build** — Build system or CI changes
- **Removed** — Removed features
- **Security** — Security fixes

Each entry should be a single clear sentence describing what changed and why it matters.

**Example:**

```
v1.0.1

Added
- Version-based patch detection for Claude Code (automatically checks every 5 minutes)
- Tray icon indicator (turns blue when a new Claude Code version requires re-patching)

Fixed
- Fixed Linux build for Ubuntu 22.04+ by switching to libayatana-appindicator3-dev
```

### 3. Create the annotated tag

Replace `VERSION` with the new version (e.g. `v1.0.1`) and `CHANGELOG` with the changelog from step 2:

```
git tag -a VERSION -m "CHANGELOG"
```

### 4. Push the tag to trigger CI

// turbo
```
git push origin VERSION
```

This triggers the GitHub Actions release workflow (`.github/workflows/release.yml`) which:
- Builds binaries for Windows x86, Windows x64, macOS, and Linux
- Packages them as `.zip` (Windows) or `.tar.gz` (Unix)
- Creates a GitHub Release with checksums

### 5. Verify the release

Open the GitHub Actions page to confirm all builds pass:
```
https://github.com/nguyenhx2/Claude-Telex/actions
```

Then check the release page:
```
https://github.com/nguyenhx2/Claude-Telex/releases/tag/VERSION
```

### Re-tagging (if needed)

If a tag needs to be recreated (e.g. changelog typo, missing commit):

```
git tag -d VERSION
git push origin :refs/tags/VERSION
```

Then repeat from step 3.
