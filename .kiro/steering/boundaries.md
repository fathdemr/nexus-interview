---
inclusion: always
---

# Agent Boundaries

Hard limits on what the agent is allowed to do in this project.

---

## AWS — Strictly Off Limits

The agent must not provision, configure, or interact with any AWS resource.
This includes but is not limited to:

- Running `aws` CLI commands
- Writing or executing AWS SDK calls (any language)
- Creating, modifying, or deleting AWS resources (RDS, S3, ECS, SES, IAM, VPC, ALB, ECR, CodePipeline, CloudWatch…)
- Generating or suggesting Terraform / CloudFormation / CDK code
- Writing `.aws/credentials`, `~/.aws/config`, or any IAM policy document

All AWS infrastructure is set up and managed manually by the user.
The agent's job is to write application code that **consumes** AWS services via environment variables (connection strings, API keys, region) — never to manage the infrastructure itself.

**When an AWS resource is needed** (e.g. an S3 bucket URL, an RDS connection string), the agent stops, states what is needed, and waits for the user to provide it.

---

## Frontend — Backend Must Be Complete First

The agent does not work on frontend code until the backend is declared complete by the user.

Scope while backend is active:
- Go source code under `cmd/` and `internal/`
- Shared packages under `pkg/`
- `Dockerfile` and `deployments/`
- `docs/` (API documentation)

Out of scope until backend is complete:
- Anything under a `frontend/`, `web/`, or `app/` directory
- Next.js / React components, pages, or styles
- Any `package.json` or Node.js tooling

---

## Git — No Autonomous Commits or Pushes

See `git-workflow.md` for the full rules.
Short version: the agent never runs `git commit`, `git push`, or creates pull requests.
All version control operations are performed by the user.
