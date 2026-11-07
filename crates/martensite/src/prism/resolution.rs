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

/// Choose explicit bounds or scale a reference set to the current profile extent.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum PrismResolutionPolicy {
    Fixed(PrismResolution),
    ModelRelative {
        reference: PrismResolution,
        reference_size: f64,
        minimum_size: f64,
    },
}
impl PrismResolutionPolicy {
    /// Check a policy before configuration; example: `policy.validate()?`.
    pub fn validate(self) -> Result<Self, String> {
        match self {
            Self::Fixed(reference) => {
                reference.validate()?;
            }
            Self::ModelRelative {
                reference,
                reference_size,
                minimum_size,
            } => {
                reference.validate()?;
                if !reference_size.is_finite()
                    || reference_size <= 0.
                    || !minimum_size.is_finite()
                    || minimum_size <= 0.
                {
                    return Err(format!(
                        "relative prism sizes reference={reference_size}, minimum={minimum_size}: expected finite positive lengths"
                    ));
                }
            }
        }
        Ok(self)
    }
    pub(crate) fn for_sketch(
        self,
        sketch: &oblikovati_model::history::ResolvedDocumentSketch<'_>,
    ) -> Result<PrismResolution, crate::EvaluationFailure> {
        let sketch_id = sketch.sketch.id();
        let policy = self.validate().map_err(|cause| {
            crate::EvaluationPhase::Input
                .refused(
                    crate::EvaluationFailureKind::InvalidInput,
                    crate::EvaluationEntity::Body,
                    cause,
                )
                .with_sketch_identity(sketch_id)
        })?;
        match policy {
            Self::Fixed(resolution) => Ok(resolution),
            Self::ModelRelative { .. } => {
                let extent = crate::sketch_prism::profile_extent::resolved_profile_extent(sketch)?;
                policy.for_size(extent.diagonal).map_err(|cause| {
                    crate::EvaluationPhase::Input
                        .refused(
                            crate::EvaluationFailureKind::InvalidInput,
                            crate::EvaluationEntity::Body,
                            cause,
                        )
                        .with_sketch_identity(sketch_id)
                })
            }
        }
    }
    /// Resolve a policy for a measured positive profile diagonal.
    /// Example: `policy.for_size(2.0)?` for a two-unit profile.
    pub fn for_size(self, size: f64) -> Result<PrismResolution, String> {
        if !size.is_finite() || size <= 0. {
            return Err(format!(
                "profile diagonal {size}: expected a finite positive length"
            ));
        }
        match self.validate()? {
            Self::Fixed(reference) => Ok(reference),
            Self::ModelRelative {
                reference,
                reference_size,
                minimum_size,
            } => {
                let factor = size.max(minimum_size) / reference_size;
                PrismResolution {
                    weld: reference.weld * factor,
                    sew: reference.sew * factor,
                    maximum_tolerance: reference.maximum_tolerance * factor,
                    ..reference
                }
                .validate()
            }
        }
    }
}

#[cfg(test)]
#[path = "../../tests/unit/prism_resolution.rs"]
mod tests;
