// SPDX-License-Identifier: GPL-2.0-only
//! Authored Part Surface Replace Face feature recipe, validation, and resolution (#3918).
use super::{FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity};
use std::collections::BTreeSet;

/// One replacement face reference paired with its owning body reference key.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ReplaceFaceTarget {
    pub body: Vec<u8>,
    pub face: Vec<u8>,
}

/// Persisted Part Surface Replace Face recipe (Part program version 72, source version 62).
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ReplaceFaceRecipe {
    pub version: u32,
    pub body: Vec<u8>,
    pub existing_faces: Vec<Vec<u8>>,
    pub new_faces: Vec<ReplaceFaceTarget>,
    #[serde(default)]
    pub automatic_face_chain: bool,
}

/// Validated authored Replace Face definition.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ReplaceFaceDefinition {
    recipe: ReplaceFaceRecipe,
    body_source: FeatureId,
    existing_face_sources: Vec<FeatureId>,
    new_face_body_sources: Vec<FeatureId>,
    new_face_sources: Vec<FeatureId>,
}

/// Resolved Replace Face inputs ready for Martensite evaluation.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ResolvedReplaceFace {
    pub existing_faces: Vec<Vec<u8>>,
    pub new_faces: Vec<Vec<u8>>,
    pub automatic_face_chain: bool,
}

impl ReplaceFaceDefinition {
    /// Validate and construct a Replace Face definition from its serialized recipe.
    /// Example: `ReplaceFaceDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: ReplaceFaceRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "replace face recipe version {}: expected 1",
                recipe.version
            ));
        }
        let (document, body_source) = reference_identity(&recipe.body, 4, "replace face body")?;
        if recipe.existing_faces.is_empty() {
            return Err(
                "replace face existing_faces: expected at least one existing face to replace"
                    .into(),
            );
        }
        if recipe.new_faces.is_empty() {
            return Err(
                "replace face new_faces: expected at least one replacement face or surface".into(),
            );
        }
        let mut unique_existing = BTreeSet::new();
        let mut existing_face_sources = Vec::with_capacity(recipe.existing_faces.len());
        for face in &recipe.existing_faces {
            if !unique_existing.insert(face.as_slice()) {
                return Err(
                    "replace face existing_faces: expected unique existing face references".into(),
                );
            }
            let (face_doc, face_source) = reference_identity(face, 1, "replace face existing")?;
            if face_doc != document {
                return Err(format!(
                    "replace face existing face document {face_doc:?}: expected body document {document:?}"
                ));
            }
            existing_face_sources.push(face_source);
        }
        let mut unique_new = BTreeSet::new();
        let mut new_face_body_sources = Vec::with_capacity(recipe.new_faces.len());
        let mut new_face_sources = Vec::with_capacity(recipe.new_faces.len());
        for target in &recipe.new_faces {
            if !unique_new.insert(target.face.as_slice()) {
                return Err(
                    "replace face new_faces: expected unique replacement face references".into(),
                );
            }
            if unique_existing.contains(target.face.as_slice()) {
                return Err(
                    "replace face new_faces: replacement face must differ from existing faces"
                        .into(),
                );
            }
            let (new_body_doc, new_body_source) =
                reference_identity(&target.body, 4, "replace face new body")?;
            if new_body_doc != document {
                return Err(format!(
                    "replace face new body document {new_body_doc:?}: expected body document {document:?}"
                ));
            }
            let (new_face_doc, new_face_source) =
                reference_identity(&target.face, 1, "replace face new")?;
            if new_face_doc != document {
                return Err(format!(
                    "replace face new face document {new_face_doc:?}: expected body document {document:?}"
                ));
            }
            new_face_body_sources.push(new_body_source);
            new_face_sources.push(new_face_source);
        }
        Ok(Self {
            recipe,
            body_source,
            existing_face_sources,
            new_face_body_sources,
            new_face_sources,
        })
    }

    /// Return a copy of the serialized Replace Face recipe; example: `definition.recipe()`.
    pub fn recipe(&self) -> ReplaceFaceRecipe {
        self.recipe.clone()
    }

    /// Return the encoded target body reference key; example: `definition.body()`.
    pub fn body(&self) -> &[u8] {
        &self.recipe.body
    }

    /// Return the encoded existing face reference keys; example: `definition.existing_faces()`.
    pub fn existing_faces(&self) -> &[Vec<u8>] {
        &self.recipe.existing_faces
    }

    /// Return the replacement face targets; example: `definition.new_faces()`.
    pub fn new_faces(&self) -> &[ReplaceFaceTarget] {
        &self.recipe.new_faces
    }

    /// Whether Automatic Face Chain is enabled; example: `definition.automatic_face_chain()`.
    pub fn automatic_face_chain(&self) -> bool {
        self.recipe.automatic_face_chain
    }

    /// Return the producer feature id for the target body; example: `definition.body_source()`.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }

    /// Return producer feature ids aligned with `new_faces()`; example: `definition.new_face_body_sources()`.
    pub fn new_face_body_sources(&self) -> &[FeatureId] {
        &self.new_face_body_sources
    }

    /// Report every upstream feature read by this Replace Face definition; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let mut keys = BTreeSet::new();
        keys.insert(FeatureReadKey::Feature(self.body_source));
        for &source in &self.existing_face_sources {
            keys.insert(FeatureReadKey::Feature(source));
        }
        for &source in &self.new_face_body_sources {
            keys.insert(FeatureReadKey::Feature(source));
        }
        for &source in &self.new_face_sources {
            keys.insert(FeatureReadKey::Feature(source));
        }
        FeatureReadFootprint::new(keys, false)
    }

    /// Resolve validated Replace Face parameters for kernel evaluation; example: `definition.resolve()?`.
    pub fn resolve(&self) -> Result<ResolvedReplaceFace, String> {
        if self.recipe.existing_faces.is_empty() {
            return Err(
                "replace face existing_faces: expected at least one existing face to replace"
                    .into(),
            );
        }
        if self.recipe.new_faces.is_empty() {
            return Err(
                "replace face new_faces: expected at least one replacement face or surface".into(),
            );
        }
        Ok(ResolvedReplaceFace {
            existing_faces: self.recipe.existing_faces.clone(),
            new_faces: self
                .recipe
                .new_faces
                .iter()
                .map(|target| target.face.clone())
                .collect(),
            automatic_face_chain: self.recipe.automatic_face_chain,
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
    fn replace_face_recipe_validates_existing_new_uniqueness_and_document() {
        let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let other: DocumentGuid = "22222222-3333-4444-5555-666666666666".parse().unwrap();
        let body = sample_ref(doc, 1, 4, "body");
        let existing = sample_ref(doc, 1, 1, "end");
        let offset_body = sample_ref(doc, 2, 4, "srf-body");
        let offset_face = sample_ref(doc, 2, 1, "srf-face");

        let valid = ReplaceFaceRecipe {
            version: 1,
            body: body.clone(),
            existing_faces: vec![existing.clone()],
            new_faces: vec![ReplaceFaceTarget {
                body: offset_body.clone(),
                face: offset_face.clone(),
            }],
            automatic_face_chain: true,
        };
        let def = ReplaceFaceDefinition::from_recipe(valid.clone()).unwrap();
        let resolved = def.resolve().unwrap();
        assert_eq!(resolved.existing_faces.len(), 1);
        assert_eq!(resolved.new_faces.len(), 1);
        assert!(resolved.automatic_face_chain);
        assert!(
            def.reads()
                .keys()
                .contains(&FeatureReadKey::Feature(FeatureId(1)))
        );
        assert!(
            def.reads()
                .keys()
                .contains(&FeatureReadKey::Feature(FeatureId(2)))
        );

        let empty_existing = ReplaceFaceRecipe {
            existing_faces: Vec::new(),
            ..valid.clone()
        };
        assert!(ReplaceFaceDefinition::from_recipe(empty_existing).is_err());

        let empty_new = ReplaceFaceRecipe {
            new_faces: Vec::new(),
            ..valid.clone()
        };
        assert!(ReplaceFaceDefinition::from_recipe(empty_new).is_err());

        let same_face = ReplaceFaceRecipe {
            new_faces: vec![ReplaceFaceTarget {
                body: body.clone(),
                face: existing.clone(),
            }],
            ..valid.clone()
        };
        assert!(ReplaceFaceDefinition::from_recipe(same_face).is_err());

        let wrong_version = ReplaceFaceRecipe {
            version: 2,
            ..valid.clone()
        };
        assert!(ReplaceFaceDefinition::from_recipe(wrong_version).is_err());

        let wrong_doc = ReplaceFaceRecipe {
            new_faces: vec![ReplaceFaceTarget {
                body: sample_ref(other, 2, 4, "srf-body"),
                face: sample_ref(other, 2, 1, "srf-face"),
            }],
            ..valid
        };
        assert!(ReplaceFaceDefinition::from_recipe(wrong_doc).is_err());
    }
}
