---
name: commit-pr
description: Create a pull request with full CI validation
disable-model-invocation: false
---

# Create Pull Request

Create a GitHub pull request following project conventions with mandatory pre-PR checks.

## Language Rules

- **PR title**: Always in English
- **PR description**: Always in English
- **Commit messages**: Always in English

## Mandatory Pre-PR Checks

**NEVER skip these steps. All must pass before creating the PR.**

1. **Format & Vet**:
   ```bash
   make fmt && make vet
   ```

2. **Lint**:
   ```bash
   make lint
   ```

3. **Unit & Integration Tests**:
   ```bash
   make test
   ```

If any check fails, fix the issues before proceeding. Do NOT create the PR with failing checks.

## PR Creation Workflow

1. **Inspect changes**: Run `git status`, `git diff`, and `git log` to understand all changes since branching from main
2. **Run all checks**: Execute the mandatory pre-PR checks above
3. **Stage and commit**: If there are uncommitted changes, create a commit following `/commit` conventions
4. **Push branch**: Push to remote with `-u` flag
5. **Create PR**: Use `gh pr create` with the format below

## PR Format

```bash
gh pr create --title "<imperative mood title, max 70 chars>" --body "$(cat <<'EOF'
## Summary
- <bullet point 1: what changed and why>
- <bullet point 2: what changed and why>

## Test plan
- [ ] Unit tests pass (`make test`)
- [ ] Lint passes (`make lint`)
- [ ] <additional test items specific to this change>
EOF
)"
```

## PR Title Examples

- `Add Prometheus monitoring sidecar injection`
- `Fix rolling restart when config hash changes`
- `Update webhook validation for CE namespace limit`
- `Refactor StatefulSet reconciliation to support multi-rack`

## Test Plan Checkbox Rules

- Only mark checkboxes as done (`[x]`) if the tests actually ran and passed during this session
- If tests were not run, leave as unchecked (`[ ]`)
- Add specific test items relevant to the change:
  - API/webhook changes: `- [x] Webhook validation tests pass`
  - Controller changes: `- [x] envtest integration tests pass`
  - E2E impact: `- [ ] E2E tests pass (requires Kind cluster)`

## CI Workflows Reference

The following CI checks will run on the PR:
- **test.yml**: `make test` (unit + envtest)
- **lint.yml**: golangci-lint v2.8.0
- **test-e2e.yml**: `make test-e2e` (Kind cluster, Ginkgo v2)
