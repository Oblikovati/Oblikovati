# Martensite v0.15.0 on rust-work

The application pins the geometry and renderer toolkits to the published
Martensite v0.15.0 commit `db01546bb032659682715c6120e628b95fe5773d`.
The lock resolves all 20 directly and transitively consumed Martensite
packages from that revision. Vulkan advances to 0.8.2. The release contains
the upstream correction for the independent ellipse oracle.

The release preflight verified the published dependency tag and crate
archives. The packaging portion stopped at the missing application icon in
the local source checkout. Cargo audit reported the same advisory IDs for
the previous and updated lockfiles.

[Release](https://github.com/Oblikovati/Martensite/releases/tag/v0.15.0).
[Ellipse correction](https://github.com/Oblikovati/Martensite/commit/5f4e6bf86).
[Verification output](tests.txt).
