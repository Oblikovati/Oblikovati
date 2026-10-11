# Native add-in library admission — partial R4

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This starts the native host boundary in
[ADR-0002](../decisions/ADR-0002-api-and-addin-host.md), using Go
`head/internal/addinhost/loader.go` and `dl_shared.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The new `oblikovati-addin-host` workspace crate probes real Linux shared
libraries. It resolves required ID, manifest, Activate, Deactivate, Notify and
Free exports, plus optional Automation and version exports. All function types
come from the pinned `oblikovati_api::ffi`. It reports copied identity/manifest,
API version, and whether the optional automation export exists. It does not
activate an add-in or promise that its automation methods work.

Admission requires both version exports, equal major and no newer minor,
including API 0.x. Negative add-in or host versions are refused as malformed
semantic versions. The negative-minor check strengthens the Go gate, which
accepted negative values with a matching major. Errors name the supplied
version and expected range. Missing required symbols, null strings, malformed
UTF-8 identity and invalid paths also produce named declines.

## Native ownership and unsafe scope

The workspace's other crates retain `unsafe_code = forbid`. The native host
crate uses `deny`, with a local allowance only for its Linux boundary and trusted
integration tests. No model or solver crate imports this loader. `probe_library`
is an unsafe entry point: callers must trust constructors, exact export
prototypes and returned string storage. It cannot contain foreign faults.

Linux loading uses libc's platform constants and primitives behind the owned
boundary, with immediate relocation and local symbol visibility. Each lookup
clears and checks `dlerror`; null symbol addresses are not accepted as callable
functions. See the primary [Linux loader documentation](https://man7.org/linux/man-pages/man3/dlsym.3.html).
A private pointer conversion checks the imported function type's size. String
reads check null and copy valid UTF-8 before returning. Valid NUL-terminated
storage remains a caller/library safety obligation, as specified by
[Rust's CStr contract](https://doc.rust-lang.org/stable/std/ffi/struct.CStr.html).

Opened libraries remain mapped for process lifetime, including rejected probes.
Go load-time constructors can start runtime threads before API activation;
unknown libraries therefore have no proven safe unloading point. This is a
conservative application of ADR-0002's residency policy, not a native close or
hot-reload implementation. Returned metadata owns no library pointer or borrow.

## Evidence and remaining work

Eight tests cover the compatibility matrix and real-library metadata. Trusted
Rust fixture libraries import the actual pinned `ffi.rs` source and exercise
optional automation, missing required/version symbols, invalid version values,
null/invalid strings and path failures. The fixed Go echo fixture is built as a
real c-shared library and passes the older-minor handshake. Fixture builds use
isolated temporary directories; Cargo metadata is filtered to the installed
Linux compiler's host so no deferred-platform dependencies are requested.
Go and Rust compiler failures are gate failures, not skipped tests.

Run full local `make gate`, which now discovers this ninth workspace crate and
requires its coverage. Windows/macOS native support and tests remain deferred;
their existing explicit pipeline failures remain unchanged.

Directory/bundle discovery, duplicate registry IDs, activation and deactivation,
host callback registry, queue attachment, signed lengths, response/free ownership,
notifications, optional automation calls, safe reload policy and real typed-client
Go/Rust lifecycle tests remain required. Metadata admission alone does not prove
C callback transport, live add-in compatibility, or foreground application use.

The [native lifecycle adapter](implementation-native-addin-lifecycle.md) now
implements explicit lifecycle calls and copied/free-managed automation replies.
The managed host and queued callback transport remain open.
