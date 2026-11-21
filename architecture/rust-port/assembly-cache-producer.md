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

## Windows and macOS investigation (#3964)

The desktop owner and its producer code currently compile only on Linux. A
producer implementation alone will not enable Express on macOS or Windows.
Those platforms need the native desktop port as well as the checks below.

Do not hash `current_exe()` by pathname without proving that the opened file is
the process image. Windows can rename a running executable and put a different
file at its former name. The Go toolchain encountered exactly this race when
replacing running executables: [Go issue 21997](https://github.com/golang/go/issues/21997).
An atomic replacement on macOS creates the same concern. An open file handle
must be checked against the running image before its contents are accepted.

On Windows, Microsoft documents that an executable is backed by an image
mapping rather than a file handle and that the file system prevents writes
to the mapped image. See [Executable Images](https://learn.microsoft.com/en-us/windows-hardware/drivers/ifs/executable-images).
The process-information API is explicitly internal, may change, and must be
looked up at runtime if used; see [NtQueryInformationProcess](https://learn.microsoft.com/en-us/windows/win32/api/winternl/nf-winternl-ntqueryinformationprocess).
Its public documentation does not specify an image-to-file identity query.
An implementation must establish a supported identity query or make an
explicit compatibility decision for an internal API. It must fail closed if
the query is unavailable or does not establish identity.

On macOS, the [XNU process information header](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/sys/proc_info.h)
defines region information that includes vnode metadata and a path. This is
a possible investigation point, not proof that reopening the path gives the
mapped executable. An implementation must establish that the queried region
belongs to the actual process image, compare the image and held file identities,
and establish whether the file can change while it is hashed. It must decline
if any of these conditions cannot be established.

For each platform, a native regression must start an independently built test
executable, replace or rename its original path while it runs, and verify that
the producer check either identifies the original image or refuses the cache.
Also cover an unreadable image, a mismatched image, and two builds with distinct
geometry behavior. Cross-compilation and an emulated Windows run do not prove
the native operating-system behavior. Until those checks pass, retain the
existing named decline and keep ordinary full opening available.
