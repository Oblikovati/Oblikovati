// SPDX-License-Identifier: GPL-2.0-only
//! Authored Part Modify Split feature recipe, validation, and resolution (#3918).
use super::{
    FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity,
    hole::HoleOriginPlane,
};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{SketchEntityId, SketchPlane},
};
use std::{collections::BTreeSet, fmt};

/// Mode of a Part Modify Split feature (split faces on one solid or split a solid body).
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum SplitMode {
    #[default]
    Faces,
    Solid,
}

impl fmt::Display for SplitMode {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Faces => f.write_str("Split Face"),
            Self::Solid => f.write_str("Split Solid"),
        }
    }
}

/// Which side(s) of a split solid body to retain in `SplitMode::Solid`.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum SplitKeepSide {
    /// Keep both split solid bodies (first Keep Side option).
    #[default]
    Both,
    /// Keep only the positive-normal half of the split solid (second Keep Side option).
    SideA,
    /// Keep only the negative-normal half of the split solid (third Keep Side option).
    SideB,
}

impl fmt::Display for SplitKeepSide {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Both => f.write_str("Keep Both"),
            Self::SideA => f.write_str("Keep Side A"),
            Self::SideB => f.write_str("Keep Side B"),
        }
    }
}

/// Splitting tool reference (open sketch curve, Origin plane, or Work Plane).
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum SplitTool {
    SketchCurve {
        sketch: DocumentSketchId,
        entity: SketchEntityId,
    },
    OriginPlane(HoleOriginPlane),
    WorkPlane(FeatureId),
}

/// Target geometry for a Part Modify Split feature.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum SplitTarget {
    AllFaces { body: Vec<u8> },
    Faces { body: Vec<u8>, faces: Vec<Vec<u8>> },
    Solid { body: Vec<u8> },
}

impl SplitTarget {
    /// Borrow the target body reference key; example: `target.body()`.
    pub fn body(&self) -> &[u8] {
        match self {
            Self::AllFaces { body } | Self::Faces { body, .. } | Self::Solid { body } => body,
        }
    }
}

/// Persisted Part Modify Split recipe (Part program version 61, source version 51).
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SplitRecipe {
    pub version: u32,
    #[serde(default)]
    pub mode: SplitMode,
    pub tool: SplitTool,
    pub target: SplitTarget,
    #[serde(default)]
    pub keep_side: SplitKeepSide,
}

/// Validated authored Split definition.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SplitDefinition {
    recipe: SplitRecipe,
    body_source: FeatureId,
    face_sources: Vec<FeatureId>,
    plane_source: Option<FeatureId>,
}

/// Resolved 3D sketch curve geometry ready for Martensite evaluation.
#[derive(Clone, Debug, PartialEq)]
pub enum ResolvedSplitCurve {
    Line {
        start: [f64; 3],
        end: [f64; 3],
        plane_normal: [f64; 3],
    },
    Arc {
        center: [f64; 3],
        start: [f64; 3],
        end: [f64; 3],
        counter_clockwise: bool,
        plane: SketchPlane,
    },
    Circle {
        center: [f64; 3],
        radius: f64,
        plane: SketchPlane,
    },
}

/// Resolved splitting tool geometry in model coordinates.
#[derive(Clone, Debug, PartialEq)]
pub enum ResolvedSplitTool {
    Curve(ResolvedSplitCurve),
    Plane(SketchPlane),
}

/// Resolved target face set or solid mode for kernel evaluation.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum ResolvedSplitTarget {
    AllFaces,
    Faces(Vec<Vec<u8>>),
    Solid,
}

/// Resolved Split inputs ready for Martensite B-rep evaluation.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedSplit {
    pub mode: SplitMode,
    pub tool: ResolvedSplitTool,
    pub target: ResolvedSplitTarget,
    pub keep_side: SplitKeepSide,
}

impl SplitDefinition {
    /// Validate and construct a Split definition from its serialized recipe.
    /// Example: `SplitDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: SplitRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "split recipe version {}: expected 1",
                recipe.version
            ));
        }
        let (document, body_source) = reference_identity(recipe.target.body(), 4, "split body")?;
        let mut face_sources = Vec::new();
        match (&recipe.mode, &recipe.target) {
            (SplitMode::Faces, SplitTarget::AllFaces { .. }) => {}
            (SplitMode::Faces, SplitTarget::Faces { faces, .. }) => {
                if faces.is_empty() {
                    return Err("split faces []: expected at least one face in Faces mode".into());
                }
                let mut seen = BTreeSet::new();
                for face in faces {
                    if !seen.insert(face.as_slice()) {
                        return Err("split faces: expected unique face references".into());
                    }
                    let (face_doc, face_source) = reference_identity(face, 1, "split face")?;
                    if face_doc != document {
                        return Err(format!(
                            "split face document {face_doc:?}: expected body document {document:?}"
                        ));
                    }
                    face_sources.push(face_source);
                }
            }
            (SplitMode::Solid, SplitTarget::Solid { .. }) => {}
            (mode, target) => {
                return Err(format!(
                    "split mode {mode:?}, target {target:?}: expected matching target kind"
                ));
            }
        }
        let plane_source = match &recipe.tool {
            SplitTool::WorkPlane(id) => {
                if id.0 == 0 {
                    return Err("split tool work plane: expected nonzero feature id".into());
                }
                Some(*id)
            }
            SplitTool::SketchCurve { sketch, entity } => {
                if sketch.0 == 0 || entity.0 == 0 {
                    return Err(
                        "split tool sketch curve: expected nonzero sketch and entity ids".into(),
                    );
                }
                None
            }
            SplitTool::OriginPlane(_) => None,
        };
        Ok(Self {
            recipe,
            body_source,
            face_sources,
            plane_source,
        })
    }

    /// Copy the serializable recipe; example: `definition.recipe()`.
    pub fn recipe(&self) -> SplitRecipe {
        self.recipe.clone()
    }

    /// Selected Split mode; example: `definition.mode()`.
    pub fn mode(&self) -> SplitMode {
        self.recipe.mode
    }

    /// Selected Split tool; example: `definition.tool()`.
    pub fn tool(&self) -> &SplitTool {
        &self.recipe.tool
    }

    /// Selected Split target; example: `definition.target()`.
    pub fn target(&self) -> &SplitTarget {
        &self.recipe.target
    }

    /// Borrow the target body key; example: `definition.body()`.
    pub fn body(&self) -> &[u8] {
        self.recipe.target.body()
    }

    /// Producer feature of the target body; example: `definition.body_source()`.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }

    /// Keep side option for Solid mode; example: `definition.keep_side()`.
    pub fn keep_side(&self) -> SplitKeepSide {
        self.recipe.keep_side
    }

    /// Attribute all read dependencies; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let mut keys = BTreeSet::new();
        keys.insert(FeatureReadKey::Feature(self.body_source));
        for source in &self.face_sources {
            keys.insert(FeatureReadKey::Feature(*source));
        }
        if let Some(source) = self.plane_source {
            keys.insert(FeatureReadKey::Feature(source));
        }
        FeatureReadFootprint::new(keys, false)
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
    fn split_definition_validates_modes_targets_and_reads() {
        let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let body = sample_ref(doc, 1, 4, "body");
        let face = sample_ref(doc, 1, 1, "top");

        let all_faces = SplitRecipe {
            version: 1,
            mode: SplitMode::Faces,
            tool: SplitTool::SketchCurve {
                sketch: DocumentSketchId(2),
                entity: SketchEntityId(3),
            },
            target: SplitTarget::AllFaces { body: body.clone() },
            keep_side: SplitKeepSide::Both,
        };
        let def = SplitDefinition::from_recipe(all_faces).unwrap();
        assert_eq!(def.body_source(), FeatureId(1));
        assert!(
            def.reads()
                .keys()
                .contains(&FeatureReadKey::Feature(FeatureId(1)))
        );

        let one_face = SplitRecipe {
            version: 1,
            mode: SplitMode::Faces,
            tool: SplitTool::SketchCurve {
                sketch: DocumentSketchId(2),
                entity: SketchEntityId(3),
            },
            target: SplitTarget::Faces {
                body: body.clone(),
                faces: vec![face],
            },
            keep_side: SplitKeepSide::Both,
        };
        assert!(SplitDefinition::from_recipe(one_face).is_ok());

        let solid = SplitRecipe {
            version: 1,
            mode: SplitMode::Solid,
            tool: SplitTool::WorkPlane(FeatureId(4)),
            target: SplitTarget::Solid { body: body.clone() },
            keep_side: SplitKeepSide::SideA,
        };
        let solid_def = SplitDefinition::from_recipe(solid).unwrap();
        assert!(
            solid_def
                .reads()
                .keys()
                .contains(&FeatureReadKey::Feature(FeatureId(4)))
        );

        // Reject empty faces list
        let empty = SplitRecipe {
            version: 1,
            mode: SplitMode::Faces,
            tool: SplitTool::OriginPlane(HoleOriginPlane::Xy),
            target: SplitTarget::Faces {
                body: body.clone(),
                faces: vec![],
            },
            keep_side: SplitKeepSide::Both,
        };
        assert!(SplitDefinition::from_recipe(empty).is_err());

        // Reject mode/target mismatch
        let mismatch = SplitRecipe {
            version: 1,
            mode: SplitMode::Solid,
            tool: SplitTool::OriginPlane(HoleOriginPlane::Xy),
            target: SplitTarget::AllFaces { body },
            keep_side: SplitKeepSide::Both,
        };
        assert!(SplitDefinition::from_recipe(mismatch).is_err());
    }
}
