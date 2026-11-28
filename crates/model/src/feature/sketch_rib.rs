// SPDX-License-Identifier: GPL-2.0-only
//! Authored Rib inputs (open profile web/fin with thickness, direction, draft, and extension).
use crate::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId, FeatureReadFootprint, FeatureReadKey,
        fillet::reference_identity,
        work_geometry::{DatumAngle, DatumLength, WorkGeometryEvaluation},
    },
    history::{DocumentSnapshot, ResolvedDocumentSketch},
    sketch::SketchEntityId,
};

/// Orientation of the rib wall relative to its profile sketch plane.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum RibDirection {
    /// Normal to the sketch plane (web extruded normal to the profile plane).
    #[default]
    NormalToSketchPlane,
    /// Parallel to the sketch plane (fin extruded in the sketch plane, thickened normal to it).
    ParallelToSketchPlane,
}

/// Which side of the open profile curve the rib thickness grows on.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum RibThickenSide {
    #[default]
    Symmetric,
    Side1,
    Side2,
}

/// Which end holds the nominal rib thickness when draft is applied.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum RibThicknessPlane {
    #[default]
    AtTop,
    AtRoot,
}

/// Serialized Rib recipe stored inside the Part feature program (v48).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchRibRecipe {
    pub version: u32,
    pub sketch: DocumentSketchId,
    pub profile_curves: Vec<SketchEntityId>,
    pub thickness: DatumLength,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub depth: Option<DatumLength>,
    #[serde(default = "default_true")]
    pub to_next: bool,
    #[serde(default)]
    pub direction: RibDirection,
    #[serde(default)]
    pub flip_direction: bool,
    #[serde(default)]
    pub thicken_side: RibThickenSide,
    #[serde(default = "default_true")]
    pub extend_profile: bool,
    pub draft: DatumAngle,
    #[serde(default)]
    pub thickness_plane: RibThicknessPlane,
    pub target_body: Vec<u8>,
}

fn default_true() -> bool {
    true
}

/// Validated authored Rib definition referencing an open sketch profile and a target solid body.
#[derive(Clone, Debug)]
pub struct SketchRibDefinition {
    recipe: SketchRibRecipe,
    target_feature: FeatureId,
}

/// Resolved Rib inputs ready for certified Martensite evaluation.
#[derive(Debug)]
pub struct ResolvedSketchRib<'a> {
    pub sketch: ResolvedDocumentSketch<'a>,
    pub profile_curves: &'a [SketchEntityId],
    pub thickness: f64,
    pub depth: Option<f64>,
    pub to_next: bool,
    pub direction: RibDirection,
    pub flip_direction: bool,
    pub thicken_side: RibThickenSide,
    pub extend_profile: bool,
    pub draft: f64,
    pub thickness_plane: RibThicknessPlane,
}

impl SketchRibDefinition {
    /// Validate and construct a Rib definition from its versioned recipe.
    pub fn from_recipe(recipe: &SketchRibRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!("rib recipe version {}: expected 1", recipe.version));
        }
        if recipe.profile_curves.is_empty() {
            return Err("rib profile: expected at least one open sketch curve".into());
        }
        recipe
            .thickness
            .validate_source()
            .map_err(|cause| format!("rib thickness: {cause}"))?;
        if let Some(depth) = recipe.depth {
            depth
                .validate_source()
                .map_err(|cause| format!("rib depth: {cause}"))?;
        }
        recipe
            .draft
            .validate_source()
            .map_err(|cause| format!("rib draft: {cause}"))?;
        let (_, target_feature) = reference_identity(&recipe.target_body, 4, "rib target body")?;
        Ok(Self {
            recipe: recipe.clone(),
            target_feature,
        })
    }

    pub fn recipe(&self) -> SketchRibRecipe {
        self.recipe.clone()
    }

    pub fn sketch(&self) -> DocumentSketchId {
        self.recipe.sketch
    }

    pub fn profile_curves(&self) -> &[SketchEntityId] {
        &self.recipe.profile_curves
    }

    pub fn thickness(&self) -> DatumLength {
        self.recipe.thickness
    }

    pub fn depth(&self) -> Option<DatumLength> {
        self.recipe.depth
    }

    pub fn to_next(&self) -> bool {
        self.recipe.to_next
    }

    pub fn direction(&self) -> RibDirection {
        self.recipe.direction
    }

    pub fn flip_direction(&self) -> bool {
        self.recipe.flip_direction
    }

    pub fn thicken_side(&self) -> RibThickenSide {
        self.recipe.thicken_side
    }

    pub fn extend_profile(&self) -> bool {
        self.recipe.extend_profile
    }

    pub fn draft(&self) -> DatumAngle {
        self.recipe.draft
    }

    pub fn thickness_plane(&self) -> RibThicknessPlane {
        self.recipe.thickness_plane
    }

    pub fn target_body(&self) -> (&[u8], FeatureId) {
        (&self.recipe.target_body, self.target_feature)
    }

    pub fn same_recipe(&self, other: &Self) -> bool {
        self.recipe == other.recipe
    }

    pub fn source_reads(&self) -> FeatureReadFootprint {
        let mut keys = vec![FeatureReadKey::Feature(self.target_feature)];
        if let Some(param) = self.recipe.thickness.parameter() {
            keys.push(FeatureReadKey::Parameter(param));
        }
        if let Some(param) = self.recipe.depth.and_then(DatumLength::parameter) {
            keys.push(FeatureReadKey::Parameter(param));
        }
        if let Some(param) = self.recipe.draft.parameter_id() {
            keys.push(FeatureReadKey::Parameter(param));
        }
        FeatureReadFootprint::new(keys, false)
    }

    pub fn reads(&self, snapshot: &DocumentSnapshot) -> Result<FeatureReadFootprint, String> {
        let sketch = snapshot
            .sketches
            .get(self.recipe.sketch)
            .map_err(|cause| format!("rib sketch {:?}: {cause}", self.recipe.sketch))?;
        let mut keys = sketch.read_footprint().keys().clone();
        keys.extend(self.source_reads().keys().iter().copied());
        Ok(FeatureReadFootprint::new(
            keys,
            sketch.read_footprint().has_unattributed_reads(),
        ))
    }

    pub fn resolve_for_replay<'a>(
        &'a self,
        snapshot: &'a DocumentSnapshot,
        datums: &WorkGeometryEvaluation,
    ) -> Result<ResolvedSketchRib<'a>, String> {
        let sketch = snapshot
            .require_converged_sketch_with_datums(self.recipe.sketch, datums)
            .map_err(|cause| format!("rib sketch {}: {cause}", self.recipe.sketch.0))?;
        let thickness = self
            .recipe
            .thickness
            .resolve(&snapshot.parameters)
            .map_err(|cause| format!("rib thickness: {cause}"))?;
        if !thickness.is_finite() || thickness <= 0.0 {
            return Err(format!(
                "The expression is out of the expected range: expected positive rib thickness, got {thickness:?}"
            ));
        }
        let depth = match self.recipe.depth {
            Some(length) => {
                let value = length
                    .resolve(&snapshot.parameters)
                    .map_err(|cause| format!("rib depth: {cause}"))?;
                if !value.is_finite() || value <= 0.0 {
                    return Err(format!(
                        "The expression is out of the expected range: expected positive rib depth, got {value:?}"
                    ));
                }
                Some(value)
            }
            None => None,
        };
        let draft = self
            .recipe
            .draft
            .resolve(&snapshot.parameters)
            .map_err(|cause| format!("rib draft: {cause}"))?;
        if !draft.is_finite() || draft.abs() >= std::f64::consts::FRAC_PI_2 {
            return Err(format!(
                "rib draft {draft:?}: expected finite angle strictly within (-90 deg, 90 deg)"
            ));
        }
        Ok(ResolvedSketchRib {
            sketch,
            profile_curves: &self.recipe.profile_curves,
            thickness,
            depth,
            to_next: self.recipe.to_next,
            direction: self.recipe.direction,
            flip_direction: self.recipe.flip_direction,
            thicken_side: self.recipe.thicken_side,
            extend_profile: self.recipe.extend_profile,
            draft,
            thickness_plane: self.recipe.thickness_plane,
        })
    }
}

impl PartialEq for SketchRibDefinition {
    fn eq(&self, other: &Self) -> bool {
        self.same_recipe(other)
    }
}
