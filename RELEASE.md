# Release Process

This repository uses GitHub Actions to automatically build binaries for all platforms and attach them to releases.

## How to Create a Release

1. **Commit and push your changes** to the main branch:
   ```bash
   git add .
   git commit -m "Your commit message"
   git push origin main
   ```

2. **Create and push a version tag**:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

3. **GitHub Actions will automatically**:
   - Build binaries for Windows, Linux, macOS (Intel), and macOS (ARM64)
   - Create a new release with your tag
   - Attach all binaries as downloadable assets
   - Generate release notes from your commits

## Binaries Built

The workflow builds the following binaries:
- `daily-task.exe` - Windows (amd64)
- `daily-task-linux` - Linux (amd64)
- `daily-task-mac` - macOS Intel (amd64)
- `daily-task-mac-arm64` - macOS Apple Silicon (arm64)

## Manual Triggering

You can also manually trigger a build without creating a release by going to:
- Actions tab → Build and Release → Run workflow

This will build the binaries and upload them as artifacts (but won't create a release).

## Version Numbering

Follow semantic versioning (semver):
- `v1.0.0` - Major release (breaking changes)
- `v1.1.0` - Minor release (new features, backward compatible)
- `v1.0.1` - Patch release (bug fixes)

## First Release

For your first release:
```bash
git tag v1.0.0
git push origin v1.0.0
```

Then visit: https://github.com/maximedelvaux/daily-cli/releases
