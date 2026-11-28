// SPDX-License-Identifier: GPL-2.0-only
//! Authored Surface Trim feature recipe, validation, and resolution (#3918).
use super::{
    FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity,
    hole::HoleOriginPlane,
};
use crate::sketch::SketchPlane;
use std::{collections::BTreeSet, fmt};

/// Cutting tool reference for trimming an open surface body.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum SurfaceTrimTool {
    OriginPlane(HoleOriginPlane),
    WorkPlane(FeatureId),
    SurfaceFace { body: Vec<u8>, face: Vec<u8> },
}

/// Which half of the split surface region to remove relative to the cutting tool normal.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum SurfaceTrimSide {
    /// Remove the region on the positive-normal side of the cutting tool.
    #[default]
    Positive,
    /// Remove the region on the negative-normal side of the cutting tool.
    Negative,
}

impl SurfaceTrimSide {
    /// Flip to the opposite removal region; example: `side.opposite()`.
    pub fn opposite(self) -> Self {
        match self {
            Self::Positive => Self::Negative,
            Self::Negative => Self::Positive,
        }
    }
}

impl fmt::Display for SurfaceTrimSide {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Positive => f.write_str("Positive"),
            Self::Negative => f.write_str("Negative"),
        }
    }
}

/// Persisted Surface Trim recipe (Part program version 75, source version 65).
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SurfaceTrimRecipe {
    pub version: u32,
    pub tool: SurfaceTrimTool,
    pub target_body: Vec<u8>,
    #[serde(default)]
    pub remove_side: SurfaceTrimSide,
}

/// Validated authored Surface Trim definition.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SurfaceTrimDefinition {
    recipe: SurfaceTrimRecipe,
    body_source: FeatureId,
    tool_plane_source: Option<FeatureId>,
    tool_body_source: Option<FeatureId>,
    tool_face_source: Option<FeatureId>,
}

/// Resolved cutting tool geometry ready for Martensite evaluation.
#[derive(Clone, Debug, PartialEq)]
pub enum ResolvedSurfaceTrimTool {
    Plane(SketchPlane),
    SurfaceFace { face_key: Vec<u8> },
}

/// Resolved Surface Trim inputs ready for Martensite B-rep evaluation.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedSurfaceTrim {
    pub tool: ResolvedSurfaceTrimTool,
    pub remove_side: SurfaceTrimSide,
}

impl SurfaceTrimDefinition {
    /// Validate and construct a Surface Trim definition from its serialized recipe.
    /// Example: `SurfaceTrimDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: SurfaceTrimRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "surface trim recipe version {}: expected 1",
                recipe.version
            ));
        }
        let (document, body_source) =
            reference_identity(&recipe.target_body, 4, "surface trim target body")?;
        let (tool_plane_source, tool_body_source, tool_face_source) = match &recipe.tool {
            SurfaceTrimTool::OriginPlane(_) => (None, None, None),
            SurfaceTrimTool::WorkPlane(id) => {
                if id.0 == 0 {
                    return Err("surface trim tool work plane: expected nonzero feature id".into());
                }
                (Some(*id), None, None)
            }
            SurfaceTrimTool::SurfaceFace { body, face } => {
                let (tool_body_doc, tool_body_source) =
                    reference_identity(body, 4, "surface trim tool body")?;
                let (tool_face_doc, tool_face_source) =
                    reference_identity(face, 1, "surface trim tool face")?;
                if tool_body_doc != document || tool_face_doc != document {
                    return Err(format!(
                        "surface trim tool document ({tool_body_doc:?}, {tool_face_doc:?}): expected target document {document:?}"
                    ));
                }
                (None, Some(tool_body_source), Some(tool_face_source))
            }
        };
        Ok(Self {
            recipe,
            body_source,
            tool_plane_source,
            tool_body_source,
            tool_face_source,
        })
    }

    /// Copy the serializable recipe; example: `definition.recipe()`.
    pub fn recipe(&self) -> SurfaceTrimRecipe {
        self.recipe.clone()
    }

    /// Selected cutting tool; example: `definition.tool()`.
    pub fn tool(&self) -> &SurfaceTrimTool {
        &self.recipe.tool
    }

    /// Borrow the target surface body reference key; example: `definition.target_body()`.
    pub fn target_body(&self) -> &[u8] {
        &self.recipe.target_body
    }

    /// Producer feature of the target surface body; example: `definition.body_source()`.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }

    /// Producer feature of the cutting tool body when trimming with a surface face.
    pub fn tool_body_source(&self) -> Option<FeatureId> {
        self.tool_body_source
    }

    /// Side of the cutting tool removed by this trim; example: `definition.remove_side()`.
    pub fn remove_side(&self) -> SurfaceTrimSide {
        self.recipe.remove_side
    }

    /// Attribute all read dependencies; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let mut keys = BTreeSet::new();
        keys.insert(FeatureReadKey::Feature(self.body_source));
        if let Some(source) = self.tool_plane_source {
            keys.insert(FeatureReadKey::Feature(source));
        }
        if let Some(source) = self.tool_body_source {
            keys.insert(FeatureReadKey::Feature(source));
        }
        if let Some(source) = self.tool_face_source {
            keys.insert(FeatureReadKey::Feature(source));
        }
        FeatureReadFootprint::new(keys, false)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        feature::{
            AuthoredFeature, FeatureSequence,
            part::{PartFeatureDefinition, PartFeatureProgramRecipe},
        },
        identity::{DocumentGuid, SemanticReferenceName},
    };

    fn sample_ref(doc: DocumentGuid, id: u64, kind: u32, tag: &str) -> Vec<u8> {
        SemanticReferenceName::new(doc, id, kind, "thicken-offset", tag, &[1])
            .unwrap()
            .encode()
            .unwrap()
    }

    #[test]
    fn surface_trim_validates_tools_reads_and_program_v75_source_v65_round_trip() {
        let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let target_body = sample_ref(doc, 2, 4, "body");
        let tool_body = sample_ref(doc, 3, 4, "body");
        let tool_face = sample_ref(doc, 3, 1, "face");

        let recipe = SurfaceTrimRecipe {
            version: 1,
            tool: SurfaceTrimTool::OriginPlane(HoleOriginPlane::Xz),
            target_body: target_body.clone(),
            remove_side: SurfaceTrimSide::Negative,
        };
        let def = SurfaceTrimDefinition::from_recipe(recipe.clone()).unwrap();
        assert_eq!(def.body_source(), FeatureId(2));
        assert_eq!(def.remove_side(), SurfaceTrimSide::Negative);
        assert_eq!(def.remove_side().opposite(), SurfaceTrimSide::Positive);
        assert!(
            def.reads()
                .keys()
                .contains(&FeatureReadKey::Feature(FeatureId(2)))
        );

        let mut seq = FeatureSequence::default();
        seq.append(AuthoredFeature::new(
            FeatureId(2),
            "BasePlane",
            PartFeatureDefinition::Fixed(SketchPlane::xy()),
            FeatureReadFootprint::default(),
        ))
        .unwrap();
        seq.append(AuthoredFeature::new(
            FeatureId(5),
            "Trim1",
            PartFeatureDefinition::SurfaceTrim(Box::new(def.clone())),
            def.reads(),
        ))
        .unwrap();
        let program = seq.part_recipe().unwrap();
        assert_eq!(program.version, 75);
        assert_eq!(program.features[1].definition.version, 65);
        let json = serde_json::to_string(&program).unwrap();
        let decoded: PartFeatureProgramRecipe = serde_json::from_str(&json).unwrap();
        let restored = FeatureSequence::from_part_recipe(&decoded).unwrap();
        assert!(seq.same_work_plane_recipe(&restored));

        let face_tool = SurfaceTrimRecipe {
            version: 1,
            tool: SurfaceTrimTool::SurfaceFace {
                body: tool_body,
                face: tool_face,
            },
            target_body: target_body.clone(),
            remove_side: SurfaceTrimSide::Positive,
        };
        let face_def = SurfaceTrimDefinition::from_recipe(face_tool).unwrap();
        assert_eq!(face_def.tool_body_source(), Some(FeatureId(3)));
        assert!(
            face_def
                .reads()
                .keys()
                .contains(&FeatureReadKey::Feature(FeatureId(3)))
        );

        // Reject unsupported version and zero work plane id.
        assert!(
            SurfaceTrimDefinition::from_recipe(SurfaceTrimRecipe {
                version: 2,
                ..recipe
            })
            .is_err()
        );
        assert!(
            SurfaceTrimDefinition::from_recipe(SurfaceTrimRecipe {
                version: 1,
                tool: SurfaceTrimTool::WorkPlane(FeatureId(0)),
                target_body,
                remove_side: SurfaceTrimSide::Positive,
            })
            .is_err()
        );
    }
}
