// SPDX-License-Identifier: GPL-2.0-only
//! Authored Part Modify Finish feature recipe, validation, and resolution (#3918).
use super::{FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity};
use std::collections::BTreeSet;

/// The five finish categories exposed by the Part Modify Finish command (#3918).
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum FinishType {
    /// Type 1: Appearance / general coating.
    Appearance,
    /// Type 2: Material coating / plating (e.g., `Zn12/Cn/T2`).
    MaterialCoating,
    /// Type 3: Heat treatment (exposes `Hardness` and `Depth`).
    HeatTreatment,
    /// Type 4: Surface texture (e.g., `Barrel Finishing`).
    SurfaceTexture,
    /// Type 5: Paint (defaults to `Prime/Paint RAL 3018`, `Red`, `50 micron`).
    #[default]
    Paint,
}

impl FinishType {
    /// All five finish types in UI button order (1st..5th); example: `FinishType::ALL`.
    pub const ALL: [Self; 5] = [
        Self::Appearance,
        Self::MaterialCoating,
        Self::HeatTreatment,
        Self::SurfaceTexture,
        Self::Paint,
    ];

    /// Return the 1-based ordinal for this finish type; example: `FinishType::Paint.ordinal() == 5`.
    pub const fn ordinal(self) -> u8 {
        match self {
            Self::Appearance => 1,
            Self::MaterialCoating => 2,
            Self::HeatTreatment => 3,
            Self::SurfaceTexture => 4,
            Self::Paint => 5,
        }
    }

    /// Return the user-facing category name; example: `FinishType::Paint.label() == "Paint"`.
    pub const fn label(self) -> &'static str {
        match self {
            Self::Appearance => "Appearance",
            Self::MaterialCoating => "Material Coating",
            Self::HeatTreatment => "Heat Treatment",
            Self::SurfaceTexture => "Surface Texture",
            Self::Paint => "Paint",
        }
    }
}

/// Persisted Part Modify Finish recipe (Part program version 65, source version 55).
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct FinishRecipe {
    pub version: u32,
    pub body: Vec<u8>,
    pub included_faces: Vec<Vec<u8>>,
    #[serde(default)]
    pub excluded_faces: Vec<Vec<u8>>,
    pub finish_type: FinishType,
    pub process: String,
    pub appearance: String,
    #[serde(default)]
    pub thickness: String,
    #[serde(default)]
    pub hardness: String,
    #[serde(default)]
    pub depth: String,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub short_description: String,
    #[serde(default)]
    pub comment: String,
    #[serde(default)]
    pub export_all_parameters: bool,
}

/// Validated authored Finish definition.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct FinishDefinition {
    recipe: FinishRecipe,
    body_source: FeatureId,
    face_sources: Vec<FeatureId>,
}

/// Resolved Finish inputs ready for Martensite evaluation.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ResolvedFinish {
    pub included_faces: Vec<Vec<u8>>,
    pub excluded_faces: Vec<Vec<u8>>,
    pub finish_type: FinishType,
    pub process: String,
    pub appearance: String,
    pub thickness: String,
    pub hardness: String,
    pub depth: String,
    pub description: String,
    pub short_description: String,
    pub comment: String,
    pub export_all_parameters: bool,
}

impl ResolvedFinish {
    /// Return effective faces after subtracting excluded faces from included faces.
    /// Example: `resolved.effective_faces()` preserves authored order.
    pub fn effective_faces(&self) -> Vec<&[u8]> {
        self.included_faces
            .iter()
            .filter(|face| !self.excluded_faces.iter().any(|ex| ex == *face))
            .map(Vec::as_slice)
            .collect()
    }
}

impl FinishDefinition {
    /// Validate and construct a Finish definition from its serialized recipe.
    /// Example: `FinishDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: FinishRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "finish recipe version {}: expected 1",
                recipe.version
            ));
        }
        let (document, body_source) = reference_identity(&recipe.body, 4, "finish body")?;
        if recipe.included_faces.is_empty() {
            return Err("finish included faces: expected at least one included face".into());
        }
        let mut face_sources = Vec::new();
        let mut unique_included = BTreeSet::new();
        for face in &recipe.included_faces {
            if !unique_included.insert(face) {
                return Err("finish included faces: expected unique face references".into());
            }
            let (face_doc, source) = reference_identity(face, 1, "finish included face")?;
            if face_doc != document {
                return Err(format!(
                    "finish included face document {face_doc:?}: expected body document {document:?}"
                ));
            }
            face_sources.push(source);
        }
        let mut unique_excluded = BTreeSet::new();
        for face in &recipe.excluded_faces {
            if !unique_excluded.insert(face) {
                return Err("finish excluded faces: expected unique face references".into());
            }
            let (face_doc, source) = reference_identity(face, 1, "finish excluded face")?;
            if face_doc != document {
                return Err(format!(
                    "finish excluded face document {face_doc:?}: expected body document {document:?}"
                ));
            }
            face_sources.push(source);
        }
        let effective_count = recipe
            .included_faces
            .iter()
            .filter(|face| !unique_excluded.contains(face))
            .count();
        if effective_count == 0 {
            return Err(
                "finish faces: expected at least one included face remaining after exclusion"
                    .into(),
            );
        }
        Ok(Self {
            recipe,
            body_source,
            face_sources,
        })
    }

    /// Return a copy of the serialized Finish recipe; example: `definition.recipe()`.
    pub fn recipe(&self) -> FinishRecipe {
        self.recipe.clone()
    }

    /// Return the encoded target body reference key; example: `definition.body()`.
    pub fn body(&self) -> &[u8] {
        &self.recipe.body
    }

    /// Return the producer feature id for the target body; example: `definition.body_source()`.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }

    /// Return the encoded included face keys; example: `definition.included_faces()`.
    pub fn included_faces(&self) -> &[Vec<u8>] {
        &self.recipe.included_faces
    }

    /// Return the encoded excluded face keys; example: `definition.excluded_faces()`.
    pub fn excluded_faces(&self) -> &[Vec<u8>] {
        &self.recipe.excluded_faces
    }

    /// Report every upstream feature read by this Finish definition; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let mut keys = BTreeSet::new();
        keys.insert(FeatureReadKey::Feature(self.body_source));
        for source in &self.face_sources {
            keys.insert(FeatureReadKey::Feature(*source));
        }
        FeatureReadFootprint::new(keys, false)
    }

    /// Resolve validated Finish inputs for kernel evaluation; example: `definition.resolve()?`.
    pub fn resolve(&self) -> Result<ResolvedFinish, String> {
        let resolved = ResolvedFinish {
            included_faces: self.recipe.included_faces.clone(),
            excluded_faces: self.recipe.excluded_faces.clone(),
            finish_type: self.recipe.finish_type,
            process: self.recipe.process.clone(),
            appearance: self.recipe.appearance.clone(),
            thickness: self.recipe.thickness.clone(),
            hardness: self.recipe.hardness.clone(),
            depth: self.recipe.depth.clone(),
            description: self.recipe.description.clone(),
            short_description: self.recipe.short_description.clone(),
            comment: self.recipe.comment.clone(),
            export_all_parameters: self.recipe.export_all_parameters,
        };
        if resolved.effective_faces().is_empty() {
            return Err("finish faces: expected at least one effective included face".into());
        }
        Ok(resolved)
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
    fn finish_recipe_validates_faces_exclusions_and_round_trip() {
        let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let other: DocumentGuid = "22222222-3333-4444-5555-666666666666".parse().unwrap();
        let body = sample_ref(doc, 1, 4, "body");
        let face_a = sample_ref(doc, 1, 1, "end");
        let face_b = sample_ref(doc, 1, 1, "side");

        let valid = FinishRecipe {
            version: 1,
            body: body.clone(),
            included_faces: vec![face_a.clone(), face_b.clone()],
            excluded_faces: vec![face_b.clone()],
            finish_type: FinishType::Paint,
            process: "Prime/Paint RAL 3018".into(),
            appearance: "Red".into(),
            thickness: "50 micron".into(),
            hardness: String::new(),
            depth: String::new(),
            description: "Painted top face".into(),
            short_description: "Paint".into(),
            comment: String::new(),
            export_all_parameters: true,
        };
        let def = FinishDefinition::from_recipe(valid.clone()).unwrap();
        let resolved = def.resolve().unwrap();
        assert_eq!(resolved.effective_faces(), vec![face_a.as_slice()]);
        assert!(resolved.export_all_parameters);
        assert!(
            def.reads()
                .keys()
                .contains(&FeatureReadKey::Feature(FeatureId(1)))
        );

        let all_excluded = FinishRecipe {
            included_faces: vec![face_a.clone()],
            excluded_faces: vec![face_a.clone()],
            ..valid.clone()
        };
        assert!(FinishDefinition::from_recipe(all_excluded).is_err());

        let empty = FinishRecipe {
            included_faces: Vec::new(),
            excluded_faces: Vec::new(),
            ..valid.clone()
        };
        assert!(FinishDefinition::from_recipe(empty).is_err());

        let wrong_version = FinishRecipe {
            version: 2,
            ..valid.clone()
        };
        assert!(FinishDefinition::from_recipe(wrong_version).is_err());

        let wrong_doc = FinishRecipe {
            included_faces: vec![sample_ref(other, 1, 1, "end")],
            excluded_faces: Vec::new(),
            ..valid
        };
        assert!(FinishDefinition::from_recipe(wrong_doc).is_err());
    }
}
