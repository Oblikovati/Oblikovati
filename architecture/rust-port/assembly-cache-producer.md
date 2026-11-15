# Assembly cache producer identity

An assembly cache is a display artifact. Its producer digest is the SHA-256
digest of the desktop executable that generated the triangles. Cache version 2
requires this field. Version 1 caches cannot establish their producer and are
refused for Express open; a normal open and save can create a new cache.

Each process computes the digest from its executable and retains the result for
its scene owner. Linux reads `/proc/self/exe` so the digest refers to the inode
that the process is executing even if its original path was replaced. On macOS
and Windows, the host refuses to create or open an Express cache until it can
verify the running image. Normal full opening remains available. If the running
executable cannot be read, the host reports that the cache is unavailable. It
must not substitute a version string, a source revision, or an empty identity.

The digest is an identity for a particular binary, not a stable file format or
a promise of compatibility between builds. A rebuilt executable invalidates
old caches even when it has equivalent behavior. A CLI and GUI executable can
share a saved view only when their executable digests match. The identity covers
statically linked code, including the Martensite dependency. It does not attest
to external libraries loaded at run time.
