# Issue tracker: GitHub

Issues and PRDs for this repo live as GitHub issues on `010CodingCollective/postgres-backup`. Use the `gh` CLI for all operations.

## Conventions

- **Create an issue**: `gh issue create --title "..." --body "..."`. Use a heredoc for multi-line bodies.
- **Read an issue**: `gh issue view <number> --comments`, filtering comments by `jq` and also fetching labels.
- **List issues**: `gh issue list --state open --json number,title,body,labels,comments --jq '[.[] | {number, title, body, labels: [.labels[].name], comments: [.comments[].body]}]'` with appropriate `--label` and `--state` filters.
- **Comment on an issue**: `gh issue comment <number> --body "..."`
- **Apply / remove labels**: `gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **Close**: `gh issue close <number> --comment "..."`

Infer the repo from `git remote -v` — `gh` does this automatically when run inside a clone.

## Local `gh` version caveat

The `gh` CLI on this machine is **v2.4.0**, which predates the `gh label` subcommand. All the commands listed above work, but anything that manages the label vocabulary itself must go through the REST API instead:

```bash
# List labels
gh api repos/010CodingCollective/postgres-backup/labels --jq '.[].name'

# Create a label
gh api repos/010CodingCollective/postgres-backup/labels \
  -f name="some-label" -f color="ededed" -f description="..."
```

If `gh` is upgraded past v2.9, prefer `gh label list` / `gh label create` and delete this section.

## When a skill says "publish to the issue tracker"

Create a GitHub issue.

## When a skill says "fetch the relevant ticket"

Run `gh issue view <number> --comments`.
