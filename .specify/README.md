# Specs for this repository

The specification set for this project is not duplicated here. Authoritative
sources, in reading order:

- `CONTEXT.md` — domain vocabulary, system seams, testing boundaries.
- `docs/superpowers/specs/` — the design spec for the admin/MCP/Lab publishing
  system, and `docs/superpowers/plans/` for the implementation plans behind it.
- `docs/security/admin-mcp-lab-threat-model.md` — trust boundaries and the
  mitigations each boundary depends on.
- `docs/runbooks/` — deployment and rollback procedures.
- `docs/analysis/` — dated review reports (findings, verification status, limits).
- `AGENTS.md` — invariants, commands, and state facts an agent must respect.

This directory exists so the pre-push quality gate recognises the repository as
spec-separated without a second copy of the same documents drifting away from
the files above.
