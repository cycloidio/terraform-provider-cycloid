<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **terraform-provider-cycloid** (3240 symbols, 6071 relationships, 141 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## When GitNexus is Available

- Before modifying a symbol, prefer running `gitnexus_impact({target: "symbolName", direction: "upstream"})` to understand callers and blast radius.
- Before committing, consider running `gitnexus_detect_changes()` to verify the affected scope matches expectations.
- If impact analysis returns HIGH or CRITICAL risk, surface it to the user before proceeding.
- When exploring unfamiliar code, prefer `gitnexus_query({query: "concept"})` over grepping — it returns process-grouped results ranked by relevance.
- For full context on a specific symbol — callers, callees, execution flows — use `gitnexus_context({name: "symbolName"})`.
- Prefer `gitnexus_rename` over find-and-replace for symbol renames — it understands the call graph.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/terraform-provider-cycloid/context` | Codebase overview, check index freshness |
| `gitnexus://repo/terraform-provider-cycloid/clusters` | All functional areas |
| `gitnexus://repo/terraform-provider-cycloid/processes` | All execution flows |
| `gitnexus://repo/terraform-provider-cycloid/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
