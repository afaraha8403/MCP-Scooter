# Releasing MCP Scooter

This guide explains how to create releases for MCP Scooter using the automated GitHub Actions workflow.

## Release Channels

MCP Scooter supports two release channels:

| Channel | Tag Format | Example | Who Gets It |
|---------|------------|---------|-------------|
| **Stable** | `vX.Y.Z` | `v1.0.0`, `v1.2.3` | All users |
| **Beta** | `vX.Y.Z-beta.N` | `v1.0.0-beta.1` | Users with "Include Beta Releases" enabled |

Other prerelease formats also work: `v1.0.0-alpha.1`, `v1.0.0-rc.1`

## Quick Reference

### Release a Beta Version

```bash
# 1. Make sure all changes are committed and pushed
git push origin main

# 2. Create and push the tag
git tag -a v0.0.2-beta.1 -m "Beta release v0.0.2-beta.1"
git push origin v0.0.2-beta.1
```

### Release a Stable Version

```bash
# 1. Make sure all changes are committed and pushed
git push origin main

# 2. Create and push the tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## What Happens After You Push a Tag

1. **GitHub Actions triggers** the Release workflow
2. **Tests run** (Go tests + registry validation)
3. **Builds start** in parallel for:
   - Windows (x64)
   - macOS (Intel x64)
   - macOS (Apple Silicon ARM64)
   - Linux (x64)
4. **Draft release created** with all installers attached
5. **Updater manifests updated** (`latest.json` for stable, `beta.json` for beta)

## After the Build Completes

1. Go to [GitHub Releases](https://github.com/mcp-scooter/scooter/releases)
2. Find your **draft** release
3. Review the release notes and installers
4. Click **"Publish release"** to make it public

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** (1.0.0 → 2.0.0): Breaking changes
- **MINOR** (1.0.0 → 1.1.0): New features, backwards compatible
- **PATCH** (1.0.0 → 1.0.1): Bug fixes

For prereleases:
- `v1.0.0-alpha.1` → Early development, unstable
- `v1.0.0-beta.1` → Feature complete, testing phase
- `v1.0.0-rc.1` → Release candidate, final testing

## Updating Version Numbers

Before creating a release tag, update the version in these files:

1. `desktop/src-tauri/tauri.conf.json` → `"version": "X.Y.Z"`
2. `desktop/package.json` → `"version": "X.Y.Z"`
3. `desktop/src-tauri/Cargo.toml` → `version = "X.Y.Z"`

Or use the Makefile shortcut (if available):
```bash
make release-beta  # Interactive prompt for version
make release       # Interactive prompt for stable version
```

## Troubleshooting

### Build Failed

1. Check the [Actions tab](https://github.com/mcp-scooter/scooter/actions) for error logs
2. Common issues:
   - Go tests failing → Fix tests first
   - Missing dependencies → Check workflow file
   - Rust compilation errors → Check `desktop/src-tauri/`

### Delete a Tag (if needed)

```bash
# Delete local tag
git tag -d v0.0.1-beta.1

# Delete remote tag
git push origin --delete v0.0.1-beta.1
```

### Re-run a Failed Build

1. Go to the failed workflow run in GitHub Actions
2. Click "Re-run all jobs"

## Auto-Updates

Users receive updates based on their settings:

- **"Include Beta Releases" OFF** → Only stable releases (`latest.json`)
- **"Include Beta Releases" ON** → All releases including beta (`beta.json`)

The app checks for updates on startup and periodically while running.

## Files Involved

- `.github/workflows/release.yml` - The release automation workflow
- `desktop/src-tauri/tauri.conf.json` - App version and updater config
- `desktop/src-tauri/src/lib.rs` - Update checking logic
