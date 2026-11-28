// SPDX-License-Identifier: GPL-2.0-only
//! Extended v2 Chamfer options: modes, reference face, reverse, edge chain, setback, and partial edges (#4046).
use crate::{
    feature::{FeatureReadFootprint, FeatureReadKey, work_geometry::DatumLength},
    parameter::ParameterCollection,
};

/// Chamfer sizing mode selectable in the Chamfer dialog.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum ChamferMode {
    #[default]
    EqualDistance,
    DistanceAndAngle,
    TwoDistances,
}

/// Which field in the Partial chamfer table is computed from the other two and edge length.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum ChamferDrivenDimension {
    ToStart,
    #[default]
    Chamfer,
    ToEnd,
}

/// Partial-edge chamfer extent in centimeters along the selected straight edge.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ChamferPartialSpec {
    pub edge_length_cm: f64,
    pub to_start_cm: f64,
    pub chamfer_length_cm: f64,
    pub to_end_cm: f64,
    #[serde(default)]
    pub driven: ChamferDrivenDimension,
}

impl ChamferPartialSpec {
    /// Validate partial chamfer lengths against the edge span.
    pub fn validate(&self) -> Result<(), String> {
        if !self.edge_length_cm.is_finite() || self.edge_length_cm <= 0.0 {
            return Err("chamfer partial edge length: expected a finite positive value".into());
        }
        if !self.to_start_cm.is_finite() || self.to_start_cm < 0.0 {
            return Err("chamfer partial To Start: expected a non-negative length".into());
        }
        if !self.to_end_cm.is_finite() || self.to_end_cm < 0.0 {
            return Err("chamfer partial To End: expected a non-negative length".into());
        }
        if !self.chamfer_length_cm.is_finite() || self.chamfer_length_cm <= 0.0 {
            return Err("chamfer partial length: expected a positive length".into());
        }
        let sum = self.to_start_cm + self.chamfer_length_cm + self.to_end_cm;
        if sum > self.edge_length_cm + 1e-4 {
            return Err(format!(
                "chamfer partial lengths sum ({sum:.4} cm) exceeds edge length ({:.4} cm)",
                self.edge_length_cm
            ));
        }
        Ok(())
    }
}

/// Authored v2 Chamfer options; `Default` produces a byte-identical v1 equal-distance recipe.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ChamferOptions {
    #[serde(default)]
    pub mode: ChamferMode,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub distance2: Option<DatumLength>,
    #[serde(default = "default_angle_deg")]
    pub angle_deg: f64,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub reference_face: Option<Vec<u8>>,
    #[serde(default)]
    pub reversed: bool,
    #[serde(default)]
    pub edge_chain_second: bool,
    #[serde(default)]
    pub setback_second: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub partial: Option<ChamferPartialSpec>,
}

const fn default_angle_deg() -> f64 {
    45.0
}

impl Default for ChamferOptions {
    fn default() -> Self {
        Self {
            mode: ChamferMode::EqualDistance,
            distance2: None,
            angle_deg: default_angle_deg(),
            reference_face: None,
            reversed: false,
            edge_chain_second: false,
            setback_second: false,
            partial: None,
        }
    }
}

impl ChamferOptions {
    /// True when no v2 field diverges from the v1 equal-distance chamfer contract.
    pub fn is_default(&self) -> bool {
        self.mode == ChamferMode::EqualDistance
            && self.distance2.is_none()
            && (self.angle_deg - default_angle_deg()).abs() <= 1e-9
            && self.reference_face.is_none()
            && !self.reversed
            && !self.edge_chain_second
            && !self.setback_second
            && self.partial.is_none()
    }

    /// Validate authored v2 fields before saving or replaying.
    pub fn validate(&self) -> Result<(), String> {
        match self.mode {
            ChamferMode::EqualDistance => {}
            ChamferMode::DistanceAndAngle => {
                if !self.angle_deg.is_finite() || self.angle_deg <= 0.0 || self.angle_deg >= 90.0 {
                    return Err(format!(
                        "chamfer angle {}: expected finite angle in (0, 90) degrees",
                        self.angle_deg
                    ));
                }
            }
            ChamferMode::TwoDistances => {
                let d2 = self
                    .distance2
                    .ok_or("chamfer two-distance mode: expected Distance2")?;
                d2.validate_source().map_err(|e| e.to_string())?;
            }
        }
        if let Some(partial) = &self.partial {
            partial.validate()?;
        }
        Ok(())
    }

    /// Append parameter dependencies from v2 fields into `reads`.
    pub fn extend_reads(&self, reads: &mut FeatureReadFootprint) {
        if let Some(id) = self.distance2.and_then(DatumLength::parameter) {
            reads.keys.insert(FeatureReadKey::Parameter(id));
        }
    }
}

/// Resolved numeric Chamfer inputs handed to the Martensite backend.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedChamfer {
    pub mode: ChamferMode,
    pub distance: f64,
    pub distance2: Option<f64>,
    pub angle_rad: f64,
    pub reference_face: Option<Vec<u8>>,
    pub reversed: bool,
    pub partial: Option<ChamferPartialSpec>,
}

impl ChamferOptions {
    /// Resolve all numeric distances and angle for kernel execution.
    pub fn resolve(
        &self,
        distance: f64,
        parameters: &ParameterCollection<u64>,
    ) -> Result<ResolvedChamfer, String> {
        let distance2 = match self.mode {
            ChamferMode::TwoDistances => {
                let d2 = self
                    .distance2
                    .ok_or("chamfer two-distance mode: expected Distance2")?;
                let val = d2.resolve(parameters).map_err(|e| e.to_string())?;
                if !val.is_finite() || val <= 0.0 {
                    return Err(format!(
                        "chamfer Distance2 {val}: expected a finite positive length"
                    ));
                }
                Some(val)
            }
            _ => None,
        };
        if self.mode == ChamferMode::DistanceAndAngle
            && (!self.angle_deg.is_finite() || self.angle_deg <= 0.0 || self.angle_deg >= 90.0)
        {
            return Err(format!(
                "chamfer angle {}: expected finite angle in (0, 90) degrees",
                self.angle_deg
            ));
        }
        if let Some(partial) = &self.partial {
            partial.validate()?;
        }
        Ok(ResolvedChamfer {
            mode: self.mode,
            distance,
            distance2,
            angle_rad: self.angle_deg.to_radians(),
            reference_face: self.reference_face.clone(),
            reversed: self.reversed,
            partial: self.partial,
        })
    }
}
