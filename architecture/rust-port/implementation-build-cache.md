# Rust compiler cache

Issue #3804 adds [Kache](https://github.com/kunobi-ninja/kache) 2.0.0 to local
Make builds and the shared CI setup. Both the client and engine are pinned.
The installer checks the release archive SHA-256 before extraction. It installs
only those two binaries in `.tools`; it does not change global Cargo settings.

## Local commands

Run `make cache-tools` once, then use `make build`, `make test`, or `make gate`.
`make tools` also installs the cache tools. Run `make cache-stats` to read JSON
statistics. Use `make BUILD_CACHE=off build` to build without a compiler wrapper.
The full gate still requires the pinned Kache version in the tool inventory.

Each worktree keeps its own Cargo target directory. The cache store defaults to
`oblikovati-kache` under the absolute Git common directory. This permits reuse
between worktrees without shared Cargo target locks. A source archive without
Git uses `.cache/kache`. Set `KACHE_CACHE_DIR` to select another store.
`KACHE_CONFIG` selects the project configuration explicitly, including compiler
calls that run from dependency source directories. Direct local Cargo commands
do not enable the wrapper automatically.

The local store has a 32 GiB limit. Automatic target deletion and target sharing
are disabled. Worktree and target cleanup remain the caller's responsibility.
No remote cache credentials or global cache configuration are installed.
These settings follow the upstream
[configuration](https://ninja.kunobi.com/docs/kache/getting-started/configuration)
and [worktree guidance](https://ninja.kunobi.com/docs/kache/getting-started/worktrees).

## CI

The shared setup uses the official Kache action at commit
`ad7317540dcdd71c904f7a60a87b620a0c0e67ed`, with Kache v2.0.0 and a 4 GiB cache.
The action supplies the wrapper and GitHub cache storage for direct Cargo calls.
PR comments are disabled. Make preserves the action's cache environment.
C and C++ compiler wrapping is not enabled by this change.

## Measured application build

On Linux x86-64, build source `e41f0af7348beb5a4a1cc829ddbde0f794a411f4`
with `cargo build --locked -p oblikovati-head --all-features` and four build jobs.
Use two separate worktrees, empty private Cargo targets, and one scoped store.

| Run | Seconds | Local hits | Misses |
| --- | ---: | ---: | ---: |
| Cold cache | 154.324 | 0 | 190 |
| Second worktree | 27.710 | 149 | 1 |

The second build took 82% less time, or about 5.6 times less time. This compares
cold and warm Kache builds; it is not a cache-disabled baseline. Both executable
SHA-256 values were
`3293a1dfdb3dc16fb01aaf2d1696fc02f837cd53aa0a12a9e62e0613d131d140`.
Both executables passed a two-frame native window smoke test under Xvfb.
The reported hits are actual compiler cache hits, not Cargo `Fresh` results.
These results establish local cross-worktree reuse, not hosted CI speed.

Tests cover six desktop archive pins, limited extraction, bad checksum refusal,
download cleanup, Make paths with spaces, and the cache-disabled build path.
