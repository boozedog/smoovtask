# Franken UI Contexts (Local Snapshot)

This directory keeps a local copy of the upstream Franken UI context files so
we can reference them while working without leaving the repo.

## Source

- Upstream repo: `https://github.com/franken-ui/contexts`
- Snapshot path: `docs/franken-ui/contexts/`
- Snapshot commit: `bb57524487d2cf16c78d44337c0c62e543455229`

## Refresh

Run the unified vendor script:

```bash
just vendor docs    # context docs only
just vendor         # everything (CSS/JS + docs)
```

See `scripts/vendor.sh` for version pins and file mappings.

## Notes

- This is a vendored snapshot, not a git submodule.
- Keep edits to this folder limited to upstream syncs and local indexing notes.
