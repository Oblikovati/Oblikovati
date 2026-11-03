# Martensite v0.6 host-target handoff

## Current state

The Oblikovati Rust host must consume tagged Martensite packages directly.
`RustyCADKernel` is a temporary dependency. It is not an accepted final host
target.

Martensite v0.6.0 is tagged at commit
`ff285151264bca32674098431aab7edec8ceb549`.

The target policy change is committed on branch `chore/martensite-0-6`:

```text
4d0c23ab Document direct Martensite v0.6 target
```

The branch is clean and has not been pushed. Run the required local gate before
the first push.

## Blocking release surface

The current host imports these packages from `RustyCADKernel`:

- `rustycad-tkv3d` for the native scene and view boundary;
- `martensite-vulkan` for native presentation and GUI composition;
- `rustycad-tkbo` for the profile-region qualification tests.

Martensite v0.6.0 does not publish corresponding direct packages
`martensite-tkv3d`, `martensite-vulkan`, and `martensite-tkbo`.

This blocks a complete removal of `RustyCADKernel`. The model packages needed
for the basic prism path are present in v0.6.0. The UI-driven vertical slice is
not complete without the missing viewport packages.

Martensite issue [#365](https://github.com/Oblikovati/Martensite/issues/365)
tracks this as P0. It requests a tagged release with compatible public crate
names and a host build that has no `RustyCADKernel` dependency.

## Integration locations

The host package pins are in these files:

- `crates/martensite/Cargo.toml` for model, topology, meshing, and profile work;
- `crates/viewport/Cargo.toml` for the native viewport;
- Rust imports below `crates/martensite/` and `crates/viewport/` use the
  `rustycad_*` crate names.

The existing host plan is `architecture/rust-port/martensite.md`. Commit
`4d0c23ab` changes its target statement and records the P0 release block.

## Next actions

1. Wait for a Martensite tag that provides all three blocking packages.
2. Create a fresh migration branch from the current Rust-port baseline.
3. Replace all package pins with one Martensite tag and rename imports from
   `rustycad_*` to `martensite_*`.
4. Update `Cargo.lock` with Cargo. Do not edit it by hand.
5. Run the Martensite adapter, profile-region, and native viewport tests. Run
   the full required gate before the first push.
6. Verify the live UI journey: create Part, sketch rectangle on XY, fully
   constrain it, extrude, edit size, undo/redo, save/reopen, and resolve a
   named face.
7. Remove the temporary RustyCADKernel dependencies only after all checks pass.

## Worktree hygiene

The active port worktree has concurrent changes. The v0.6 documentation work
is isolated at `oblikovati-rust-martensite-0-6`. Do not delete that worktree
until its commit is merged or intentionally superseded. Do not alter the
concurrent worktree while preparing this migration.
