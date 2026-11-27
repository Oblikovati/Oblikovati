// SPDX-License-Identifier: GPL-2.0-only
//! Authored Unwrap feature inputs (planar surface unfold, Part program v51).
use crate::feature::{FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity};

/// Orientation alignment applied to the planar unwrapped surface result.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum UnwrapAlignment {
    #[default]
    OriginXY,
    OriginXZ,
    OriginYZ,
    FlippedXY,
}

impl UnwrapAlignment {
    /// Ordered list of alignment buttons displayed in the Unwrap Properties panel.
    pub const ALL: [Self; 4] = [
        Self::OriginXY,
        Self::OriginXZ,
        Self::OriginYZ,
        Self::FlippedXY,
    ];

    /// Short caption for each Alignment orientation button.
    pub const fn label(self) -> &'static str {
        match self {
            Self::OriginXY => "XY",
            Self::OriginXZ => "XZ",
            Self::OriginYZ => "YZ",
            Self::FlippedXY => "Flip",
        }
    }
}

/// Serialized Unwrap recipe stored inside the Part feature program (v51).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct UnwrapRecipe {
    pub version: u32,
    pub target_body: Vec<u8>,
    pub faces: Vec<Vec<u8>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub alignment_vertex: Option<Vec<u8>>,
    #[serde(default = "default_true")]
    pub auto_face_chain: bool,
    #[serde(default = "default_true")]
    pub merge_result_body: bool,
    #[serde(default)]
    pub alignment: UnwrapAlignment,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub linear_result_edges: Vec<Vec<u8>>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub rigid_result_edges: Vec<Vec<u8>>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub seam_edges: Vec<Vec<u8>>,
}

fn default_true() -> bool {
    true
}

/// Validated authored Unwrap definition referencing one or more faces on a live solid.
#[derive(Clone, Debug)]
pub struct UnwrapDefinition {
    recipe: UnwrapRecipe,
    target_feature: FeatureId,
}

/// Resolved Unwrap inputs ready for certified Martensite evaluation.
#[derive(Clone, Debug)]
pub struct ResolvedUnwrap<'a> {
    pub faces: &'a [Vec<u8>],
    pub alignment_vertex: Option<&'a [u8]>,
    pub auto_face_chain: bool,
    pub merge_result_body: bool,
    pub alignment: UnwrapAlignment,
}

impl UnwrapDefinition {
    /// Validate and construct an Unwrap definition from its versioned recipe.
    pub fn from_recipe(recipe: &UnwrapRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "unwrap recipe version {}: expected 1",
                recipe.version
            ));
        }
        if recipe.faces.is_empty() {
            return Err("unwrap faces: expected at least one selected face".into());
        }
        for face in &recipe.faces {
            reference_identity(face, 1, "unwrap face")?;
        }
        if let Some(vertex) = &recipe.alignment_vertex {
            reference_identity(vertex, 3, "unwrap alignment vertex")?;
        }
        let (_, target_feature) = reference_identity(&recipe.target_body, 4, "unwrap target body")?;
        Ok(Self {
            recipe: recipe.clone(),
            target_feature,
        })
    }

    pub fn recipe(&self) -> UnwrapRecipe {
        self.recipe.clone()
    }

    pub fn target_body(&self) -> (&[u8], FeatureId) {
        (&self.recipe.target_body, self.target_feature)
    }

    pub fn faces(&self) -> &[Vec<u8>] {
        &self.recipe.faces
    }

    pub fn alignment_vertex(&self) -> Option<&[u8]> {
        self.recipe.alignment_vertex.as_deref()
    }

    pub fn auto_face_chain(&self) -> bool {
        self.recipe.auto_face_chain
    }

    pub fn merge_result_body(&self) -> bool {
        self.recipe.merge_result_body
    }

    pub fn alignment(&self) -> UnwrapAlignment {
        self.recipe.alignment
    }

    pub fn linear_result_edges(&self) -> &[Vec<u8>] {
        &self.recipe.linear_result_edges
    }

    pub fn rigid_result_edges(&self) -> &[Vec<u8>] {
        &self.recipe.rigid_result_edges
    }

    pub fn seam_edges(&self) -> &[Vec<u8>] {
        &self.recipe.seam_edges
    }

    pub fn same_recipe(&self, other: &Self) -> bool {
        self.recipe == other.recipe
    }

    pub fn source_reads(&self) -> FeatureReadFootprint {
        FeatureReadFootprint::new(vec![FeatureReadKey::Feature(self.target_feature)], false)
    }

    pub fn resolve(&self) -> ResolvedUnwrap<'_> {
        ResolvedUnwrap {
            faces: &self.recipe.faces,
            alignment_vertex: self.recipe.alignment_vertex.as_deref(),
            auto_face_chain: self.recipe.auto_face_chain,
            merge_result_body: self.recipe.merge_result_body,
            alignment: self.recipe.alignment,
        }
    }
}

impl PartialEq for UnwrapDefinition {
    fn eq(&self, other: &Self) -> bool {
        self.same_recipe(other)
    }
}
