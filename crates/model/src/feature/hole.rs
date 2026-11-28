// SPDX-License-Identifier: GPL-2.0-only
//! Authored straight-hole inputs and their versioned recipe (#4093).
use super::{
    FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity,
    work_geometry::DatumLength,
};
use crate::parameter::ParameterCollection;
mod serde_inputs;
mod types;
pub use types::{
    HoleDrillPoint, HoleFastenerFit, HoleFastenerSpec, HoleKind, HoleOptions, HoleOriginPlane,
    HoleSeat, HoleTerminationSurface, HoleThreadSpec, ResolvedHoleSeat,
};

/// Position on the named placement face in model coordinates.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum HolePosition {
    FaceCentroid,
    ModelPoint([DatumLength; 3]),
}
/// The extent of a straight bore, measured into the selected face.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum HoleExtent {
    ThroughAll,
    Distance(DatumLength),
    ToSurface(HoleTerminationSurface),
}
/// Stable inputs for a straight bore; membership and planarity are checked during replay.
#[derive(Clone, Debug, PartialEq)]
pub struct HoleDefinition {
    recipe: HoleRecipe,
    body_source: FeatureId,
    face_source: FeatureId,
}
/// Versioned authored inputs with no cached geometry.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct HoleRecipe {
    pub version: u32,
    pub body: Vec<u8>,
    pub face: Vec<u8>,
    #[serde(with = "serde_inputs::position")]
    pub position: HolePosition,
    #[serde(with = "serde_inputs::length")]
    pub diameter: DatumLength,
    #[serde(with = "serde_inputs::extent")]
    pub extent: HoleExtent,
    #[serde(default, skip_serializing_if = "HoleOptions::is_default")]
    pub options: HoleOptions,
}
/// Current numeric inputs, resolved from the document parameters.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedHole {
    pub position: Option<[f64; 3]>,
    pub diameter: f64,
    pub depth: Option<f64>,
    pub termination_surface: Option<HoleTerminationSurface>,
    pub seat: ResolvedHoleSeat,
    pub drill_point: HoleDrillPoint,
    pub direction_flipped: bool,
    pub extend_start: bool,
}

impl HoleRecipe {
    /// Return whether this recipe uses v2 Hole features (options or `ToSurface` extent).
    pub fn requires_v2(&self) -> bool {
        !self.options.is_default() || matches!(self.extent, HoleExtent::ToSurface(_))
    }

    /// Return the canonical recipe version (`1` for plain straight bores, `2` for extended holes).
    pub fn canonical_version(&self) -> u32 {
        if self.requires_v2() { 2 } else { 1 }
    }
}

impl HoleDefinition {
    /// Compare authored values without losing signed-zero changes; example: `before.same_recipe(&after)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self.recipe.version == other.recipe.version
            && self.recipe.body == other.recipe.body
            && self.recipe.face == other.recipe.face
            && self.recipe.diameter.same_recipe(other.recipe.diameter)
            && same_position(self.recipe.position, other.recipe.position)
            && same_extent(&self.recipe.extent, &other.recipe.extent)
            && self.recipe.options.same_recipe(&other.recipe.options)
    }
    /// Validate authored sources; example: `HoleDefinition::from_recipe(recipe)?` before adoption.
    pub fn from_recipe(recipe: HoleRecipe) -> Result<Self, String> {
        let expected = recipe.canonical_version();
        if recipe.version != expected {
            return Err(format!(
                "hole recipe version {}: expected version {expected}",
                recipe.version
            ));
        }
        let (document, body_source) = reference_identity(&recipe.body, 4, "hole body")?;
        let (face_document, face_source) = reference_identity(&recipe.face, 1, "hole face")?;
        if document != face_document {
            return Err(format!(
                "hole face document {face_document:?}: expected body document {document:?}"
            ));
        }
        validate_inputs(&recipe)?;
        Ok(Self {
            recipe,
            body_source,
            face_source,
        })
    }
    /// Copy authored inputs for persistence; example: `serde_json::to_string(&hole.recipe())?`.
    pub fn recipe(&self) -> HoleRecipe {
        self.recipe.clone()
    }
    /// Inspect the current source body key; example: `hole.body()` for guarded replay.
    pub fn body(&self) -> &[u8] {
        &self.recipe.body
    }
    /// Inspect the named placement face; example: `hole.face()` for planar resolution.
    pub fn face(&self) -> &[u8] {
        &self.recipe.face
    }
    /// Inspect the source feature; example: `hole.body_source()` for dependency lookup.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }
    /// Collect the feature and parameter dependencies; example: `hole.reads()` before insertion.
    pub fn reads(&self) -> FeatureReadFootprint {
        let features = [self.body_source, self.face_source]
            .into_iter()
            .map(FeatureReadKey::Feature);
        let parameters = input_lengths(&self.recipe)
            .into_iter()
            .filter_map(DatumLength::parameter)
            .map(FeatureReadKey::Parameter);
        FeatureReadFootprint::new(features.chain(parameters), false)
    }
    /// Resolve current dimensions; example: `hole.resolve(&parameters)?` before geometry construction.
    pub fn resolve(&self, parameters: &ParameterCollection<u64>) -> Result<ResolvedHole, String> {
        let position = match self.recipe.position {
            HolePosition::FaceCentroid => None,
            HolePosition::ModelPoint(point) => Some(resolve_point(point, parameters)?),
        };
        let diameter = positive_length(self.recipe.diameter, parameters, "diameter")?;
        let (depth, termination_surface) = match &self.recipe.extent {
            HoleExtent::ThroughAll => (None, None),
            HoleExtent::Distance(value) => {
                (Some(positive_length(*value, parameters, "depth")?), None)
            }
            HoleExtent::ToSurface(surface) => (None, Some(surface.clone())),
        };
        let seat = self.recipe.options.resolve_seat(parameters, diameter)?;
        Ok(ResolvedHole {
            position,
            diameter,
            depth,
            termination_surface,
            seat,
            drill_point: self.recipe.options.drill_point,
            direction_flipped: self.recipe.options.direction_flipped,
            extend_start: self.recipe.options.extend_start,
        })
    }
}

fn same_position(first: HolePosition, second: HolePosition) -> bool {
    match (first, second) {
        (HolePosition::FaceCentroid, HolePosition::FaceCentroid) => true,
        (HolePosition::ModelPoint(first), HolePosition::ModelPoint(second)) => first
            .into_iter()
            .zip(second)
            .all(|(left, right)| left.same_recipe(right)),
        _ => false,
    }
}
fn same_extent(first: &HoleExtent, second: &HoleExtent) -> bool {
    match (first, second) {
        (HoleExtent::ThroughAll, HoleExtent::ThroughAll) => true,
        (HoleExtent::Distance(first), HoleExtent::Distance(second)) => first.same_recipe(*second),
        (HoleExtent::ToSurface(first), HoleExtent::ToSurface(second)) => first == second,
        _ => false,
    }
}

fn input_lengths(recipe: &HoleRecipe) -> Vec<DatumLength> {
    let mut values = vec![recipe.diameter];
    if let HoleExtent::Distance(depth) = recipe.extent {
        values.push(depth);
    }
    if let HolePosition::ModelPoint(point) = recipe.position {
        values.extend(point);
    }
    values.extend(recipe.options.seat_lengths());
    values
}
fn validate_inputs(recipe: &HoleRecipe) -> Result<(), String> {
    for value in input_lengths(recipe) {
        value.validate_source().map_err(|error| {
            format!("hole input {value:?}: expected a finite length source: {error}")
        })?;
        if value.parameter() == Some(0) {
            return Err("hole parameter 0: expected a nonzero parameter id".into());
        }
    }
    validate_positive_source(recipe.diameter, "diameter")?;
    if let HoleExtent::Distance(value) = recipe.extent {
        validate_positive_source(value, "depth")?;
    }
    recipe.options.validate(recipe.diameter)?;
    Ok(())
}
fn validate_positive_source(value: DatumLength, label: &str) -> Result<(), String> {
    if matches!(value, DatumLength::Constant(number) if number <= 0.) {
        return Err(format!(
            "hole {label} {value:?}: expected a positive length"
        ));
    }
    Ok(())
}
fn positive_length(
    value: DatumLength,
    parameters: &ParameterCollection<u64>,
    label: &str,
) -> Result<f64, String> {
    let resolved = value
        .resolve(parameters)
        .map_err(|error| format!("hole {label} {value:?}: expected a finite length: {error}"))?;
    if resolved <= 0. {
        return Err(format!(
            "hole {label} {resolved:?}: expected a positive length"
        ));
    }
    Ok(resolved)
}
fn resolve_point(
    point: [DatumLength; 3],
    parameters: &ParameterCollection<u64>,
) -> Result<[f64; 3], String> {
    let [x, y, z] = point;
    let coordinate = |value: DatumLength| {
        value.resolve(parameters).map_err(|error| {
            format!("hole coordinate {value:?}: expected a finite length: {error}")
        })
    };
    Ok([coordinate(x)?, coordinate(y)?, coordinate(z)?])
}
