// SPDX-License-Identifier: GPL-2.0-only
//! Authored settings for the optional Generic document appearance effects.

/// Generic appearance settings retained when the individual effects are disabled.
#[derive(Clone, Debug, Default, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(default, rename_all = "camelCase")]
pub struct GenericAppearanceEffects {
    pub cutouts_enabled: bool,
    pub bump: AppearanceBump,
    pub illumination: AppearanceIllumination,
    pub tint: AppearanceTint,
}

impl GenericAppearanceEffects {
    pub(super) fn validate(&self) -> Result<(), String> {
        if self.bump.amount > 100 {
            return Err(format!(
                "bump amount {}: expected a percentage from 0 to 100",
                self.bump.amount
            ));
        }
        if !(1000..=12000).contains(&self.illumination.temperature) {
            return Err(format!(
                "illumination temperature {}: expected 1000 to 12000 K",
                self.illumination.temperature
            ));
        }
        Ok(())
    }
}

/// Image-based bump strength.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(default)]
pub struct AppearanceBump {
    pub enabled: bool,
    pub amount: u8,
}

impl Default for AppearanceBump {
    fn default() -> Self {
        Self {
            enabled: false,
            amount: 25,
        }
    }
}

/// Generic self-illumination color, luminance in cd/m², and temperature in K.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(default)]
pub struct AppearanceIllumination {
    pub enabled: bool,
    pub color: [u8; 3],
    pub luminance: u16,
    pub temperature: u16,
}

impl Default for AppearanceIllumination {
    fn default() -> Self {
        Self {
            enabled: false,
            color: [255; 3],
            luminance: 100,
            temperature: 6500,
        }
    }
}

/// A multiplicative filter on the appearance's base color.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(default)]
pub struct AppearanceTint {
    pub enabled: bool,
    pub color: [u8; 3],
}

impl Default for AppearanceTint {
    fn default() -> Self {
        Self {
            enabled: false,
            color: [255; 3],
        }
    }
}
