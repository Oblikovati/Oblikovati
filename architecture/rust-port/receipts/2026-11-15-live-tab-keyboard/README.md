<!-- SPDX-License-Identifier: GPL-2.0-only -->

# OS-input document tab check

On 2026-11-15, a Linux Xvfb display ran the desktop with Vulkan Lavapipe. The
checkout was at `71ad97a08`; `README.md` had a local edit. The GUI was built
with `cargo build --locked -p oblikovati-head --bin oblikovati`. The executable
SHA-256 was `386afefadc0f69accd54e5de3d3869cc0bd338904576acf8ec15f5b36fd7aefa`.
The MCPBridge shared library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The build inputs were not bounded (#3978), so these are observations, not
commit-attributed acceptance evidence.

XTest sent mouse and key events to the OS window. The sequence was:

1. Click New Part, then use Ctrl+N and the template chooser to create Part2.
2. Press Ctrl+Shift+Tab: Part1 and its browser are active (`previous.png`).
3. Press Ctrl+Tab: Part2 and its browser are active (`next.png`).
4. Press Ctrl+Tab: Home is active (`home.png`).
5. Press Ctrl+Shift+Tab: Part2 and its browser are active (`wrapped.png`).

Both part tabs remain visible. MCPBridge captured each image; the images were
reviewed. `key.py.txt` and `input.py.txt` preserve the XTest input used. The
key script's window ID and display authentication were specific to this run.
This manually inspected check does not cover a populated viewport or the full
document-tabs feature. It does not replace a pinned native scenario receipt.
The application and Xvfb server were closed after the run.
