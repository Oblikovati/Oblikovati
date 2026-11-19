// SPDX-License-Identifier: GPL-2.0-only
//! Authored shell inputs use exact face references and parameter-backed thickness (#4106).
use super::{FeatureId, FeatureReadFootprint, FeatureReadKey, work_geometry::DatumLength};
use crate::parameter::ParameterCollection;
use std::collections::BTreeSet;

/// The side on which the shell wall is constructed.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ShellDirection {
    #[default]
    Inside,
    Outside,
    Both,
}

/// One retained face with a distinct wall-thickness source.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ShellFaceThickness {
    pub face: Vec<u8>,
    pub thickness: DatumLength,
}

/// Versioned Shell inputs, independent of the computed body.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ShellRecipe {
    pub version: u32,
    pub body: Vec<u8>,
    pub openings: Vec<Vec<u8>>,
    pub thickness: DatumLength,
    pub direction: ShellDirection,
    pub face_thicknesses: Vec<ShellFaceThickness>,
}

/// A validated, canonical Shell recipe and its source-feature dependencies.
#[derive(Clone, Debug, PartialEq)]
pub struct ShellDefinition {
    recipe: ShellRecipe,
    body_source: FeatureId,
    sources: Vec<FeatureId>,
}

/// Numeric Shell inputs resolved against the current document parameters.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedShell {
    pub thickness: f64,
    pub direction: ShellDirection,
    pub face_thicknesses: Vec<(Vec<u8>, f64)>,
}

impl ShellDefinition {
    /// Validate named inputs; for example, `ShellDefinition::new(body, faces, thickness, direction, overrides)`.
    pub fn new(
        body: Vec<u8>,
        openings: Vec<Vec<u8>>,
        thickness: DatumLength,
        direction: ShellDirection,
        face_thicknesses: Vec<ShellFaceThickness>,
    ) -> Result<Self, String> {
        let (document, body_source) = super::fillet::reference_identity(&body, 4, "shell")?;
        super::fillet::validate_radius_source(thickness, "shell thickness")?;
        let (openings, opening_sources) = super::fillet::validate_topology_keys(
            document,
            body_source,
            openings,
            1,
            "opening face",
            "shell",
        )?;
        let (face_thicknesses, override_sources) =
            validate_overrides(document, body_source, &openings, face_thicknesses)?;
        let sources = opening_sources
            .into_iter()
            .chain(override_sources)
            .collect::<BTreeSet<_>>();
        Ok(Self {
            recipe: ShellRecipe {
                version: 1,
                body,
                openings,
                thickness,
                direction,
                face_thicknesses,
            },
            body_source,
            sources: sources.into_iter().collect(),
        })
    }
    /// Read the canonical authored recipe; for example, `definition.recipe().openings`.
    pub fn recipe(&self) -> ShellRecipe {
        self.recipe.clone()
    }
    /// Validate a saved recipe; for example, `ShellDefinition::from_recipe(&saved)`.
    pub fn from_recipe(recipe: &ShellRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "shell recipe version {}: expected version 1",
                recipe.version
            ));
        }
        Self::new(
            recipe.body.clone(),
            recipe.openings.clone(),
            recipe.thickness,
            recipe.direction,
            recipe.face_thicknesses.clone(),
        )
    }
    /// Identify the source body producer; for example, `definition.body_source()`.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }
    /// Inspect the named source body; for example, `definition.body()` during replay.
    pub fn body(&self) -> &[u8] {
        &self.recipe.body
    }
    /// Inspect the named openings; for example, `definition.openings()` during replay.
    pub fn openings(&self) -> &[Vec<u8>] {
        &self.recipe.openings
    }
    /// Record all feature and scalar dependencies; for example, `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let features = self.sources.iter().copied().map(FeatureReadKey::Feature);
        let base = self
            .recipe
            .thickness
            .parameter()
            .map(FeatureReadKey::Parameter);
        let overrides = self
            .recipe
            .face_thicknesses
            .iter()
            .filter_map(|entry| entry.thickness.parameter().map(FeatureReadKey::Parameter));
        FeatureReadFootprint::new(features.chain(base).chain(overrides), false)
    }
    /// Resolve the base wall thickness; for example, `definition.resolved_thickness(&parameters)`.
    pub fn resolved_thickness(&self, parameters: &ParameterCollection<u64>) -> Result<f64, String> {
        positive_thickness(self.recipe.thickness, parameters)
    }
    /// Resolve every thickness; for example, `definition.resolve(&parameters)` before construction.
    pub fn resolve(&self, parameters: &ParameterCollection<u64>) -> Result<ResolvedShell, String> {
        let mut face_thicknesses = Vec::new();
        for entry in &self.recipe.face_thicknesses {
            face_thicknesses.push((
                entry.face.clone(),
                positive_thickness(entry.thickness, parameters)?,
            ));
        }
        Ok(ResolvedShell {
            thickness: self.resolved_thickness(parameters)?,
            direction: self.recipe.direction,
            face_thicknesses,
        })
    }
}

fn positive_thickness(
    source: DatumLength,
    parameters: &ParameterCollection<u64>,
) -> Result<f64, String> {
    let value = source
        .resolve(parameters)
        .map_err(|cause| cause.to_string())?;
    if !value.is_finite() || value <= 0. {
        return Err(format!(
            "shell thickness {value:?}: expected a finite positive length"
        ));
    }
    Ok(value)
}

fn validate_overrides(
    document: crate::identity::DocumentGuid,
    body: FeatureId,
    openings: &[Vec<u8>],
    overrides: Vec<ShellFaceThickness>,
) -> Result<(Vec<ShellFaceThickness>, Vec<FeatureId>), String> {
    let mut faces = BTreeSet::new();
    let mut sources = BTreeSet::new();
    for entry in &overrides {
        let (owner, source) = super::fillet::reference_identity(&entry.face, 1, "shell override")?;
        if owner != document || openings.contains(&entry.face) || !faces.insert(entry.face.clone())
        {
            return Err(format!(
                "shell override face {:?}: expected a unique retained face in document {document:?}",
                entry.face
            ));
        }
        super::fillet::validate_radius_source(entry.thickness, "shell override thickness")?;
        sources.insert(source);
    }
    sources.insert(body);
    let mut overrides = overrides;
    overrides.sort_by(|left, right| left.face.cmp(&right.face));
    Ok((overrides, sources.into_iter().collect()))
}
