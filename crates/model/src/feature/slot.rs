// SPDX-License-Identifier: GPL-2.0-only
//! Authored Part Modify Slot feature recipe, validation, and parameter resolution.
use super::{
    FeatureId, FeatureReadFootprint, FeatureReadKey,
    fillet::reference_identity,
    hole::{HoleExtent, HoleFastenerSpec, HoleKind, HolePosition, HoleTerminationSurface},
    work_geometry::DatumLength,
};
use crate::parameter::ParameterCollection;
use std::collections::BTreeSet;

/// End geometry of a machined slot.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum SlotShape {
    #[default]
    Rounded,
    Rectangular,
}

/// Optional recessed seat at the start of a slot.
#[derive(Clone, Debug, Default, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum SlotSeat {
    #[default]
    None,
    Counterbore {
        width: DatumLength,
        depth: DatumLength,
    },
    Spotface {
        width: DatumLength,
        depth: DatumLength,
    },
    Countersink {
        width: DatumLength,
        angle_deg: f64,
    },
}

/// Resolved numeric slot seat geometry in working units (cm, degrees).
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub enum ResolvedSlotSeat {
    #[default]
    None,
    Counterbore {
        width: f64,
        depth: f64,
    },
    Spotface {
        width: f64,
        depth: f64,
    },
    Countersink {
        width: f64,
        angle_deg: f64,
    },
}

/// Bottom end-mill tip geometry for blind slots.
#[derive(Clone, Copy, Debug, Default, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum SlotEndMill {
    #[default]
    Flat,
    Point {
        angle_deg: f64,
    },
}

/// Center-mark display choice retained on a Slot feature.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum SlotCenterMarks {
    #[default]
    None,
    Center,
    EndCenters,
    All,
}

/// Persisted Part Modify Slot recipe (Part program version 53).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SlotRecipe {
    pub version: u32,
    pub body: Vec<u8>,
    pub face: Vec<u8>,
    pub position: HolePosition,
    pub shape: SlotShape,
    pub length: DatumLength,
    pub width: DatumLength,
    #[serde(default)]
    pub rotation_deg: f64,
    pub extent: HoleExtent,
    #[serde(default)]
    pub kind: HoleKind,
    #[serde(default)]
    pub seat: SlotSeat,
    #[serde(default)]
    pub end_mill: SlotEndMill,
    #[serde(default)]
    pub center_marks: SlotCenterMarks,
    #[serde(default)]
    pub direction_flipped: bool,
    #[serde(default)]
    pub extend_start: bool,
    #[serde(default)]
    pub imate: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub fastener: Option<HoleFastenerSpec>,
}

/// Validated authored Slot definition.
#[derive(Clone, Debug, PartialEq)]
pub struct SlotDefinition {
    recipe: SlotRecipe,
    body_source: FeatureId,
    face_source: FeatureId,
}

/// Resolved numeric Slot inputs ready for Martensite evaluation.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedSlot {
    pub position: Option<[f64; 3]>,
    pub shape: SlotShape,
    pub length: f64,
    pub width: f64,
    pub rotation_deg: f64,
    pub depth: Option<f64>,
    pub termination_surface: Option<HoleTerminationSurface>,
    pub seat: ResolvedSlotSeat,
    pub end_mill: SlotEndMill,
    pub direction_flipped: bool,
    pub extend_start: bool,
}

impl SlotDefinition {
    /// Validate and construct a Slot definition from its serialized recipe.
    pub fn from_recipe(recipe: SlotRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "slot recipe version {}: expected 1",
                recipe.version
            ));
        }
        let (document, body_source) = reference_identity(&recipe.body, 4, "slot body")?;
        let (face_document, face_source) = reference_identity(&recipe.face, 1, "slot face")?;
        if document != face_document {
            return Err(format!(
                "slot face document {face_document:?}: expected body document {document:?}"
            ));
        }
        if !recipe.rotation_deg.is_finite() {
            return Err(format!(
                "slot rotation {}: expected finite degrees",
                recipe.rotation_deg
            ));
        }
        validate_length_source(recipe.length, "slot length")?;
        validate_length_source(recipe.width, "slot width")?;
        if let HoleExtent::Distance(depth) = recipe.extent {
            validate_length_source(depth, "slot depth")?;
        }
        match &recipe.seat {
            SlotSeat::None => {}
            SlotSeat::Counterbore { width, depth } | SlotSeat::Spotface { width, depth } => {
                validate_length_source(*width, "slot seat width")?;
                validate_length_source(*depth, "slot seat depth")?;
            }
            SlotSeat::Countersink { width, angle_deg } => {
                validate_length_source(*width, "slot countersink width")?;
                if !angle_deg.is_finite() || *angle_deg <= 0.0 || *angle_deg >= 180.0 {
                    return Err(format!(
                        "slot countersink angle {angle_deg}: expected angle in (0, 180)"
                    ));
                }
            }
        }
        if let SlotEndMill::Point { angle_deg } = recipe.end_mill
            && (!angle_deg.is_finite() || angle_deg <= 0.0 || angle_deg >= 180.0)
        {
            return Err(format!(
                "slot end mill angle {angle_deg}: expected angle in (0, 180)"
            ));
        }
        Ok(Self {
            body_source,
            face_source,
            recipe,
        })
    }

    pub fn recipe(&self) -> SlotRecipe {
        self.recipe.clone()
    }

    pub fn body(&self) -> &[u8] {
        &self.recipe.body
    }

    pub fn face(&self) -> &[u8] {
        &self.recipe.face
    }

    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }

    pub fn reads(&self) -> FeatureReadFootprint {
        let mut keys = BTreeSet::new();
        keys.insert(FeatureReadKey::Feature(self.body_source));
        keys.insert(FeatureReadKey::Feature(self.face_source));
        for len in [Some(self.recipe.length), Some(self.recipe.width)] {
            if let Some(DatumLength::Parameter(id)) = len {
                keys.insert(FeatureReadKey::Parameter(id));
            }
        }
        if let HoleExtent::Distance(DatumLength::Parameter(id)) = self.recipe.extent {
            keys.insert(FeatureReadKey::Parameter(id));
        }
        if let HolePosition::ModelPoint(coords) = self.recipe.position {
            for c in coords {
                if let DatumLength::Parameter(id) = c {
                    keys.insert(FeatureReadKey::Parameter(id));
                }
            }
        }
        match self.recipe.seat {
            SlotSeat::None => {}
            SlotSeat::Counterbore { width, depth } | SlotSeat::Spotface { width, depth } => {
                if let DatumLength::Parameter(id) = width {
                    keys.insert(FeatureReadKey::Parameter(id));
                }
                if let DatumLength::Parameter(id) = depth {
                    keys.insert(FeatureReadKey::Parameter(id));
                }
            }
            SlotSeat::Countersink { width, .. } => {
                if let DatumLength::Parameter(id) = width {
                    keys.insert(FeatureReadKey::Parameter(id));
                }
            }
        }
        FeatureReadFootprint::new(keys, false)
    }

    pub fn resolve(&self, parameters: &ParameterCollection<u64>) -> Result<ResolvedSlot, String> {
        let length = positive(self.recipe.length, parameters, "length")?;
        let width = positive(self.recipe.width, parameters, "width")?;
        if length + 1e-9 < width {
            return Err(format!(
                "slot length {length} must be greater than or equal to width {width}"
            ));
        }
        let position = match self.recipe.position {
            HolePosition::FaceCentroid => None,
            HolePosition::ModelPoint(coords) => Some([
                resolve_coord(coords[0], parameters)?,
                resolve_coord(coords[1], parameters)?,
                resolve_coord(coords[2], parameters)?,
            ]),
        };
        let (depth, termination_surface) = match &self.recipe.extent {
            HoleExtent::ThroughAll => (None, None),
            HoleExtent::Distance(d) => (Some(positive(*d, parameters, "depth")?), None),
            HoleExtent::ToSurface(surface) => (None, Some(surface.clone())),
        };
        let seat = match &self.recipe.seat {
            SlotSeat::None => ResolvedSlotSeat::None,
            SlotSeat::Counterbore {
                width: sw,
                depth: sd,
            } => {
                let sw = positive(*sw, parameters, "counterbore width")?;
                if sw <= width {
                    return Err(format!(
                        "slot counterbore width {sw} must exceed slot width {width}"
                    ));
                }
                ResolvedSlotSeat::Counterbore {
                    width: sw,
                    depth: positive(*sd, parameters, "counterbore depth")?,
                }
            }
            SlotSeat::Spotface {
                width: sw,
                depth: sd,
            } => {
                let sw = positive(*sw, parameters, "spotface width")?;
                if sw <= width {
                    return Err(format!(
                        "slot spotface width {sw} must exceed slot width {width}"
                    ));
                }
                ResolvedSlotSeat::Spotface {
                    width: sw,
                    depth: positive(*sd, parameters, "spotface depth")?,
                }
            }
            SlotSeat::Countersink {
                width: sw,
                angle_deg,
            } => {
                let sw = positive(*sw, parameters, "countersink width")?;
                if sw <= width {
                    return Err(format!(
                        "slot countersink width {sw} must exceed slot width {width}"
                    ));
                }
                ResolvedSlotSeat::Countersink {
                    width: sw,
                    angle_deg: *angle_deg,
                }
            }
        };
        Ok(ResolvedSlot {
            position,
            shape: self.recipe.shape,
            length,
            width,
            rotation_deg: self.recipe.rotation_deg,
            depth,
            termination_surface,
            seat,
            end_mill: self.recipe.end_mill,
            direction_flipped: self.recipe.direction_flipped,
            extend_start: self.recipe.extend_start,
        })
    }
}

fn validate_length_source(length: DatumLength, label: &str) -> Result<(), String> {
    length
        .validate_source()
        .map_err(|cause| format!("{label} {length:?}: expected finite length source: {cause}"))?;
    if length.parameter() == Some(0) {
        return Err(format!("{label}: expected nonzero parameter id"));
    }
    if let DatumLength::Constant(v) = length
        && (!v.is_finite() || v <= 0.0)
    {
        return Err(format!("{label} {v}: expected finite positive length"));
    }
    Ok(())
}

fn positive(
    length: DatumLength,
    parameters: &ParameterCollection<u64>,
    label: &str,
) -> Result<f64, String> {
    let value = length
        .resolve(parameters)
        .map_err(|cause| format!("slot {label} {length:?}: {cause}"))?;
    if !value.is_finite() || value <= 0.0 {
        return Err(format!(
            "slot {label} {value}: expected finite positive length"
        ));
    }
    Ok(value)
}

fn resolve_coord(
    length: DatumLength,
    parameters: &ParameterCollection<u64>,
) -> Result<f64, String> {
    length
        .resolve(parameters)
        .map_err(|cause| format!("slot coordinate {length:?}: {cause}"))
}
