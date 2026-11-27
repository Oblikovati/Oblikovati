// SPDX-License-Identifier: GPL-2.0-only
//! Authored Decal inputs (sketch image applied to a named solid face, Part program v49).
use crate::{
    document_sketch::{DocumentSketchId, SketchImage},
    feature::{
        FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity,
        work_geometry::WorkGeometryEvaluation,
    },
    history::{DocumentSnapshot, ResolvedDocumentSketch},
    sketch::SketchEntityId,
};

/// Serialized Decal recipe stored inside the Part feature program (v49).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchDecalRecipe {
    pub version: u32,
    pub sketch: DocumentSketchId,
    pub image: SketchEntityId,
    pub target_face: Vec<u8>,
    pub target_body: Vec<u8>,
    #[serde(default = "default_true")]
    pub chain_faces: bool,
    #[serde(default)]
    pub wrap_to_face: bool,
}

fn default_true() -> bool {
    true
}

/// Validated authored Decal definition referencing a sketch image and a named target face/body.
#[derive(Clone, Debug)]
pub struct SketchDecalDefinition {
    recipe: SketchDecalRecipe,
    target_feature: FeatureId,
}

/// Resolved Decal inputs ready for certified Martensite evaluation.
#[derive(Debug)]
pub struct ResolvedSketchDecal<'a> {
    pub sketch: ResolvedDocumentSketch<'a>,
    pub image: &'a SketchImage,
    pub target_face: &'a [u8],
    pub chain_faces: bool,
    pub wrap_to_face: bool,
}

impl SketchDecalDefinition {
    /// Validate and construct a Decal definition from its versioned recipe.
    pub fn from_recipe(recipe: &SketchDecalRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "decal recipe version {}: expected 1",
                recipe.version
            ));
        }
        if recipe.image.0 == 0 {
            return Err("decal image: expected a nonzero sketch image identity".into());
        }
        reference_identity(&recipe.target_face, 1, "decal target face")?;
        let (_, target_feature) = reference_identity(&recipe.target_body, 4, "decal target body")?;
        Ok(Self {
            recipe: recipe.clone(),
            target_feature,
        })
    }

    pub fn recipe(&self) -> SketchDecalRecipe {
        self.recipe.clone()
    }

    pub fn sketch(&self) -> DocumentSketchId {
        self.recipe.sketch
    }

    pub fn image(&self) -> SketchEntityId {
        self.recipe.image
    }

    pub fn target_face(&self) -> &[u8] {
        &self.recipe.target_face
    }

    pub fn target_body(&self) -> (&[u8], FeatureId) {
        (&self.recipe.target_body, self.target_feature)
    }

    pub fn chain_faces(&self) -> bool {
        self.recipe.chain_faces
    }

    pub fn wrap_to_face(&self) -> bool {
        self.recipe.wrap_to_face
    }

    pub fn same_recipe(&self, other: &Self) -> bool {
        self.recipe == other.recipe
    }

    pub fn source_reads(&self) -> FeatureReadFootprint {
        FeatureReadFootprint::new(vec![FeatureReadKey::Feature(self.target_feature)], false)
    }

    pub fn reads(&self, snapshot: &DocumentSnapshot) -> Result<FeatureReadFootprint, String> {
        let sketch = snapshot
            .sketches
            .get(self.recipe.sketch)
            .map_err(|cause| format!("decal sketch {:?}: {cause}", self.recipe.sketch))?;
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
    ) -> Result<ResolvedSketchDecal<'a>, String> {
        let sketch = snapshot
            .require_converged_sketch_with_datums(self.recipe.sketch, datums)
            .map_err(|cause| format!("decal sketch {}: {cause}", self.recipe.sketch.0))?;
        let image = sketch
            .sketch
            .images()
            .iter()
            .find(|item| item.id == self.recipe.image)
            .ok_or_else(|| {
                format!(
                    "decal image {}: expected an inserted image in sketch {}",
                    self.recipe.image.0, self.recipe.sketch.0
                )
            })?;
        Ok(ResolvedSketchDecal {
            sketch,
            image,
            target_face: &self.recipe.target_face,
            chain_faces: self.recipe.chain_faces,
            wrap_to_face: self.recipe.wrap_to_face,
        })
    }
}

impl PartialEq for SketchDecalDefinition {
    fn eq(&self, other: &Self) -> bool {
        self.same_recipe(other)
    }
}
