# postgres-backup

A Go service that dumps PostgreSQL databases and ships the archives to object storage.

## Conventions

- Use `slog` for logging.
- Ask questions if a requirement is ambiguous rather than guessing.

## Agent skills

### Issue tracker

Issues live in GitHub Issues on `010CodingCollective/postgres-backup`, managed with the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

The five canonical triage roles use their default label strings (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`), all of which exist on the repo. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `CONTEXT.md` and one `docs/adr/` at the repo root. See `docs/agents/domain.md`.
