// SPDX-License-Identifier: GPL-2.0-only
//! Authored Part Surface Extend feature recipe, validation, and resolution (#3918).
use super::{
    FeatureId, FeatureReadFootprint, FeatureReadKey,
    fillet::reference_identity,
    work_plane::{WorkPlaneDefinition, WorkPlaneError},
};
use crate::sketch::SketchPlane;
use std::{collections::BTreeSet, fmt};

/// Surface extension method (`More` / `Type` in the Extend Surface dialog).
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum ExtendSurfaceMethod {
    /// Create a contiguous tangent extension surface stitched to the selected boundary edge(s).
    #[default]
    Extend,
    /// Stretch the existing surface face boundary outward without adding an internal seam.
    Stretch,
}

impl ExtendSurfaceMethod {
    /// Display caption in the Extend Surface Properties panel.
    pub const fn label(self) -> &'static str {
        match self {
            Self::Extend => "Extend",
            Self::Stretch => "Stretch",
        }
    }
}

impl fmt::Display for ExtendSurfaceMethod {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(self.label())
    }
}

/// Origin datum plane used as a `To` termination target for Extend Surface.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum ExtendSurfaceOriginPlane {
    Xy,
    Xz,
    Yz,
}

impl ExtendSurfaceOriginPlane {
    /// Supporting plane in model coordinates.
    pub fn sketch_plane(self) -> SketchPlane {
        match self {
            Self::Xy => SketchPlane::xy(),
            Self::Xz => SketchPlane::xz(),
            Self::Yz => SketchPlane::yz(),
        }
    }
}

/// Termination target when Extend Surface uses the `To` extent.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum ExtendSurfaceTermination {
    Face { body: Vec<u8>, face: Vec<u8> },
    WorkPlane(FeatureId),
    OriginPlane(ExtendSurfaceOriginPlane),
}

/// Extent mode for Extend Surface (`Distance` or `To`).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum ExtendSurfaceExtent {
    /// Extend by a positive distance in model units (cm).
    Distance(f64),
    /// Extend up to a selected planar face or work plane.
    To(ExtendSurfaceTermination),
}

impl ExtendSurfaceExtent {
    /// Distance in cm when `Distance` extent is active.
    pub const fn distance_cm(&self) -> Option<f64> {
        match self {
            Self::Distance(cm) => Some(*cm),
            Self::To(_) => None,
        }
    }
}

/// Persisted Part Surface Extend recipe (Part program version 71, source version 61).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ExtendSurfaceRecipe {
    pub version: u32,
    pub target_body: Vec<u8>,
    pub edges: Vec<Vec<u8>>,
    #[serde(default = "default_edge_chain")]
    pub edge_chain: bool,
    pub extent: ExtendSurfaceExtent,
    #[serde(default)]
    pub method: ExtendSurfaceMethod,
}

const fn default_edge_chain() -> bool {
    true
}

/// Validated authored Extend Surface definition.
#[derive(Clone, Debug, PartialEq)]
pub struct ExtendSurfaceDefinition {
    recipe: ExtendSurfaceRecipe,
    body_source: FeatureId,
    termination_source: Option<FeatureId>,
}

/// Resolved extent ready for Martensite evaluation.
#[derive(Clone, Debug, PartialEq)]
pub enum ResolvedExtendSurfaceExtent {
    Distance(f64),
    ToPlane(SketchPlane),
}

/// Resolved Extend Surface inputs ready for Martensite evaluation.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedExtendSurface {
    pub edges: Vec<Vec<u8>>,
    pub edge_chain: bool,
    pub extent: ResolvedExtendSurfaceExtent,
    pub method: ExtendSurfaceMethod,
}

impl ExtendSurfaceDefinition {
    /// Validate and construct an Extend Surface definition from its serialized recipe.
    /// Example: `ExtendSurfaceDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: ExtendSurfaceRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "extend surface recipe version {}: expected 1",
                recipe.version
            ));
        }
        let (document, body_source) =
            reference_identity(&recipe.target_body, 4, "extend surface body")?;
        if recipe.edges.is_empty() {
            return Err(
                "extend surface edges []: expected at least one selected boundary edge".into(),
            );
        }
        let mut seen = BTreeSet::new();
        for edge in &recipe.edges {
            let (edge_doc, _) = reference_identity(edge, 2, "extend surface edge")?;
            if edge_doc != document {
                return Err(format!(
                    "extend surface edge document {edge_doc:?}: expected body document {document:?}"
                ));
            }
            if !seen.insert(edge.as_slice()) {
                return Err(format!(
                    "extend surface edge {edge:?}: expected distinct selected edges"
                ));
            }
        }
        let termination_source = match &recipe.extent {
            ExtendSurfaceExtent::Distance(distance) => {
                if !distance.is_finite() || *distance <= 0.0 {
                    return Err(format!(
                        "extend surface distance {distance}: expected a strictly positive finite distance"
                    ));
                }
                None
            }
            ExtendSurfaceExtent::To(ExtendSurfaceTermination::Face { body, face }) => {
                let (term_body_doc, term_source) =
                    reference_identity(body, 4, "extend surface termination body")?;
                let (term_face_doc, _) =
                    reference_identity(face, 1, "extend surface termination face")?;
                if term_body_doc != document || term_face_doc != document {
                    return Err(format!(
                        "extend surface termination document {term_face_doc:?}: expected body document {document:?}"
                    ));
                }
                Some(term_source)
            }
            ExtendSurfaceExtent::To(ExtendSurfaceTermination::WorkPlane(plane_id)) => {
                Some(*plane_id)
            }
            ExtendSurfaceExtent::To(ExtendSurfaceTermination::OriginPlane(_)) => None,
        };
        Ok(Self {
            recipe,
            body_source,
            termination_source,
        })
    }

    /// Copy the serializable recipe; example: `definition.recipe()`.
    pub fn recipe(&self) -> ExtendSurfaceRecipe {
        self.recipe.clone()
    }

    /// Borrow the authored target surface body key; example: `definition.target_body()`.
    pub fn target_body(&self) -> &[u8] {
        &self.recipe.target_body
    }

    /// Producer feature of the target surface body; example: `definition.body_source()`.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }

    /// Selected boundary edges in authored order; example: `definition.edges()`.
    pub fn edges(&self) -> &[Vec<u8>] {
        &self.recipe.edges
    }

    /// Whether Edge Chain was enabled when authored; example: `definition.edge_chain()`.
    pub fn edge_chain(&self) -> bool {
        self.recipe.edge_chain
    }

    /// Authored extent (`Distance` or `To`); example: `definition.extent()`.
    pub fn extent(&self) -> &ExtendSurfaceExtent {
        &self.recipe.extent
    }

    /// Authored extension method (`Extend` or `Stretch`); example: `definition.method()`.
    pub fn method(&self) -> ExtendSurfaceMethod {
        self.recipe.method
    }

    /// Attribute all read dependencies; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let mut keys = BTreeSet::new();
        keys.insert(FeatureReadKey::Feature(self.body_source));
        if let Some(source) = self.termination_source {
            keys.insert(FeatureReadKey::Feature(source));
        }
        FeatureReadFootprint::new(keys, false)
    }
}

/// Restore a serialized Extend Surface recipe into the shared Part feature enum.
pub(crate) fn restore_definition(
    recipe: ExtendSurfaceRecipe,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    ExtendSurfaceDefinition::from_recipe(recipe)
        .map(|definition| WorkPlaneDefinition::ExtendSurface(Box::new(definition)))
        .map_err(WorkPlaneError::BodyReference)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::feature::part::PartFeatureDefinition;
    use crate::feature::work_plane::{WorkPlaneDefinition, WorkPlaneRecipe};
    use crate::identity::{DocumentGuid, SemanticReferenceName};

    fn key(doc: DocumentGuid, kind: u32, idx: u64) -> Vec<u8> {
        SemanticReferenceName::new(doc, 1, kind, "thicken-offset", "e", &[idx])
            .unwrap()
            .encode()
            .unwrap()
    }

    #[test]
    fn extend_surface_validates_distance_edges_and_recipe_versions() {
        let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let body = key(doc, 4, 1);
        let edge1 = key(doc, 2, 1);
        let edge2 = key(doc, 2, 2);
        let base = ExtendSurfaceRecipe {
            version: 1,
            target_body: body.clone(),
            edges: vec![edge1.clone()],
            edge_chain: false,
            extent: ExtendSurfaceExtent::Distance(0.5),
            method: ExtendSurfaceMethod::Extend,
        };
        let def = ExtendSurfaceDefinition::from_recipe(base.clone()).unwrap();
        assert_eq!(def.method(), ExtendSurfaceMethod::Extend);
        assert_eq!(def.extent().distance_cm(), Some(0.5));
        assert!(!def.edge_chain());

        // Reject zero/negative distance and empty/duplicate edges.
        assert!(
            ExtendSurfaceDefinition::from_recipe(ExtendSurfaceRecipe {
                extent: ExtendSurfaceExtent::Distance(0.0),
                ..base.clone()
            })
            .is_err()
        );
        assert!(
            ExtendSurfaceDefinition::from_recipe(ExtendSurfaceRecipe {
                extent: ExtendSurfaceExtent::Distance(-0.5),
                ..base.clone()
            })
            .is_err()
        );
        assert!(
            ExtendSurfaceDefinition::from_recipe(ExtendSurfaceRecipe {
                edges: Vec::new(),
                ..base.clone()
            })
            .is_err()
        );
        assert!(
            ExtendSurfaceDefinition::from_recipe(ExtendSurfaceRecipe {
                edges: vec![edge1.clone(), edge1.clone()],
                ..base.clone()
            })
            .is_err()
        );

        // Stretch + Edge Chain + To extent round-trip through WorkPlaneRecipe v61 / program v71.
        let stretch = ExtendSurfaceDefinition::from_recipe(ExtendSurfaceRecipe {
            edges: vec![edge1, edge2],
            edge_chain: true,
            extent: ExtendSurfaceExtent::Distance(1.0),
            method: ExtendSurfaceMethod::Stretch,
            ..base
        })
        .unwrap();
        let wrapped = PartFeatureDefinition::ExtendSurface(Box::new(stretch));
        assert_eq!(wrapped.kind(), "extend-surface");
        let recipe: WorkPlaneRecipe = wrapped.recipe();
        assert_eq!(recipe.version, 61);
        let restored = WorkPlaneDefinition::from_recipe(recipe).unwrap();
        assert_eq!(restored, wrapped);

        // To extent with OriginPlane validates cleanly.
        let to_origin = ExtendSurfaceDefinition::from_recipe(ExtendSurfaceRecipe {
            version: 1,
            target_body: body,
            edges: vec![key(doc, 2, 3)],
            edge_chain: false,
            extent: ExtendSurfaceExtent::To(ExtendSurfaceTermination::OriginPlane(
                ExtendSurfaceOriginPlane::Yz,
            )),
            method: ExtendSurfaceMethod::Extend,
        })
        .unwrap();
        assert_eq!(to_origin.extent().distance_cm(), None);
    }
}
