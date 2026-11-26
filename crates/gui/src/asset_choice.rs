// SPDX-License-Identifier: GPL-2.0-only
//! Asset identity and the name shown by a selector.
/// One named asset in a selector and its authored base-colour swatch.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiAssetChoice {
    pub id: String,
    pub name: String,
    pub swatch: Option<[u8; 3]>,
    pub appearance_preview: Option<GuiAppearancePreview>,
    pub category: Option<&'static str>,
    pub document: bool,
    pub used: bool,
    pub library: Option<GuiAssetLibrary>,
}

/// Asset source shown in the browser's library region.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub enum GuiAssetLibrary {
    Shipped,
    Project,
}

/// Quantized authored finish properties used for small browser previews.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct GuiAppearancePreview {
    pub roughness: u8,
    pub metalness: u8,
    pub anisotropy: u8,
    pub fuzz_weight: u8,
    pub fuzz_color: [u8; 3],
}
