// SPDX-License-Identifier: GPL-2.0-only
//! Authored Metal type, finish, and optional surface inputs.
use super::{AppearanceBump, AppearanceHeightMap, AppearanceTint};

/// The Metal editor recipe carried with a document appearance.
#[derive(Clone, Debug, Default, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(default, rename_all = "camelCase")]
pub struct MetalAppearance {
    pub kind: MetalKind,
    pub finish: MetalFinish,
    pub relief: AppearanceBump,
    pub cutouts: bool,
    pub tint: AppearanceTint,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub relief_image: Option<AppearanceHeightMap>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cutout_image: Option<AppearanceHeightMap>,
}

impl MetalAppearance {
    pub(super) fn validate(&self) -> Result<(), String> {
        if self.relief.amount > 100 {
            return Err(format!(
                "metal relief amount {}: expected a percentage from 0 to 100",
                self.relief.amount
            ));
        }
        for map in self.relief_image.iter().chain(&self.cutout_image) {
            map.validate()?;
        }
        Ok(())
    }
}

/// Alloy and surface treatment presets in the Metal editor.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub enum MetalKind {
    #[default]
    Aluminum,
    AnodizedAluminum,
    Chrome,
    Copper,
    Brass,
    Bronze,
    StainlessSteel,
    Zinc,
}

impl MetalKind {
    /// Return the preset base color, for example `MetalKind::Brass.color()`.
    pub fn color(self) -> [u8; 3] {
        match self {
            Self::Aluminum => [195, 202, 211],
            Self::AnodizedAluminum => [158, 171, 187],
            Self::Chrome => [222, 226, 231],
            Self::Copper => [215, 121, 77],
            Self::Brass => [215, 172, 78],
            Self::Bronze => [171, 116, 69],
            Self::StainlessSteel => [169, 179, 190],
            Self::Zinc => [163, 171, 179],
        }
    }
}

/// The finish of a metal surface.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub enum MetalFinish {
    #[default]
    Polished,
    SemiPolished,
    Satin,
    Brushed,
}

impl MetalFinish {
    /// Compute roughness and highlight weight, for example `MetalFinish::Satin.surface()`.
    pub fn surface(self) -> [u8; 2] {
        match self {
            Self::Polished => [27, 241],
            Self::SemiPolished => [74, 204],
            Self::Satin => [132, 170],
            Self::Brushed => [109, 195],
        }
    }

    /// Compute roughness anisotropy, for example `MetalFinish::Brushed.anisotropy()`.
    pub fn anisotropy(self) -> u8 {
        if self == Self::Brushed { 192 } else { 0 }
    }
}
