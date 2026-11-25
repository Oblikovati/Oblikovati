// SPDX-License-Identifier: GPL-2.0-only
//! Asset identity and the name shown by a selector.
/// One named asset in a selector and its authored base-colour swatch.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiAssetChoice {
    pub id: String,
    pub name: String,
    pub swatch: Option<[u8; 3]>,
}
