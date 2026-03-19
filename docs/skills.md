# Skills

Skills are reusable AI workflows bundled with Knowns.

---

## Built-in Skills

Knowns currently ships these built-in skills:

| Skill | Trigger | Purpose |
| ----- | ------- | ------- |
| `kn-init` | `/kn-init` | Read project context and current state |
| `kn-plan` | `/kn-plan` | Create an implementation plan |
| `kn-research` | `/kn-research` | Research docs and code before changes |
| `kn-implement` | `/kn-implement` | Execute an approved task |
| `kn-verify` | `/kn-verify` | Validate work and report coverage |
| `kn-spec` | `/kn-spec` | Create a spec document |
| `kn-template` | `/kn-template` | Work with templates |
| `kn-extract` | `/kn-extract` | Extract reusable knowledge |
| `kn-doc` | `/kn-doc` | Create or update documentation |
| `kn-commit` | `/kn-commit` | Commit changes safely |

---

## CLI Commands

The current CLI supports:

```bash
knowns skill list
knowns skill view <name>
knowns skill sync
```

- `knowns skill list` shows available skills
- `knowns skill view <name>` shows a skill definition
- `knowns skill sync` syncs skills from imported packages

Top-level project instruction syncing is handled by `knowns sync`, not by extra `knowns skill` subcommands.

---

## Where Skills Live

Built-in skill files are typically synced into Claude-compatible folders such as:

```text
.claude/skills/
  kn-init/
    SKILL.md
  kn-plan/
    SKILL.md
  ...
```

The exact synced output depends on the active platforms and sync implementation in the current CLI build.

---

## Typical Workflow

Common flow when working with tasks:

```text
/kn-init -> /kn-research -> /kn-plan -> /kn-implement -> /kn-verify -> /kn-commit
```

This mirrors the Knowns workflow of reading context first, planning before implementation, validating before completion, and only then committing.

---

## Imported Skills

Imported packages may provide additional skills.

```bash
knowns import add <source>
knowns import sync
knowns skill sync
```

`knowns import sync` currently exists, but the CLI notes that full network sync behavior is not yet implemented in this build.

---

## Custom Skills

You can add custom skill folders alongside synced skills, but the current CLI does not provide `knowns skill create`.

If you add custom skills manually, keep them separate from generated content so future syncs are easier to manage.

---

## Related

- [Guidelines](./guidelines.md) - How full guidance is exposed
- [Multi-Platform](./multi-platform.md) - Platform targets and sync behavior
- [Command Reference](./commands.md) - Current CLI syntax
