# Releasing Guide

Use this checklist for versioned releases of `github.com/deccara-tech/pkg-logging`.

## Versioning Rules

Semantic versioning:
- `v0.x.y`: pre-1.0, may include breaking changes
- `v1.x.y+`: stable API guarantees

Bumps:
- patch (`z`): bugfix, no contract break
- minor (`y`): backward-compatible additions
- major (`x`): breaking API/schema behavior

## Pre-Release Checklist

1. Ensure tests pass:
```bash
rtk go test ./...
```

2. Ensure docs and changelog updated:
- `CHANGELOG.md` updated
- `README.md` / `docs/integration.md` updated if behavior changed

3. Validate module path and imports:
- module should remain `github.com/deccara-tech/pkg-logging`

## Tag + Push

Example for `v0.1.1`:

```bash
rtk git add -A
rtk git commit -m "release: v0.1.1"
rtk git tag -a v0.1.1 -m "v0.1.1"
rtk git push origin main
rtk git push origin v0.1.1
```

## Post-Release Verification

From clean environment or another repo:

```bash
go get github.com/deccara-tech/pkg-logging@v0.1.1
go test ./...
```

Optional check:

```bash
go list -m -json github.com/deccara-tech/pkg-logging@v0.1.1
```

## Breaking Release (v2+)

If introducing breaking change:

1. Update `go.mod` module path with major suffix:
- `module github.com/deccara-tech/pkg-logging/v2`

2. Update imports in code and docs:
- `github.com/deccara-tech/pkg-logging/v2/observe`

3. Tag release:
- `v2.0.0`

## Release Notes Template

Copy into release body:

```text
## Summary
- <one-line release summary>

## Added
- ...

## Changed
- ...

## Fixed
- ...

## Breaking Changes
- none / list here
```
