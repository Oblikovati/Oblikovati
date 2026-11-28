// SPDX-License-Identifier: GPL-2.0-only
//! Authored Part Modify Delete Face feature recipe, validation, and resolution (#3918).
use super::{FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity};
use std::collections::BTreeSet;

/// Persisted Part Modify Delete Face recipe (Part program version 63, source version 53).
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct DeleteFaceRecipe {
    pub version: u32,
    pub body: Vec<u8>,
    pub faces: Vec<Vec<u8>>,
    #[serde(default)]
    pub heal_remaining_faces: bool,
    #[serde(default)]
    pub select_lump_or_void: bool,
}

/// Validated authored Delete Face definition.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct DeleteFaceDefinition {
    recipe: DeleteFaceRecipe,
    body_source: FeatureId,
    face_sources: Vec<FeatureId>,
}

/// Resolved Delete Face inputs ready for Martensite evaluation.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ResolvedDeleteFace {
    pub faces: Vec<Vec<u8>>,
    pub heal_remaining_faces: bool,
    pub select_lump_or_void: bool,
}

impl DeleteFaceDefinition {
    /// Validate and construct a Delete Face definition from its serialized recipe.
    /// Example: `DeleteFaceDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: DeleteFaceRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "delete face recipe version {}: expected 1",
                recipe.version
            ));
        }
        let (document, body_source) = reference_identity(&recipe.body, 4, "delete face body")?;
        if recipe.faces.is_empty() {
            return Err("delete face faces: expected at least one face to delete".into());
        }
        let mut unique_faces = BTreeSet::new();
        let mut face_sources = Vec::with_capacity(recipe.faces.len());
        for face in &recipe.faces {
            if !unique_faces.insert(face) {
                return Err("delete face faces: expected unique face references".into());
            }
            let (face_doc, face_source) = reference_identity(face, 1, "delete face")?;
            if face_doc != document {
                return Err(format!(
                    "delete face document {face_doc:?}: expected body document {document:?}"
                ));
            }
            face_sources.push(face_source);
        }
        Ok(Self {
            recipe,
            body_source,
            face_sources,
        })
    }

    /// Return a copy of the serialized Delete Face recipe; example: `definition.recipe()`.
    pub fn recipe(&self) -> DeleteFaceRecipe {
        self.recipe.clone()
    }

    /// Return the encoded target body reference key; example: `definition.body()`.
    pub fn body(&self) -> &[u8] {
        &self.recipe.body
    }

    /// Return the encoded face reference keys selected for removal; example: `definition.faces()`.
    pub fn faces(&self) -> &[Vec<u8>] {
        &self.recipe.faces
    }

    /// Whether healing of remaining adjacent faces is requested; example: `definition.heal_remaining_faces()`.
    pub fn heal_remaining_faces(&self) -> bool {
        self.recipe.heal_remaining_faces
    }

    /// Whether lump/void selection mode is enabled; example: `definition.select_lump_or_void()`.
    pub fn select_lump_or_void(&self) -> bool {
        self.recipe.select_lump_or_void
    }

    /// Return the producer feature id for the target body; example: `definition.body_source()`.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }

    /// Report every upstream feature read by this Delete Face definition; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let mut keys = BTreeSet::new();
        keys.insert(FeatureReadKey::Feature(self.body_source));
        for source in &self.face_sources {
            keys.insert(FeatureReadKey::Feature(*source));
        }
        FeatureReadFootprint::new(keys, false)
    }

    /// Resolve validated Delete Face parameters for kernel evaluation; example: `definition.resolve()?`.
    pub fn resolve(&self) -> Result<ResolvedDeleteFace, String> {
        if self.recipe.faces.is_empty() {
            return Err("delete face faces: expected at least one face to delete".into());
        }
        Ok(ResolvedDeleteFace {
            faces: self.recipe.faces.clone(),
            heal_remaining_faces: self.recipe.heal_remaining_faces,
            select_lump_or_void: self.recipe.select_lump_or_void,
        })
    }
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
    fn delete_face_recipe_validates_faces_uniqueness_and_document() {
        let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let other: DocumentGuid = "22222222-3333-4444-5555-666666666666".parse().unwrap();
        let body = sample_ref(doc, 1, 4, "body");
        let face_a = sample_ref(doc, 1, 1, "end");
        let face_b = sample_ref(doc, 1, 1, "side");

        let valid = DeleteFaceRecipe {
            version: 1,
            body: body.clone(),
            faces: vec![face_a.clone(), face_b.clone()],
            heal_remaining_faces: true,
            select_lump_or_void: false,
        };
        let def = DeleteFaceDefinition::from_recipe(valid.clone()).unwrap();
        let resolved = def.resolve().unwrap();
        assert_eq!(resolved.faces.len(), 2);
        assert!(resolved.heal_remaining_faces);
        assert!(
            def.reads()
                .keys()
                .contains(&FeatureReadKey::Feature(FeatureId(1)))
        );

        let empty = DeleteFaceRecipe {
            faces: Vec::new(),
            ..valid.clone()
        };
        assert!(DeleteFaceDefinition::from_recipe(empty).is_err());

        let duplicate = DeleteFaceRecipe {
            faces: vec![face_a.clone(), face_a],
            ..valid.clone()
        };
        assert!(DeleteFaceDefinition::from_recipe(duplicate).is_err());

        let wrong_version = DeleteFaceRecipe {
            version: 2,
            ..valid.clone()
        };
        assert!(DeleteFaceDefinition::from_recipe(wrong_version).is_err());

        let wrong_doc = DeleteFaceRecipe {
            faces: vec![sample_ref(other, 1, 1, "end")],
            ..valid
        };
        assert!(DeleteFaceDefinition::from_recipe(wrong_doc).is_err());
    }
}
