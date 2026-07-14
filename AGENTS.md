# Agent Instructions

## Bump Go Modules, Open PR, Merge, and Release

When asked to bump/update Go modules, follow these steps:

### 1. Update modules

```sh
git checkout main && git pull
go get -u ./...
go mod tidy
go build ./...
```

All three commands must succeed before proceeding.

### 2. Create branch, commit, and open PR

- Branch name: `chore/bump-go-packages`
- Commit message format: `chore: bump go packages` with a body listing each upgraded module (e.g. `- golang.org/x/crypto v0.50.0 => v0.54.0`)
- Push and open a PR with the same title. Include the module list in the PR body.

```sh
git checkout -b chore/bump-go-packages
git add go.mod go.sum
git commit -m "chore: bump go packages ..."
git push -u origin HEAD
gh pr create --title "chore: bump go packages" --body "..."
```

### 3. Squash merge the PR

```sh
gh pr merge <number> --squash --delete-branch
```

### 4. Tag and release

Releases are driven by pushing a semver tag. The goreleaser GitHub Actions workflow (`.github/workflows/goreleaser.yaml`) triggers on `v*` tags and builds binaries automatically.

- Check the latest existing tag: `git tag --sort=-v:refname | head -1`
- Increment the **patch** version (e.g. `v0.7.3` → `v0.7.4`). Use minor/major bump only if explicitly requested.
- Create an **annotated** tag and push it:

```sh
git tag -a v0.X.Y -m "v0.X.Y"
git push origin v0.X.Y
```

- Wait for the goreleaser workflow to complete (~2 minutes), then verify:

```sh
gh run list --limit 1
gh release view v0.X.Y
```

### Conventions

- Versioning: semver (`vMAJOR.MINOR.PATCH`), default to patch bump for dependency updates.
- Commit style: conventional commits (`chore:`, `feat:`, `fix:`, etc.).
- The goreleaser workflow `go-version` in `.github/workflows/goreleaser.yaml` may need updating if the Go version in `go.mod` was bumped beyond what's set there.
