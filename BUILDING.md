<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Build and run Oblikovati

On Linux or macOS, use Bash:

```sh
./build.sh
./build.sh run
./build.sh run -- "part with spaces.obk"
./build.sh --check
```

On Windows, use a native x64 or ARM64 PowerShell terminal:

```powershell
.\build.ps1
.\build.ps1 run
.\build.ps1 run "part with spaces.obk"
.\build.ps1 check
```

PowerShell 5.1 and PowerShell 7 syntax are supported. If Windows blocks a downloaded
script, review the files and use `Unblock-File` according to your local policy.
The scripts do not change the machine's execution policy.

The default command installs missing build dependencies, installs the Rust
version and components from `rust-toolchain.toml`, prepares the pinned kache
through `cargo xtask cache-tools`, and calls `cargo xtask build`. The Rust build
system remains responsible for workspace discovery, features, targets and build
validation. These scripts do not replace `make gate` or certify a platform.

The GUI executable is `oblikovati` (`oblikovati.exe` on Windows). The CLI remains
`oblikovati-cli`. Build outputs use `CARGO_TARGET_DIR` when set, otherwise
`target/debug`. Relative target directories are relative to this checkout.
Cross-compilation through `CARGO_BUILD_TARGET` is refused: these scripts prepare
the native desktop host.

## Run an existing build

`run` launches a verified current GUI without installing tools or rebuilding.
A build records the source state and GUI SHA-256 in an ignored `.tools` stamp.
Changed tracked or untracked source files, a changed binary, a missing binary,
or an absent stamp require a build first. The first use with an existing binary
built through another command therefore performs an incremental Rust build.
Changes made during a build prevent that binary from being marked current.
GUI arguments and the GUI's exit status are preserved.

Linux `run` needs `DISPLAY`, for the current X11/XWayland backend.
`WAYLAND_DISPLAY` alone is not sufficient. An interactive GUI is never launched
under an automatically created virtual display. For a headless **build**, the
script starts its own Xvfb server and closes that server when the script exits.
It does not close another user's display or GUI.

For the full local gate on a headless Linux machine, run `make gate-xvfb`.
It runs `make gate` under a private 1920 by 1080 Xvfb screen and closes the
server when the gate exits. Native tests capture the complete 1440 by 900
application window; the smaller default Xvfb screen cannot contain it.
Install Xvfb and xauth first. On a suitable real display, use `make gate`.

## Dependency installation

Linux selects an available package manager in this order:

- apt-get or apt: Debian, Ubuntu and related systems.
- dnf, microdnf or yum: Fedora and RPM systems.
- zypper: openSUSE.
- pacman: Arch Linux.
- apk: Alpine Linux.
- xbps-install: Void Linux.
- eopkg: Solus.
- emerge: Gentoo.
- rpm-ostree: Fedora Atomic systems.
- nix or nix-env: Nix profiles.

The required native packages include C/C++ compilation tools, Git, curl, tar,
CMake, Ninja, pkg-config, GLFW, Vulkan and X11 development libraries, and Xvfb
for headless builds. Package names are specific to the selected manager.
The script checks the resulting tools and pkg-config entries before building.
Distribution repositories must provide those packages. Installation errors are
reported; the script does not silently omit a dependency or enable third-party
repositories. Gentoo must provide Xvfb through its Xorg configuration when
building headlessly. rpm-ostree requires a reboot after installing new packages;
the script stops with that instruction and does not reboot the machine.

On macOS, the script uses Homebrew or MacPorts. If neither is available, it runs
a checksum-pinned official Homebrew installer. Missing Apple Command Line Tools
start Apple's installer; complete that OS dialog and rerun the script.

On Windows, Git is installed through WinGet, Chocolatey or Scoop, when available.
Without a manager, a checksum-pinned official Git installer is used. MSVC Build
Tools use WinGet or Chocolatey, or the official Microsoft installer with a valid
Microsoft Authenticode signature. The C++ workload, Windows SDK and native
x64/ARM64 developer environment are checked with a compiler/linker probe.
The native curl and bsdtar tools shipped with supported Windows 10/11 must be
available. A missing built-in OS tool requires OS repair rather than an
unverified replacement. Newly installed Git and Rust tools are added to the
current process's search path; no terminal restart is needed for that step.

Package installation can require `sudo`, an administrator terminal or the OS
installer's approval. The scripts report a missing right and stop. They do not
bypass authentication. Downloads use HTTPS; standalone Rustup, Git and Homebrew
inputs are pinned in `tools/bootstrap/downloads.lock`. Rust crate and kache
versions remain defined by the existing Cargo files and Rust build system.
Set `BUILD_CACHE=off` explicitly to build without kache.

## Bootstrap tests

The process tests use temporary checkouts and named fake installers/build tools;
they do not install host packages or run an application build.

```sh
python3 -m unittest discover -s tools/bootstrap/tests -p 'test_*.py' -v
```

```powershell
pwsh -NoProfile -File tools/bootstrap/tests/test_windows.ps1
```

The PowerShell tests can also run with a portable PowerShell on Linux. Such a run
checks parsing, command construction, freshness, failure and installer paths; it
is not native Windows, MSVC or macOS acceptance evidence.

Official references: [Rustup installation](https://rust-lang.github.io/rustup/installation/other.html),
[Homebrew installation](https://docs.brew.sh/Installation),
[Microsoft C++ Build Tools](https://learn.microsoft.com/en-us/cpp/overview/acquire-msvc),
and [Visual Studio installer parameters](https://learn.microsoft.com/en-us/visualstudio/install/use-command-line-parameters-to-install-visual-studio).

## Commit message check

Install the repository hook in each clone with Python 3.10 or later on `PATH`:

```sh
git config --local core.hooksPath .githooks
```

This sets the hook path for all worktrees in that clone. Check an existing
`core.hooksPath` before changing it; Git supports one hook directory at a time.
The hook needs a subject, a blank line, an explanatory body, and a separate
`Refs #123` line. The same rule applies to merge commits. For example:

```text
Fix document selection

Keep the selected document active after the tab closes.

Refs #123
```

Run the hook tests with `python3 -B -m unittest discover -s tools/git`.
The hook is local to each clone and Git permits explicit hook bypasses. It does
not enforce server-side policy or validate that the issue exists.
