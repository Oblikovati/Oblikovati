<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Sketch history diagnostic (#3880)

Application commit: `fbff41d8203cab398a23c345e8ab47977125e13e`.
Scenario snapshot: `e76cb749ed43c73a91c24854057da7c890041f51`.

The pinned `new-sketch-04` scenario passed all seven steps in a native window
on Xvfb with the Lavapipe Vulkan driver. The source report records an unchanged,
clean checkout. After the run, snapshot verification checked all 1,292 files.
The final capture shows Sketch1 and Sketch2 in the model browser and the part
ribbon after redo. The test sends mouse and keyboard input to the window frame
loop. It does not establish operating-system input delivery.

The seven `owner::tests::sketch_history` tests also passed. They cover typed
undo/redo queries, clean saved state, observer vetoes, document switching, and
multi-step history navigation. These tests and the native run used a clean
application checkout and an isolated Cargo target, but the toolchain, cached
dependencies, and host inputs were not independently bounded (#3978). This is
diagnostic evidence, not a reusable scenario receipt or a full gate result.

The strict gate rejected the first attempt because it used a scenario filter.
The reported run used the same verified snapshot without the strict-gate flag.
