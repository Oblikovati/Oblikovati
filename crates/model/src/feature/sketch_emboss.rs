// SPDX-License-Identifier: GPL-2.0-only
//! Authored Emboss inputs (raise, engrave, or level-to-plane), separate from kernel results.
use crate::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId, FeatureReadFootprint, FeatureReadKey,
        fillet::reference_identity,
        sketch_prism::SketchPrismSelection,
        work_geometry::{DatumAngle, DatumLength, WorkGeometryEvaluation},
    },
    history::{DocumentSnapshot, ResolvedDocumentSketch},
};

/// Emboss flavour (raise off face, engrave into face, or level to sketch plane).
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum EmbossType {
    #[default]
    EmbossFromFace,
    EngraveFromFace,
    EmbossEngraveFromPlane,
}

/// Serialized Emboss recipe stored inside the Part feature program (v46).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchEmbossRecipe {
    pub version: u32,
    pub sketch: DocumentSketchId,
    pub selection: SketchPrismSelection,
    pub depth: DatumLength,
    #[serde(default)]
    pub emboss_type: EmbossType,
    #[serde(default)]
    pub direction_reversed: bool,
    pub taper: DatumAngle,
    #[serde(default)]
    pub wrap_to_face: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub wrap_face: Option<Vec<u8>>,
    pub target_body: Vec<u8>,
}

/// Validated authored Emboss definition referencing an existing target body.
#[derive(Clone, Debug)]
pub struct SketchEmbossDefinition {
    sketch: DocumentSketchId,
    selection: SketchPrismSelection,
    depth: DatumLength,
    emboss_type: EmbossType,
    direction_reversed: bool,
    taper: DatumAngle,
    wrap_to_face: bool,
    wrap_face: Option<Vec<u8>>,
    target_body: Vec<u8>,
    target_feature: FeatureId,
}

/// Resolved Emboss inputs ready for certified Martensite evaluation.
#[derive(Debug)]
pub struct ResolvedSketchEmboss<'a> {
    pub sketch: ResolvedDocumentSketch<'a>,
    pub selection: &'a SketchPrismSelection,
    pub depth: f64,
    pub emboss_type: EmbossType,
    pub direction_reversed: bool,
    pub taper: f64,
    pub wrap_to_face: bool,
    pub wrap_face: Option<&'a [u8]>,
}

impl SketchEmbossDefinition {
    /// Validate and construct an Emboss definition from its versioned recipe.
    pub fn from_recipe(recipe: &SketchEmbossRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "emboss recipe version {}: expected 1",
                recipe.version
            ));
        }
        if recipe.selection.outer.is_empty() {
            return Err("emboss profile: expected a nonempty outer boundary".into());
        }
        recipe
            .depth
            .validate_source()
            .map_err(|cause| format!("emboss depth: {cause}"))?;
        recipe
            .taper
            .validate_source()
            .map_err(|cause| format!("emboss taper: {cause}"))?;
        if recipe.wrap_to_face {
            if recipe.emboss_type == EmbossType::EmbossEngraveFromPlane {
                return Err(
                    "emboss: Wrap to Face is not supported for Emboss/Engrave from Plane".into(),
                );
            }
            let Some(face) = &recipe.wrap_face else {
                return Err("emboss: Wrap to Face requires a selected face".into());
            };
            reference_identity(face, 1, "emboss wrap face")?;
        }
        let (_, target_feature) = reference_identity(&recipe.target_body, 4, "emboss target body")?;
        Ok(Self {
            sketch: recipe.sketch,
            selection: recipe.selection.clone(),
            depth: recipe.depth,
            emboss_type: recipe.emboss_type,
            direction_reversed: recipe.direction_reversed,
            taper: recipe.taper,
            wrap_to_face: recipe.wrap_to_face,
            wrap_face: recipe.wrap_face.clone(),
            target_body: recipe.target_body.clone(),
            target_feature,
        })
    }

    pub fn recipe(&self) -> SketchEmbossRecipe {
        SketchEmbossRecipe {
            version: 1,
            sketch: self.sketch,
            selection: self.selection.clone(),
            depth: self.depth,
            emboss_type: self.emboss_type,
            direction_reversed: self.direction_reversed,
            taper: self.taper,
            wrap_to_face: self.wrap_to_face,
            wrap_face: self.wrap_face.clone(),
            target_body: self.target_body.clone(),
        }
    }

    pub fn sketch(&self) -> DocumentSketchId {
        self.sketch
    }

    pub fn selection(&self) -> &SketchPrismSelection {
        &self.selection
    }

    pub fn depth(&self) -> DatumLength {
        self.depth
    }

    pub fn emboss_type(&self) -> EmbossType {
        self.emboss_type
    }

    pub fn direction_reversed(&self) -> bool {
        self.direction_reversed
    }

    pub fn taper(&self) -> DatumAngle {
        self.taper
    }

    pub fn wrap_to_face(&self) -> bool {
        self.wrap_to_face
    }

    pub fn wrap_face(&self) -> Option<&[u8]> {
        self.wrap_face.as_deref()
    }

    pub fn target_body(&self) -> (&[u8], FeatureId) {
        (&self.target_body, self.target_feature)
    }

    pub fn same_recipe(&self, other: &Self) -> bool {
        self.sketch == other.sketch
            && self.selection == other.selection
            && self.depth.same_recipe(other.depth)
            && self.emboss_type == other.emboss_type
            && self.direction_reversed == other.direction_reversed
            && self.taper.same_recipe(other.taper)
            && self.wrap_to_face == other.wrap_to_face
            && self.wrap_face == other.wrap_face
            && self.target_body == other.target_body
    }

    pub fn source_reads(&self) -> FeatureReadFootprint {
        let mut keys = Vec::new();
        if let Some(id) = self.depth.parameter() {
            keys.push(FeatureReadKey::Parameter(id));
        }
        if let Some(id) = self.taper.parameter_id() {
            keys.push(FeatureReadKey::Parameter(id));
        }
        keys.push(FeatureReadKey::Feature(self.target_feature));
        FeatureReadFootprint::new(keys, false)
    }

    pub fn reads(&self, snapshot: &DocumentSnapshot) -> Result<FeatureReadFootprint, String> {
        let sketch = snapshot
            .sketches
            .get(self.sketch)
            .map_err(|cause| format!("emboss sketch: {cause}"))?;
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
    ) -> Result<ResolvedSketchEmboss<'a>, String> {
        let sketch = snapshot
            .require_converged_sketch_with_datums(self.sketch, datums)
            .map_err(|cause| format!("emboss sketch {}: {cause}", self.sketch.0))?;
        let depth = self
            .depth
            .resolve(&snapshot.parameters)
            .map_err(|cause| format!("emboss depth: {cause}"))?;
        if !depth.is_finite() || depth <= 0.0 {
            return Err(format!("emboss depth {depth}: expected a positive length"));
        }
        let taper = self
            .taper
            .resolve(&snapshot.parameters)
            .map_err(|cause| format!("emboss taper: {cause}"))?;
        if !taper.is_finite() || taper.abs() >= std::f64::consts::FRAC_PI_2 {
            return Err(format!(
                "emboss taper {taper}: expected an angle within (-90, 90) deg"
            ));
        }
        Ok(ResolvedSketchEmboss {
            sketch,
            selection: &self.selection,
            depth,
            emboss_type: self.emboss_type,
            direction_reversed: self.direction_reversed,
            taper,
            wrap_to_face: self.wrap_to_face,
            wrap_face: self.wrap_face.as_deref(),
        })
    }
}

impl PartialEq for SketchEmbossDefinition {
    fn eq(&self, other: &Self) -> bool {
        self.same_recipe(other)
    }
}
