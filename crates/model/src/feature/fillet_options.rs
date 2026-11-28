// SPDX-License-Identifier: GPL-2.0-only
//! Extended v2 Fillet options (variable radius, continuity, corner setbacks, edge sets, advanced flags).
use super::work_geometry::DatumLength;
use crate::parameter::ParameterCollection;

/// Primary mode of a Part Modify Fillet feature.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum FilletMode {
    #[default]
    Constant,
    Variable,
    CornerSetback,
}

/// Cross-section continuity across filleted faces.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum FilletContinuity {
    #[default]
    Tangent,
    SmoothG2,
}

/// One constant-radius edge group inside a multi-set Fillet feature.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
pub struct FilletEdgeSet {
    pub edges: Vec<Vec<u8>>,
    pub radius: DatumLength,
}

/// One `(position, radius)` station along a variable-radius fillet edge (`position` in `[0.0, 1.0]`).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
pub struct FilletVariablePoint {
    pub position: f64,
    pub radius: DatumLength,
}

/// Corner setback configuration anchored at a shared vertex of 3+ filleted edges.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
pub struct FilletCornerSetback {
    pub vertex: Vec<u8>,
    pub minimal: bool,
    pub distances: Vec<DatumLength>,
}

/// Advanced Properties flags persisted on a Fillet feature.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub struct FilletAdvancedOptions {
    pub roll_along_sharp_edges: bool,
    pub rolling_ball_where_possible: bool,
    pub preserve_all_features: bool,
    pub automatic_edge_chain: bool,
}

impl Default for FilletAdvancedOptions {
    fn default() -> Self {
        Self {
            roll_along_sharp_edges: false,
            rolling_ball_where_possible: true,
            preserve_all_features: true,
            automatic_edge_chain: false,
        }
    }
}

/// Optional v2 Fillet settings; omitted from JSON when equal to v1 defaults.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
pub struct FilletOptions {
    #[serde(default)]
    pub mode: FilletMode,
    #[serde(default)]
    pub continuity: FilletContinuity,
    #[serde(default = "default_true")]
    pub smooth_radius_transition: bool,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub edge_sets: Vec<FilletEdgeSet>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub variable_points: Vec<FilletVariablePoint>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub corner_setback: Option<FilletCornerSetback>,
    #[serde(default)]
    pub advanced: FilletAdvancedOptions,
}

fn default_true() -> bool {
    true
}

impl Default for FilletOptions {
    fn default() -> Self {
        Self {
            mode: FilletMode::Constant,
            continuity: FilletContinuity::Tangent,
            smooth_radius_transition: true,
            edge_sets: Vec::new(),
            variable_points: Vec::new(),
            corner_setback: None,
            advanced: FilletAdvancedOptions::default(),
        }
    }
}

/// Resolved numeric v2 Fillet payload ready for Martensite evaluation.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedFillet {
    pub radius: f64,
    pub mode: FilletMode,
    pub continuity: FilletContinuity,
    pub smooth_radius_transition: bool,
    pub edge_sets: Vec<(Vec<Vec<u8>>, f64)>,
    pub variable_points: Vec<(f64, f64)>,
    pub corner_setback: Option<(Vec<u8>, bool, Vec<f64>)>,
    pub advanced: FilletAdvancedOptions,
}

impl FilletOptions {
    /// True when this options payload matches v1 constant-radius fillet defaults.
    pub fn is_default(&self) -> bool {
        *self == Self::default()
    }

    /// Collect all length sources referenced by v2 options (edge sets, variable stations, setbacks).
    pub fn lengths(&self) -> Vec<DatumLength> {
        let mut out = Vec::new();
        for set in &self.edge_sets {
            out.push(set.radius);
        }
        for pt in &self.variable_points {
            out.push(pt.radius);
        }
        if let Some(sb) = &self.corner_setback {
            out.extend(sb.distances.iter().copied());
        }
        out
    }

    /// Validate source lengths and ranges without resolving parameters.
    pub fn validate(&self) -> Result<(), String> {
        for set in &self.edge_sets {
            if set.edges.is_empty() {
                return Err("fillet edge set: expected at least one edge".into());
            }
            super::fillet::validate_radius_source(set.radius, "fillet set")?;
        }
        for pt in &self.variable_points {
            if !pt.position.is_finite() || !(0.0..=1.0).contains(&pt.position) {
                return Err(format!(
                    "fillet variable position {}: expected value in [0.0, 1.0]",
                    pt.position
                ));
            }
            super::fillet::validate_radius_source(pt.radius, "fillet variable")?;
        }
        if let Some(sb) = &self.corner_setback {
            if sb.vertex.is_empty() {
                return Err("fillet corner setback: expected a vertex reference".into());
            }
            for &dist in &sb.distances {
                super::fillet::validate_radius_source(dist, "fillet setback")?;
            }
        }
        Ok(())
    }

    /// Resolve all numeric lengths against current parameters.
    pub fn resolve(
        &self,
        radius: f64,
        parameters: &ParameterCollection<u64>,
    ) -> Result<ResolvedFillet, String> {
        let mut edge_sets = Vec::with_capacity(self.edge_sets.len());
        for set in &self.edge_sets {
            edge_sets.push((
                set.edges.clone(),
                resolve_positive(set.radius, parameters, "fillet set")?,
            ));
        }
        let mut variable_points = Vec::with_capacity(self.variable_points.len());
        for pt in &self.variable_points {
            variable_points.push((
                pt.position,
                resolve_positive(pt.radius, parameters, "fillet variable")?,
            ));
        }
        let corner_setback = match &self.corner_setback {
            None => None,
            Some(sb) => {
                let mut dists = Vec::with_capacity(sb.distances.len());
                for &d in &sb.distances {
                    dists.push(resolve_positive(d, parameters, "fillet setback")?);
                }
                Some((sb.vertex.clone(), sb.minimal, dists))
            }
        };
        Ok(ResolvedFillet {
            radius,
            mode: self.mode,
            continuity: self.continuity,
            smooth_radius_transition: self.smooth_radius_transition,
            edge_sets,
            variable_points,
            corner_setback,
            advanced: self.advanced,
        })
    }
}

fn resolve_positive(
    length: DatumLength,
    parameters: &ParameterCollection<u64>,
    label: &str,
) -> Result<f64, String> {
    let val = length.resolve(parameters).map_err(|e| e.to_string())?;
    if !val.is_finite() || val <= 0.0 {
        return Err(format!("{label} length {val}: expected positive length"));
    }
    Ok(val)
}
