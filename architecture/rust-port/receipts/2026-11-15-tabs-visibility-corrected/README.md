<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Document tab visibility diagnostic

On 2026-11-15, two native scenarios passed on Linux x86-64 under Xvfb and
Mesa software Vulkan. The harness injects pointer and keyboard input through
the GUI frame loop. Hide and restore passed 10 steps; the extended window list
passed seven. The second fixture opens 16 documents before the Switch menu and
dialog, then checks that an existing row moves when the list is scrolled.

The application base was `3fe4635e3a8067727fa1ec93c6d26f6cfedeb475` with
`document-tabs/steps.rs` SHA-256
`d6a8485a2383e7364dbd3ffda468351dd9614d1be2dc4409b77bc060e25dd64a`.
The pinned source was `05317bbb2d0809a532b29b7ff6f726a5722b865f`; the feature
hash was `2738bff62d9968e32a59d426e9a3e762938981a2bdb7e074f77bdf59c5d4928f`.
Both source reports record an unchanged clean snapshot.

Each run used a dedicated Cargo target and its own report directory, with
`OBK_BDD_FEATURE=document-tabs` and `OBK_BDD_SCENARIO` set to the complete
scenario name. The command was `cargo test --locked -p oblikovati-head --lib
application_scenarios_use_the_native_window -- --nocapture`, run through
`xvfb-run -a -s '-screen 0 1920x1080x24 -noreset'` with the Mesa software ICD.
The JSON, JUnit, source reports and captures are retained in `hide/` and `list/`.
The list capture was visually inspected: it shows the scrolled document list,
its scrollbar, OK and Cancel, with the document tab strip hidden behind it.
Build provenance remains open (#3978), so these are diagnostic results.
