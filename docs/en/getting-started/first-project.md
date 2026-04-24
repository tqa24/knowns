# First Project

After `knowns init`, a good first setup usually includes four things:

1. create a task
2. create one or two documents
3. confirm search works
4. connect the AI runtime(s) you actually use

## Example session

```bash
knowns task create "Add authentication" \
  -d "JWT-based auth with login and register endpoints" \
  --ac "User can register with email/password" \
  --ac "User can login and receive JWT token"

knowns doc create "Auth Architecture" \
  -d "Authentication design decisions" \
  -f architecture

knowns search "authentication" --plain
knowns validate --plain
knowns browser --open
```

## Why these steps matter

- the task gives AI a concrete execution target
- the doc gives AI structured context instead of ad-hoc chat explanations
- search confirms local retrieval is working
- validation checks that the basic project structure is sound

## Suggested next reads

- [User guide](../guides/user-guide.md)
- [MCP integration](../guides/mcp-integration.md)
