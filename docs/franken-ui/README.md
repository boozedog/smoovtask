# Franken UI Contexts (Local Snapshot)

This directory keeps a local copy of the upstream Franken UI context files so
we can reference them while working without leaving the repo.

## Source

- Upstream repo: `https://github.com/franken-ui/contexts`
- Snapshot path: `docs/franken-ui/contexts/`
- Snapshot commit: `bb57524487d2cf16c78d44337c0c62e543455229`

## Refresh

Run:

```bash
sh docs/franken-ui/update-contexts.sh
```

This re-clones the upstream repo into a temp directory, copies `*.md` files into
`docs/franken-ui/contexts/`, removes nested git metadata, and prints the new
snapshot commit.

## Notes

- This is a vendored snapshot, not a git submodule.
- Keep edits to this folder limited to upstream syncs and local indexing notes.
