// SPDX-License-Identifier: GPL-2.0-only
/// Application background colours are authored in sRGB bytes, independent of scene geometry.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ViewportBackground {
    Solid([u8; 3]),
    VerticalGradient { top: [u8; 3], bottom: [u8; 3] },
}
