// SPDX-License-Identifier: GPL-2.0-only
//! Portable Generic appearance checker parameters.

/// A procedural checker attached to the Generic Image slot.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(default, rename_all = "camelCase")]
pub struct AppearanceChecker {
    pub colors: [[u8; 3]; 2],
    pub soften: u8,
    pub fade: u8,
    pub offset_mm: [i16; 2],
    pub rotation_degrees: i16,
    pub size_mm: [u16; 2],
    pub link_scale: bool,
    pub repeat: [TextureRepeat; 2],
}

impl Default for AppearanceChecker {
    fn default() -> Self {
        Self {
            colors: [[18; 3], [244; 3]],
            soften: 0,
            fade: 100,
            offset_mm: [0; 2],
            rotation_degrees: 0,
            size_mm: [25; 2],
            link_scale: true,
            repeat: [TextureRepeat::Tile; 2],
        }
    }
}

impl AppearanceChecker {
    pub(super) fn validate(self) -> Result<(), String> {
        if self.size_mm.contains(&0) || self.soften > 100 || self.fade > 100 {
            return Err(format!(
                "checker {self:?}: expected positive sample dimensions and percentages from 0 to 100"
            ));
        }
        Ok(())
    }
}

/// Repeat a texture independently along either sample axis.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub enum TextureRepeat {
    Tile,
    Mirror,
    Clamp,
}
