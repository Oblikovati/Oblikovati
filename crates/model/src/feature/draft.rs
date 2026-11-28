// SPDX-License-Identifier: GPL-2.0-only
//! Authored Part Modify Draft feature recipe, validation, and parameter resolution (#3918).
use super::{
    FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity,
    hole::HoleOriginPlane,
};
use std::collections::BTreeSet;

/// Neutral-element method used by a Part Modify Draft feature.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum DraftMode {
    /// Draft selected faces keeping their edge on the pull-direction face fixed.
    #[default]
    FixedEdge,
    /// Draft selected faces keeping their intersection with a fixed planar face fixed.
    FixedPlane,
    /// Draft selected faces using an explicit pull direction and parting plane/face.
    PartingTool,
}

/// Angle interpretation in the Draft Options drawer.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum DraftAngleMode {
    #[default]
    Absolute,
    Relative,
}

/// Neutral parting geometry for `DraftMode::PartingTool`.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum DraftPartingTool {
    OriginPlane(HoleOriginPlane),
    WorkPlane(FeatureId),
    Face(Vec<u8>),
}

/// Persisted Part Modify Draft recipe (Part program version 57, source version 47).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct DraftRecipe {
    pub version: u32,
    pub body: Vec<u8>,
    #[serde(default)]
    pub mode: DraftMode,
    pub pull_direction: Vec<u8>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub parting_tool: Option<DraftPartingTool>,
    pub faces: Vec<Vec<u8>>,
    pub angle_deg: f64,
    #[serde(default)]
    pub angle_mode: DraftAngleMode,
    #[serde(default)]
    pub direction_flipped: bool,
    #[serde(default)]
    pub automatic_blending: bool,
}

/// Validated authored Draft definition.
#[derive(Clone, Debug, PartialEq)]
pub struct DraftDefinition {
    recipe: DraftRecipe,
    body_source: FeatureId,
    pull_source: FeatureId,
    face_sources: Vec<FeatureId>,
    parting_source: Option<FeatureId>,
}

/// Resolved numeric Draft inputs ready for Martensite evaluation.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedDraft {
    pub mode: DraftMode,
    pub pull_direction: Vec<u8>,
    pub parting_tool: Option<DraftPartingTool>,
    pub faces: Vec<Vec<u8>>,
    pub angle_deg: f64,
    pub angle_mode: DraftAngleMode,
    pub direction_flipped: bool,
}

impl DraftDefinition {
    /// Validate and construct a Draft definition from its serialized recipe.
    /// Example: `DraftDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: DraftRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "draft recipe version {}: expected 1",
                recipe.version
            ));
        }
        let (document, body_source) = reference_identity(&recipe.body, 4, "draft body")?;
        let (pull_doc, pull_source) =
            reference_identity(&recipe.pull_direction, 1, "draft pull direction")?;
        if pull_doc != document {
            return Err(format!(
                "draft pull direction document {pull_doc:?}: expected body document {document:?}"
            ));
        }
        if recipe.faces.is_empty() {
            return Err("draft faces: expected at least one face to draft".into());
        }
        let mut unique_faces = BTreeSet::new();
        let mut face_sources = Vec::with_capacity(recipe.faces.len());
        for face in &recipe.faces {
            if !unique_faces.insert(face) {
                return Err("draft faces: expected unique face references".into());
            }
            if face == &recipe.pull_direction {
                return Err(
                    "draft faces: pull direction face cannot also be selected as a drafted face"
                        .into(),
                );
            }
            let (face_doc, face_source) = reference_identity(face, 1, "draft face")?;
            if face_doc != document {
                return Err(format!(
                    "draft face document {face_doc:?}: expected body document {document:?}"
                ));
            }
            face_sources.push(face_source);
        }
        let parting_source = match (&recipe.mode, &recipe.parting_tool) {
            (DraftMode::PartingTool, None) => {
                return Err(
                    "draft parting tool: expected an origin plane, work plane, or face in Parting Tool mode".into(),
                );
            }
            (_, Some(DraftPartingTool::OriginPlane(_))) => None,
            (_, Some(DraftPartingTool::WorkPlane(id))) => {
                if id.0 == 0 {
                    return Err("draft parting work plane: expected nonzero feature id".into());
                }
                Some(*id)
            }
            (_, Some(DraftPartingTool::Face(face))) => {
                let (part_doc, part_source) =
                    reference_identity(face, 1, "draft parting tool face")?;
                if part_doc != document {
                    return Err(format!(
                        "draft parting face document {part_doc:?}: expected body document {document:?}"
                    ));
                }
                Some(part_source)
            }
            (_, None) => None,
        };
        validate_angle_deg(recipe.angle_deg)?;
        Ok(Self {
            recipe,
            body_source,
            pull_source,
            face_sources,
            parting_source,
        })
    }

    pub fn recipe(&self) -> DraftRecipe {
        self.recipe.clone()
    }

    pub fn body(&self) -> &[u8] {
        &self.recipe.body
    }

    pub fn pull_direction(&self) -> &[u8] {
        &self.recipe.pull_direction
    }

    pub fn faces(&self) -> &[Vec<u8>] {
        &self.recipe.faces
    }

    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }

    pub fn reads(&self) -> FeatureReadFootprint {
        let mut keys = BTreeSet::new();
        keys.insert(FeatureReadKey::Feature(self.body_source));
        keys.insert(FeatureReadKey::Feature(self.pull_source));
        for source in &self.face_sources {
            keys.insert(FeatureReadKey::Feature(*source));
        }
        if let Some(source) = self.parting_source {
            keys.insert(FeatureReadKey::Feature(source));
        }
        FeatureReadFootprint::new(keys, false)
    }

    pub fn resolve(&self) -> Result<ResolvedDraft, String> {
        validate_angle_deg(self.recipe.angle_deg)?;
        Ok(ResolvedDraft {
            mode: self.recipe.mode,
            pull_direction: self.recipe.pull_direction.clone(),
            parting_tool: self.recipe.parting_tool.clone(),
            faces: self.recipe.faces.clone(),
            angle_deg: self.recipe.angle_deg,
            angle_mode: self.recipe.angle_mode,
            direction_flipped: self.recipe.direction_flipped,
        })
    }
}

/// Validate a Draft angle in degrees; must be strictly inside `(0, 90)`.
pub fn validate_angle_deg(angle_deg: f64) -> Result<(), String> {
    if !angle_deg.is_finite() || angle_deg <= 0.0 || angle_deg >= 90.0 {
        return Err(format!(
            "draft angle {angle_deg}: expected finite angle in (0, 90) degrees; errors during the part update"
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::identity::{DocumentGuid, SemanticReferenceName};

    fn sample_ref(doc: DocumentGuid, id: u64, kind: u32, tag: &str) -> Vec<u8> {
        SemanticReferenceName::new(doc, id, kind, "prism", tag, &[1])
            .unwrap()
            .encode()
            .unwrap()
    }

    #[test]
    fn draft_recipe_validates_angle_and_parting_tool() {
        let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let body = sample_ref(doc, 1, 4, "body");
        let top = sample_ref(doc, 1, 1, "top");
        let side = sample_ref(doc, 1, 1, "side");
        let valid = DraftRecipe {
            version: 1,
            body: body.clone(),
            mode: DraftMode::PartingTool,
            pull_direction: top.clone(),
            parting_tool: Some(DraftPartingTool::OriginPlane(HoleOriginPlane::Xz)),
            faces: vec![side.clone()],
            angle_deg: 2.5,
            angle_mode: DraftAngleMode::Relative,
            direction_flipped: false,
            automatic_blending: true,
        };
        let def = DraftDefinition::from_recipe(valid.clone()).unwrap();
        assert_eq!(def.resolve().unwrap().angle_deg, 2.5);
        assert!(
            def.reads()
                .keys()
                .contains(&FeatureReadKey::Feature(FeatureId(1)))
        );

        let zero = DraftRecipe {
            angle_deg: 0.0,
            ..valid.clone()
        };
        let err = DraftDefinition::from_recipe(zero).unwrap_err();
        assert!(err.contains("(0, 90)"), "{err}");

        let missing_parting = DraftRecipe {
            parting_tool: None,
            ..valid
        };
        assert!(DraftDefinition::from_recipe(missing_parting).is_err());
    }
}
