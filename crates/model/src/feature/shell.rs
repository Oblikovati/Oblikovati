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

/// Corner join / behavior mode in the Shell Properties panel.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ShellType {
    #[default]
    Sharp,
    Rounded,
}

/// Approximation bias mode when `allow_approximation` is enabled.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum ShellApproximationType {
    #[default]
    Mean,
    NeverTooThin,
    NeverTooThick,
}

/// Approximation tolerance mode when `allow_approximation` is enabled.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ShellApproximationTolerance {
    #[default]
    Optimized,
    Specified,
}

/// Extended v2 Shell options; `Default` produces a byte-identical v1 recipe.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ShellOptions {
    #[serde(default)]
    pub shell_type: ShellType,
    #[serde(default)]
    pub allow_approximation: bool,
    #[serde(default)]
    pub approximation_type: ShellApproximationType,
    #[serde(default)]
    pub approximation_tolerance: ShellApproximationTolerance,
    #[serde(default = "default_approx_percent")]
    pub approximation_percent: f64,
}

const fn default_approx_percent() -> f64 {
    10.0
}

impl Default for ShellOptions {
    fn default() -> Self {
        Self {
            shell_type: ShellType::Sharp,
            allow_approximation: false,
            approximation_type: ShellApproximationType::Mean,
            approximation_tolerance: ShellApproximationTolerance::Optimized,
            approximation_percent: default_approx_percent(),
        }
    }
}

impl ShellOptions {
    /// True when no v2 field diverges from the v1 shell recipe.
    pub fn is_default(&self) -> bool {
        self.shell_type == ShellType::Sharp
            && !self.allow_approximation
            && self.approximation_type == ShellApproximationType::Mean
            && self.approximation_tolerance == ShellApproximationTolerance::Optimized
            && (self.approximation_percent - default_approx_percent()).abs() <= 1e-9
    }
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
    #[serde(default, skip_serializing_if = "ShellOptions::is_default")]
    pub options: ShellOptions,
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
    pub options: ShellOptions,
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
        Self::new_with_options(
            body,
            openings,
            thickness,
            direction,
            face_thicknesses,
            ShellOptions::default(),
        )
    }

    /// Validate named inputs with extended v2 Shell options.
    pub fn new_with_options(
        body: Vec<u8>,
        openings: Vec<Vec<u8>>,
        thickness: DatumLength,
        direction: ShellDirection,
        face_thicknesses: Vec<ShellFaceThickness>,
        options: ShellOptions,
    ) -> Result<Self, String> {
        if !options.approximation_percent.is_finite() || options.approximation_percent <= 0.0 {
            return Err(format!(
                "shell approximation percent {}: expected a finite positive value",
                options.approximation_percent
            ));
        }
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
        let version = if options.is_default() { 1 } else { 2 };
        Ok(Self {
            recipe: ShellRecipe {
                version,
                body,
                openings,
                thickness,
                direction,
                face_thicknesses,
                options,
            },
            body_source,
            sources: sources.into_iter().collect(),
        })
    }

    /// Read the canonical authored recipe; for example, `definition.recipe().openings`.
    pub fn recipe(&self) -> ShellRecipe {
        self.recipe.clone()
    }

    /// Read extended v2 shell options; for example, `definition.options()`.
    pub fn options(&self) -> ShellOptions {
        self.recipe.options
    }

    /// Validate a saved recipe; for example, `ShellDefinition::from_recipe(&saved)`.
    pub fn from_recipe(recipe: &ShellRecipe) -> Result<Self, String> {
        match recipe.version {
            1 => {
                if !recipe.options.is_default() {
                    return Err("shell recipe version 1: unexpected v2 options".into());
                }
                Self::new(
                    recipe.body.clone(),
                    recipe.openings.clone(),
                    recipe.thickness,
                    recipe.direction,
                    recipe.face_thicknesses.clone(),
                )
            }
            2 => Self::new_with_options(
                recipe.body.clone(),
                recipe.openings.clone(),
                recipe.thickness,
                recipe.direction,
                recipe.face_thicknesses.clone(),
                recipe.options,
            ),
            other => Err(format!(
                "shell recipe version {other}: expected version 1 or 2"
            )),
        }
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
            options: self.recipe.options,
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
