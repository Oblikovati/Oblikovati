# Building and releasing Oblikovati Rust

The workflows replace the Go build, test, package, and publish commands on the
`rust` branch. They implement [ADR-0012](architecture/decisions/ADR-0012-build-and-validation.md)
and [ADR-0013](architecture/decisions/ADR-0013-packaging-and-release.md).
[Issue #3560](https://github.com/Oblikovati/Oblikovati/issues/3560) tracks this port.

The application port is incomplete. The Cargo workspace contains the build tool,
model/storage/session foundations, a CLI, and a native head with saved prism
display and camera navigation. Dear ImGui chrome and editing controls remain
open work. A passing gate tests the code that exists.
Application package and publication preflight checks must fail until the real
application, released dependencies, resources, and native acceptance tests exist.

## Local commands

Install Rust with rustup, Node.js 24.15.0, npm, Git, Make, curl, and tar. The
repository pins Rust 1.99.0, including rustfmt, Clippy, and LLVM tools.
`make tools` installs the locked npm tools, cargo-llvm-cov 0.9.1, and actionlint
1.7.7. The actionlint archive has a checked-in SHA-256 pin for each host.

```sh
make tools
make gate
make build
make test
make lint
make docs-lint
cargo xtask version stable
cargo xtask version nightly
cargo xtask matrix
```

`make gate` discovers every `Cargo.toml` outside generated directories, asks
Cargo for each workspace, and checks every member with `--locked`. It runs
builds, formatting, Clippy with warnings denied, all-target/all-feature tests,
separate documentation tests, per-package production line coverage greater
than 80%, duplication below 3%, Markdown/link checks, SPDX/license checks,
workflow validation, and `git diff --check`. It rejects sibling source overrides
and kernel imports outside the adapter/viewport boundary.

Coverage excludes test source under a package's `tests/` directory. It includes
all production source, including this build tool and its command entry point.
Unit tests live in separate files to keep their lines out of the production
percentage. Duplication measures Rust production source with the same test
exclusion; generated output and installed tools are excluded. Reports go to
`target/quality/`. These measures are not application behavior coverage.

Set `package.metadata.oblikovati.requires-display = true` on crates that need a
display. `oblikovati-head` always requires one on Linux. Run `xvfb-run -a make
gate` when no physical display is available. A missing required display is an
error. Native CI checks every target below; a local Linux pass is not a macOS
or Windows runtime result.

This host's installed Vulkan validation layer 1.3.275 crashes during concurrent
device use. Local acceptance uses the official Ubuntu layer 1.4.321.0-1, kept
under ignored `.tools/vulkan-validation-1.4.321/`, with its manifest naming the
absolute library path. Validation stays enabled and tests stay concurrent:

```sh
VK_LAYER_PATH="$PWD/.tools/vulkan-validation-1.4.321/layers" \
  CARGO_NET_OFFLINE=true make gate
```

The [native stress report](architecture/rust-port/diagnostic-startup-native-crash.md)
records package provenance, the failing stack, and the comparison. This is a
local test-tool selection; it does not change system libraries or the released
Martensite pin. Windows and macOS tests remain deferred by user instruction.

## Dependency ownership

**Oblikovati.API and Martensite each test, package, and publish their own
releases.** Application CI does not run their private test suites, publish their
packages, or read sibling working trees. It tests application integration with
their released Rust crates.

Update `release.toml`, the Cargo dependency declarations, and `Cargo.lock` in a
reviewed dependency change. Each owner pin records its version and full release
commit. Cargo Git dependencies must resolve to that exact release commit. The
preflight check requires a published, stable `vVERSION` GitHub release with a
`NAME-VERSION.crate` asset for every consumed owner crate. Annotated tags are
resolved to their commit before comparison. All consumed Martensite toolkits
must resolve to the same owner release and commit.

The API repository owns Go/Rust contract synchronization, examples, and wiki
gates. The kernel repository owns geometry, oracle, and kernel-package gates.
The application still owns its API/host inventory, kernel integration, feature,
live GUI, and installation tests. The application now pins the stable Martensite
v0.3.6 release; an unpublished local kernel revision is not a release pin. See
[migration evidence](architecture/rust-port/martensite-v0.4.0-migration.md).
Product publication remains blocked by the R9 readiness fields.

## Product version

The product label remains `MAJOR.API.MINOR.PATCH`. Cargo package versions remain
three-component SemVer. `product.major` in `release.toml` replaces `version.yaml`.
The API field concatenates three decimal components, each padded to at least
two digits: `0.2.0` becomes `000200`; `0.154.0` becomes `0015400`.

Reachable `rust-v…` tags seed the version. `feat` or a breaking change increments
MINOR and resets PATCH. `fix`, `perf`, `revert`, and unknown subjects increment
PATCH. Documentation/build-only changes hold the version. A major/API change
starts a new line, with the bump measured since the preceding Rust release.
Nightlies append `-nightly.YYMMDDHH` in UTC. Go `v…` tags never seed Rust history.
Git errors fail version calculation; they cannot silently return a fresh version.

## Workflow and artifact matrix

| Workflow | Trigger and result |
| --- | --- |
| `ci.yml` | PRs to `rust`, pushes to `rust`, and reusable calls. Full Linux gate and six native Rust build/test jobs. |
| `build.yml` | Reusable call or manual development build of an exact commit on `rust`. Requires released dependency pins, complete inputs, CI, and native package smoke tests. |
| `nightly.yml` | Six-hour schedule or manual run. Inactive unless `RUST_PREVIEW_ENABLED=true`. Skips a commit that already has a preview. |
| `release.yml` | Manual stable release from `rust`; requires `RUST_STABLE_ENABLED=true`. |
| `publish.yml` | Internal reusable publisher. Verifies all nine packages and checksums before it makes a draft public. |

| Target | Native runner | CLI archive | GUI package |
| --- | --- | --- | --- |
| Linux x86-64 | Ubuntu 24.04 | tar.gz | AppImage |
| Linux arm64 | Ubuntu 24.04 arm64 | tar.gz | Not in the legacy shipping matrix |
| Windows x86-64 MSVC | Windows 2025 | zip | zip |
| Windows arm64 MSVC | Windows 11 arm64 | zip | Not in the legacy shipping matrix |
| macOS x86-64 | macOS 15 Intel | tar.gz | Universal app, combined with arm64 |
| macOS arm64 | macOS 14 | tar.gz | Universal app, combined with x86-64 |

CLI packages now include resources, native libraries, licenses, and metadata;
they are not assumed to be standalone, pure-Go executables. Linux packages use
the Ubuntu 24.04 build ABI and the host glibc/GPU driver. Windows packages use
MSVC, not the former MinGW Go toolchain. These choices need the R9 native
installation checks before distribution is enabled.

All names start with `oblikovati-rust-`. macOS uses `Oblikovati Rust.app` and
`org.oblikovati.rust`. The product label is a separate plist field; Apple's build
number is encoded from the source commit's UTC minute within its numeric limits.
The application must also isolate preview configuration and open copies of Go
projects, as required by ADR-0013. Package names alone do not prove that behavior.

## Application integration contract

The `oblikovati-cli` package now provides manifest inspection and version/help.
Finish its remaining commands and add the real `oblikovati-head` package with a
binary of the same name in the root workspace. The packaging build passes
`OBLIKOVATI_BUILD_MANIFEST` and `SOURCE_DATE_EPOCH`. The application build script
must embed that manifest, declare it as a rebuild input, and make `--version`
report the product version. Do not stamp only the Cargo tool version.

The manifest records the application commit/product version, Rust toolchain,
API version/commit, Martensite version/commit, document schema, reference-key
scheme, and C ABI identifier. Final release metadata also records every package
SHA-256. Downloads with extra, duplicate, empty, missing, or changed artifacts
are rejected. Intermediate macOS stages are not publication inputs.

Supply the required files under the resource directory in `release.toml`:
`icons/oblikovati.png`, `icons/Oblikovati.icns`, `fonts/`, `shaders/`, `templates/`,
`catalogs/`, and `NOTICES`. Each must contain real content. The Go icon generator
is no longer a build dependency. Native Windows dependencies must be available
beside the target's release binaries so the recursive PE import check can bundle
them. Linux resolves linked libraries and macOS relocates all non-system dylibs.

Add `packaging/smoke/TARGET.sh` for each target. It receives the installed package
directory, runs with that directory as its working directory, and must stress
the installed application and add-ins without a sibling checkout. A script for
a GUI target must drive MCPBridge and retain screenshots and model assertions.
For the universal macOS check, the directory is the assembled `.app`. Test
installation, library lookup, resources, file associations, updates, and rollback.
No placeholder smoke scripts are provided.

The Linux AppImage tool must have an HTTPS URL and SHA-256 pin in
`release.toml`. Distribution also needs pinned and licensed native SDK inputs;
the current apt/brew CI setup is a development environment, not an R9 native
dependency receipt. No package has native application acceptance yet.

## Enable publication after R9

1. Complete and review the application, API/host, kernel integration, native
   package, both-language add-in, installation, update, and rollback evidence.
2. Fill the released dependency pins, compatibility identifiers, resources,
   native tool pins, and smoke tests. Set `product.publication-ready=true` only
   with that evidence. `make release-check` must pass from clean source.
3. Configure the `rust-release` GitHub environment and the release secrets.
   Repository maintainers decide its reviewer and branch protection policy.
4. Enable `RUST_PREVIEW_ENABLED` for isolated previews. Enable
   `RUST_STABLE_ENABLED` only when stable Rust distribution is accepted.

Required macOS secrets are `MACOS_CERT_P12_BASE64`, `MACOS_CERT_PASSWORD`,
`MACOS_AC_APPLE_ID`, `MACOS_AC_PASSWORD`, and `MACOS_AC_TEAM_ID`. Publication
fails if any required credential or platform is absent. Development builds do
not require signing. Signing uses a temporary keychain, signs nested code,
submits notarization, and staples the app before the final archive is created.

Previews use immutable `rust-nightly-VERSION-COMMIT` tags. Stable Rust releases
use `rust-vVERSION`. Neither replaces `nightly`, a Go `v…` tag, or GitHub's Go
latest stable release. No old release is deleted by this pipeline. Stable
promotion rebuilds the selected `rust` commit and runs the complete checks; it
does not relabel an unchecked preview. A later Go retirement/update-channel
switch is a separate, tested R9 change.

To roll back a preview, reinstall its earlier immutable package and restore the
untouched project/configuration copies. Do not overwrite a Go-readable design
with a new schema. The repository contains no automatic production migration.

## Git hygiene

`.gitignore` covers root and nested Cargo output, downloaded tools, npm installs,
coverage/profiling files, distribution and macOS staging, local overrides,
signing keychains/certificates, and editor/OS files. Commit `Cargo.lock`,
`package-lock.json`, shared `.cargo/config.toml`, workflow files, and
`release.toml`. Never add a local dependency override to a release commit.
