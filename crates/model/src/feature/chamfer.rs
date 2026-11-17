// SPDX-License-Identifier: GPL-2.0-only
//! Authored equal-distance chamfer inputs and versioned recipes (#4046).
use super::{FeatureId, FeatureReadFootprint, fillet::BlendDefinition, work_geometry::DatumLength};
use crate::parameter::ParameterCollection;

/// Named chamfer inputs; the kernel result is constructed during replay.
#[derive(Clone, Debug, PartialEq)]
pub struct ChamferDefinition(BlendDefinition);

/// Versioned chamfer inputs with no cached geometry.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ChamferRecipe {
    pub version: u32,
    pub body: Vec<u8>,
    pub edges: Vec<Vec<u8>>,
    pub distance: DatumLength,
}

impl ChamferDefinition {
    /// Validate named inputs and distance; example: `ChamferDefinition::new(body, edges, distance)?`.
    pub fn new(body: Vec<u8>, edges: Vec<Vec<u8>>, distance: DatumLength) -> Result<Self, String> {
        BlendDefinition::new_blend(body, edges, distance, "chamfer").map(Self)
    }
    /// Read the source body key; example: `definition.body()`.
    pub fn body(&self) -> &[u8] {
        self.0.body()
    }
    /// Read the source feature; example: `definition.body_source()`.
    pub fn body_source(&self) -> FeatureId {
        self.0.body_source()
    }
    /// Read canonical edge names; example: `definition.edges()`.
    pub fn edges(&self) -> &[Vec<u8>] {
        self.0.edges()
    }
    /// Read the authored distance; example: `definition.distance()`.
    pub fn distance(&self) -> DatumLength {
        self.0.radius()
    }
    /// Collect feature and parameter dependencies; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        self.0.reads()
    }
    /// Resolve a positive length; example: `definition.resolved_distance(&parameters)?`.
    pub fn resolved_distance(&self, parameters: &ParameterCollection<u64>) -> Result<f64, String> {
        self.0.resolved_blend_length(parameters, "chamfer")
    }
    /// Save authored inputs; example: `serde_json::to_string(&definition.recipe())?`.
    pub fn recipe(&self) -> ChamferRecipe {
        ChamferRecipe {
            version: 1,
            body: self.body().to_vec(),
            edges: self.edges().to_vec(),
            distance: self.distance(),
        }
    }
    /// Restore validated inputs; example: `ChamferDefinition::from_recipe(&recipe)?`.
    pub fn from_recipe(recipe: &ChamferRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "chamfer recipe version {}: expected version 1",
                recipe.version
            ));
        }
        Self::new(recipe.body.clone(), recipe.edges.clone(), recipe.distance)
    }
}
