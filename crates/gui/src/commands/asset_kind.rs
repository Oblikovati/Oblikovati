// SPDX-License-Identifier: GPL-2.0-only
/// The part-level asset a ribbon selection changes.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiPartAssetKind {
    Material,
    Appearance,
}
