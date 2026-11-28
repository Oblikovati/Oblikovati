// SPDX-License-Identifier: GPL-2.0-only
//! Authored chamfer inputs and versioned recipes (v1 equal-distance, v2 modes & partial edges, #4046).
use super::{
    FeatureId, FeatureReadFootprint,
    chamfer_options::{ChamferOptions, ResolvedChamfer},
    fillet::BlendDefinition,
    work_geometry::DatumLength,
};
use crate::parameter::ParameterCollection;

/// Named chamfer inputs; the kernel result is constructed during replay.
#[derive(Clone, Debug, PartialEq)]
pub struct ChamferDefinition {
    base: BlendDefinition,
    options: ChamferOptions,
}

/// Versioned chamfer inputs with no cached geometry.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ChamferRecipe {
    pub version: u32,
    pub body: Vec<u8>,
    pub edges: Vec<Vec<u8>>,
    pub distance: DatumLength,
    #[serde(default, skip_serializing_if = "ChamferOptions::is_default")]
    pub options: ChamferOptions,
}

impl ChamferDefinition {
    /// Validate equal-distance named inputs; example: `ChamferDefinition::new(body, edges, distance)?`.
    pub fn new(body: Vec<u8>, edges: Vec<Vec<u8>>, distance: DatumLength) -> Result<Self, String> {
        Self::with_options(body, edges, distance, ChamferOptions::default())
    }
    /// Validate v2 named chamfer inputs and options.
    pub fn with_options(
        body: Vec<u8>,
        edges: Vec<Vec<u8>>,
        distance: DatumLength,
        options: ChamferOptions,
    ) -> Result<Self, String> {
        let base = BlendDefinition::new_blend(body, edges, distance, "chamfer")?;
        options.validate()?;
        Ok(Self { base, options })
    }
    /// Read the source body key; example: `definition.body()`.
    pub fn body(&self) -> &[u8] {
        self.base.body()
    }
    /// Read the source feature; example: `definition.body_source()`.
    pub fn body_source(&self) -> FeatureId {
        self.base.body_source()
    }
    /// Read canonical edge names; example: `definition.edges()`.
    pub fn edges(&self) -> &[Vec<u8>] {
        self.base.edges()
    }
    /// Read the authored primary distance; example: `definition.distance()`.
    pub fn distance(&self) -> DatumLength {
        self.base.radius()
    }
    /// Read extended v2 chamfer options; example: `definition.options()`.
    pub fn options(&self) -> &ChamferOptions {
        &self.options
    }
    /// Collect feature and parameter dependencies; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let mut reads = self.base.reads();
        self.options.extend_reads(&mut reads);
        reads
    }
    /// Resolve a positive primary length; example: `definition.resolved_distance(&parameters)?`.
    pub fn resolved_distance(&self, parameters: &ParameterCollection<u64>) -> Result<f64, String> {
        self.base.resolved_blend_length(parameters, "chamfer")
    }
    /// Resolve all numeric distances, angle, and partial lengths for kernel execution.
    pub fn resolve(
        &self,
        parameters: &ParameterCollection<u64>,
    ) -> Result<ResolvedChamfer, String> {
        let distance = self.resolved_distance(parameters)?;
        self.options.resolve(distance, parameters)
    }
    /// Save authored inputs; v1 stays byte-identical when options are default.
    pub fn recipe(&self) -> ChamferRecipe {
        let version = if self.options.is_default() { 1 } else { 2 };
        ChamferRecipe {
            version,
            body: self.body().to_vec(),
            edges: self.edges().to_vec(),
            distance: self.distance(),
            options: self.options.clone(),
        }
    }
    /// Restore validated inputs; example: `ChamferDefinition::from_recipe(&recipe)?`.
    pub fn from_recipe(recipe: &ChamferRecipe) -> Result<Self, String> {
        match recipe.version {
            1 => {
                if !recipe.options.is_default() {
                    return Err("chamfer recipe version 1: unexpected v2 options".into());
                }
                Self::new(recipe.body.clone(), recipe.edges.clone(), recipe.distance)
            }
            2 => Self::with_options(
                recipe.body.clone(),
                recipe.edges.clone(),
                recipe.distance,
                recipe.options.clone(),
            ),
            other => Err(format!(
                "chamfer recipe version {other}: expected version 1 or 2"
            )),
        }
    }
}
