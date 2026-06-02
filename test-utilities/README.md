# test-utilities

Shared assets and helpers that support the test suites but are not part of the
shipping application — e.g. neutral scene definitions, Blender projects used to
generate renderer ground-truth images, sample documents, and golden fixtures.

Per the testing strategy ([../architecture/testing](../architecture/testing)):

- **Blender oracle assets** (Tier 3): the neutral `scene.json`, `.blend` projects,
  and pinned render settings used to produce full-PBR ground truth. Containerized
  and version-pinned so goldens reproduce.
- **Golden fixtures**: persistence round-trip samples (`.obk` packages) and
  reference images, with a documented **bless** step for intentional changes.
- **Generators**: scripts that (re)produce fixtures from a source of truth, so
  goldens are regenerable rather than hand-maintained.

Keep large binary goldens out of the main history where practical (Git LFS or a
generator) — track the recipe, not just the artifact. Nothing here is imported by
`/source` production code; test code references these paths explicitly.
