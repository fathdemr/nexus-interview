---
inclusion: always
---

# Git Workflow Rules

## Agent Boundaries

The agent **never** commits, pushes, or creates pull requests autonomously.
All git operations that change history or affect the remote are the user's responsibility.

Specifically, the agent must not run:
- `git commit`
- `git push`
- `git merge`
- `git rebase`
- `git tag`
- `gh pr create` or any equivalent PR/MR creation command

When a task is complete, the agent summarizes what changed and which files were modified,
then explicitly prompts the user to review and commit.

---

## Branch Naming

Every feature, fix, or improvement is developed on its own branch.
Branch names follow the pattern:

```
<type>/<short-description>
```

### Types

| Type | When to use |
|---|---|
| `feat` | New feature or capability |
| `fix` | Bug fix |
| `perf` | Performance improvement |
| `refactor` | Code restructuring without behavior change |
| `chore` | Tooling, dependencies, config changes |
| `docs` | Documentation only |
| `test` | Adding or updating tests |

### Examples

```
feat/candidate-invite-flow
fix/jwt-clock-skew-validation
perf/interview-query-index
refactor/auth-middleware-extract-helpers
chore/update-gorm-driver
docs/api-endpoint-reference
```

Rules:
- Use lowercase and hyphens only — no underscores, no slashes beyond the type prefix.
- Keep the description concise (3–5 words max).
- Branch from `main` (or the agreed base branch) unless the task explicitly builds on another branch.

---

## Commit Message Format

Commits follow the Conventional Commits specification:

```
<type>(<scope>): <short summary>

[optional body — explain why, not what]
```

### Examples

```
feat(candidate): add magic link invitation flow
fix(auth): handle clock skew on iat validation
perf(interview): add composite index on candidate_id + status
refactor(middleware): extract token claim helpers into tokenhelper package
```

Rules:
- Summary line: imperative mood, lowercase, no period, max 72 characters.
- Scope is the module or package name (`candidate`, `auth`, `interview`, `speech`, `ai`…).
- Body is optional — add it when the reason behind the change is not obvious from the summary.
- One logical change per commit. Do not bundle unrelated changes.

---

## Agent Handoff Checklist

When the agent finishes a task it must output:

1. A summary of what was implemented or changed.
2. A list of modified/created files.
3. A suggested branch name for this work.
4. A suggested commit message following the format above.
5. A reminder that the user should review, stage, and commit the changes.

Example handoff:

```
Done. Here's what changed:

Modified files:
- internal/candidate/service.go
- internal/candidate/repository.go
- internal/candidate/handler.go

Suggested branch:  feat/candidate-invite-flow
Suggested commit:  feat(candidate): add magic link invitation flow

Review the diff, then commit when ready.
```
