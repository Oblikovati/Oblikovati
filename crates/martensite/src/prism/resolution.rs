// SPDX-License-Identifier: GPL-2.0-only
//! Application-owned modelling and display bounds for a profile extrusion.
use martensite_tkbo::profile_extrusion::{DisplayTolerance, Resolution, ToleranceCap};

/// Explicit version-one limits; distances use the document's working length unit.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct PrismResolution {
    pub version: u32,
    pub weld: f64,
    pub sew: f64,
    pub angular: f64,
    pub display_chordal: f64,
    pub display_angular: f64,
    pub maximum_tolerance: f64,
}
impl PrismResolution {
    /// Validate explicit bounds; example: `resolution.validate()?` before evaluation.
    pub fn validate(self) -> Result<Self, String> {
        if self.version != 1 {
            return Err(format!(
                "prism resolution version {}: expected 1",
                self.version
            ));
        }
        self.kernel().admit().map_err(|(field, value)| {
            format!("prism resolution {field}={value}: expected a finite positive value")
        })?;
        Ok(self)
    }
    pub(crate) fn kernel(self) -> Resolution {
        Resolution::new(
            self.weld,
            self.sew,
            self.angular,
            DisplayTolerance {
                chordal: self.display_chordal,
                angular: self.display_angular,
            },
            ToleranceCap::Refuse(self.maximum_tolerance),
        )
    }
}

#[cfg(test)]
#[path = "../../tests/unit/prism_resolution.rs"]
mod tests;
