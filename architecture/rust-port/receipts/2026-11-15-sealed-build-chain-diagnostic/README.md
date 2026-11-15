<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Sealed build and native runner diagnostic

On 2026-11-15, `tools/bdd/sealed_build_cli.py` built the pinned Oblikovati
application and ran `Inspect a part file before opening it` in a Linux Bubblewrap
runtime with Xvfb and Vulkan Lavapipe. One scenario and all 12 steps passed. The
final screenshot shows the active part after the Open dialog was cancelled. The
scenario used native frame-loop input, not operating-system mouse or keyboard
input.

The application source was exported from commit
`57a2d64d8a50966a81317ac3e63ce43619f6cf4a`. The scenario checkout was the clean
pinned snapshot `05317bbb2d0809a532b29b7ff6f726a5722b865f` before and after
the run. The offline sandbox build and sealed execution observed the same runner
SHA-256:

```text
3f559d85705b45aeb4b9669b5797922d4e124afa1ec784e1864cfe2a6011bafa
```

The sandbox executable hash also matched between build and execution. The build
checked 280 registry packages against their locked archives and 22,972 Git
vendor files against locked trees. The generated manifests for 18 Git packages
were not verified. The execution observed 556 runtime entries and 1,531 scenario
checkout entries. These observations and the build and native logs are retained
in the compressed files beside this document.

The CLI received an exported source tree, an offline vendor tree made with
`cargo vendor --locked --offline --versioned-dirs`, the installed Rust 1.99.0
toolchain, a copied Bubblewrap executable, an empty build target and log folder,
a copied runtime based on the earlier sealed-runner inventory, and the pinned
scenario checkout. Registry archives and both locked Git repositories were
provided to the optional vendor checks. The documented command form is in
[scenario dependencies](../../scenario-dependencies.md).

This is diagnostic evidence only. The build and execution both report
`reusable: false`, and runtime coverage is `unbounded`. The host toolchain and
system files, generated Git manifests, kernel, devices, runtime inputs, and
concurrent writes are not independently bounded. The runner identity link does
not establish a trusted build or a reusable receipt. Issue #3978 remains open.
