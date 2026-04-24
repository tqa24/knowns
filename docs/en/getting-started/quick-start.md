# Quick Start

This is the fastest path to a working Knowns project.

## 1. Initialize the project

```bash
knowns init
# or, without a global install:
npx knowns init
```

The init flow can configure:

- project name
- git tracking mode
- AI platforms to integrate
- optional OpenCode-powered chat UI
- semantic search
- embedding model

If the terminal is too narrow, Knowns may fall back to non-interactive defaults instead of rendering the full wizard.

## 2. Create a task

```bash
knowns task create "Setup project" -d "Initialize project with Knowns"
```

## 3. Create a document

```bash
knowns doc create "Architecture" -d "System overview" -f architecture
```

## 4. Open the browser UI

```bash
knowns browser --open
```

## 5. Sync generated artifacts when needed

```bash
knowns sync
knowns update
```

Use `knowns sync` after cloning, after changing selected platforms, or after updating the CLI.

## 6. Open the browser UI again later

```bash
knowns browser --open
```

## Related

- [First project](./first-project.md)
- [User guide](../guides/user-guide.md)
- [Workflow](../guides/workflow.md)
