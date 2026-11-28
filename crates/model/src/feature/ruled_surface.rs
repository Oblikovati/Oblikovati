// SPDX-License-Identifier: GPL-2.0-only
//! Authored Part Surface Ruled Surface feature recipe, validation, and resolution (#3918).
use super::{
    FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity,
    work_axis::WorkAxisReference,
};
use std::collections::BTreeSet;

/// Ruled surface construction mode matching the 3 task-dialog type buttons.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum RuledSurfaceType {
    /// 1st mode: perpendicular to the face adjacent to each selected edge.
    #[default]
    Normal,
    /// 2nd mode: tangent continuation outward in the face's tangent plane.
    Tangent,
    /// 3rd mode: swept along an explicit direction vector (with optional draft angle).
    Sweep,
}

impl RuledSurfaceType {
    /// Short button label for the Ruled Surface dialog.
    pub const fn label(self) -> &'static str {
        match self {
            Self::Normal => "Normal",
            Self::Tangent => "Tangent",
            Self::Sweep => "Sweep",
        }
    }
}

/// Direction reference selected for Sweep (3rd mode) ruled surfaces.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum RuledSurfaceVector {
    OriginX,
    OriginY,
    OriginZ,
    WorkAxis(FeatureId),
    Edge { body: Vec<u8>, edge: Vec<u8> },
}

impl RuledSurfaceVector {
    /// Convert a work-axis reference (Origin or feature axis) into a ruled-surface vector.
    /// Example: `RuledSurfaceVector::from_work_axis_reference(WorkAxisReference::OriginZ)`.
    pub const fn from_work_axis_reference(reference: WorkAxisReference) -> Self {
        match reference {
            WorkAxisReference::OriginX => Self::OriginX,
            WorkAxisReference::OriginY => Self::OriginY,
            WorkAxisReference::OriginZ => Self::OriginZ,
            WorkAxisReference::Feature(id) => Self::WorkAxis(id),
        }
    }

    /// Return the fixed unit vector when this reference is an Origin axis.
    pub const fn origin_direction(&self) -> Option<[f64; 3]> {
        match self {
            Self::OriginX => Some([1.0, 0.0, 0.0]),
            Self::OriginY => Some([0.0, 1.0, 0.0]),
            Self::OriginZ => Some([0.0, 0.0, 1.0]),
            Self::WorkAxis(_) | Self::Edge { .. } => None,
        }
    }

    /// Human-readable label shown in the Vector selector pill.
    pub const fn label(&self) -> &'static str {
        match self {
            Self::OriginX => "X Axis",
            Self::OriginY => "Y Axis",
            Self::OriginZ => "Z Axis",
            Self::WorkAxis(_) => "Work Axis",
            Self::Edge { .. } => "Edge",
        }
    }
}

fn default_true() -> bool {
    true
}

/// Persisted Part Surface Ruled Surface recipe (Part program version 76, source version 66).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct RuledSurfaceRecipe {
    pub version: u32,
    pub target_body: Vec<u8>,
    pub edges: Vec<Vec<u8>>,
    #[serde(default)]
    pub ruled_type: RuledSurfaceType,
    /// Ruled extension distance in model units (cm); must be strictly positive.
    pub distance: f64,
    /// Draft/tilt angle in degrees within [-180, 180].
    #[serde(default)]
    pub angle_deg: f64,
    #[serde(default)]
    pub flip: bool,
    #[serde(default = "default_true")]
    pub automatic_edge_chain: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub vector: Option<RuledSurfaceVector>,
}

/// Validated authored Ruled Surface definition.
#[derive(Clone, Debug, PartialEq)]
pub struct RuledSurfaceDefinition {
    recipe: RuledSurfaceRecipe,
    body_source: FeatureId,
    vector_source: Option<FeatureId>,
}

/// Resolved Ruled Surface inputs ready for Martensite evaluation.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedRuledSurface {
    pub edges: Vec<Vec<u8>>,
    pub ruled_type: RuledSurfaceType,
    /// Distance in model units (cm).
    pub distance: f64,
    /// Tilt/draft angle in degrees.
    pub angle_deg: f64,
    pub flip: bool,
    pub vector_direction: Option<[f64; 3]>,
    pub vector_edge: Option<Vec<u8>>,
}

impl RuledSurfaceDefinition {
    /// Validate and construct a Ruled Surface definition from its serialized recipe.
    /// Example: `RuledSurfaceDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: RuledSurfaceRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "ruled surface recipe version {}: expected 1",
                recipe.version
            ));
        }
        let (document, body_source) =
            reference_identity(&recipe.target_body, 4, "ruled surface body")?;
        if recipe.edges.is_empty() {
            return Err("ruled surface edges []: expected at least one selected edge".into());
        }
        let mut seen = BTreeSet::new();
        for edge in &recipe.edges {
            let (edge_doc, _) = reference_identity(edge, 2, "ruled surface edge")?;
            if edge_doc != document {
                return Err(format!(
                    "ruled surface edge document {edge_doc:?}: expected body document {document:?}"
                ));
            }
            if !seen.insert(edge.as_slice()) {
                return Err(format!(
                    "ruled surface edge {edge:?}: expected distinct selected edges"
                ));
            }
        }
        if !recipe.distance.is_finite() || recipe.distance <= 0.0 {
            return Err(format!(
                "ruled surface distance {}: expected a finite positive distance",
                recipe.distance
            ));
        }
        if !recipe.angle_deg.is_finite() || !(-180.0..=180.0).contains(&recipe.angle_deg) {
            return Err(format!(
                "ruled surface angle {}: expected a finite angle in [-180, 180] degrees",
                recipe.angle_deg
            ));
        }
        let vector_source = match &recipe.vector {
            None if recipe.ruled_type == RuledSurfaceType::Sweep => {
                return Err(
                    "ruled surface vector None: expected a direction vector for Sweep mode".into(),
                );
            }
            Some(RuledSurfaceVector::WorkAxis(id)) => Some(*id),
            Some(RuledSurfaceVector::Edge { body, edge }) => {
                let (vec_doc, vec_source) =
                    reference_identity(body, 4, "ruled surface vector body")?;
                let (edge_doc, _) = reference_identity(edge, 2, "ruled surface vector edge")?;
                if vec_doc != document || edge_doc != document {
                    return Err(format!(
                        "ruled surface vector document ({vec_doc:?}, {edge_doc:?}): expected {document:?}"
                    ));
                }
                Some(vec_source)
            }
            _ => None,
        };
        Ok(Self {
            recipe,
            body_source,
            vector_source,
        })
    }

    /// Copy the serializable recipe; example: `definition.recipe()`.
    pub fn recipe(&self) -> RuledSurfaceRecipe {
        self.recipe.clone()
    }

    /// Borrow the authored target body key; example: `definition.target_body()`.
    pub fn target_body(&self) -> &[u8] {
        &self.recipe.target_body
    }

    /// Producer feature of the target body; example: `definition.body_source()`.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }

    /// Selected edges in authored order; example: `definition.edges()`.
    pub fn edges(&self) -> &[Vec<u8>] {
        &self.recipe.edges
    }

    /// Selected ruled surface mode (`Normal`, `Tangent`, or `Sweep`).
    pub fn ruled_type(&self) -> RuledSurfaceType {
        self.recipe.ruled_type
    }

    /// Extension distance in model units (cm).
    pub fn distance(&self) -> f64 {
        self.recipe.distance
    }

    /// Draft/tilt angle in degrees.
    pub fn angle_deg(&self) -> f64 {
        self.recipe.angle_deg
    }

    /// Whether the ruled surface direction is reversed.
    pub fn flip(&self) -> bool {
        self.recipe.flip
    }

    /// Whether Automatic Edge Chain was enabled when authored.
    pub fn automatic_edge_chain(&self) -> bool {
        self.recipe.automatic_edge_chain
    }

    /// Selected direction vector reference (required for Sweep mode).
    pub fn vector(&self) -> Option<&RuledSurfaceVector> {
        self.recipe.vector.as_ref()
    }

    /// Attribute all read dependencies; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let mut keys = BTreeSet::new();
        keys.insert(FeatureReadKey::Feature(self.body_source));
        if let Some(source) = self.vector_source {
            keys.insert(FeatureReadKey::Feature(source));
        }
        FeatureReadFootprint::new(keys, false)
    }

    /// Resolve authored fields for kernel evaluation with a resolved sweep vector.
    /// Example: `definition.resolve_with_direction(Some([0.0, 0.0, 1.0]), None)`.
    pub fn resolve_with_direction(
        &self,
        vector_direction: Option<[f64; 3]>,
        vector_edge: Option<Vec<u8>>,
    ) -> ResolvedRuledSurface {
        ResolvedRuledSurface {
            edges: self.recipe.edges.clone(),
            ruled_type: self.recipe.ruled_type,
            distance: self.recipe.distance,
            angle_deg: self.recipe.angle_deg,
            flip: self.recipe.flip,
            vector_direction,
            vector_edge,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::identity::{DocumentGuid, SemanticReferenceName};

    fn key(doc: DocumentGuid, kind: u32, idx: u64) -> Vec<u8> {
        SemanticReferenceName::new(doc, 1, kind, "ruled-test", "e", &[idx])
            .unwrap()
            .encode()
            .unwrap()
    }

    #[test]
    fn ruled_surface_validates_modes_distance_angle_and_vector() {
        let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let body = key(doc, 4, 1);
        let edge1 = key(doc, 2, 1);
        let base = RuledSurfaceRecipe {
            version: 1,
            target_body: body,
            edges: vec![edge1],
            ruled_type: RuledSurfaceType::Normal,
            distance: 1.0,
            angle_deg: 0.0,
            flip: false,
            automatic_edge_chain: true,
            vector: None,
        };
        assert!(RuledSurfaceDefinition::from_recipe(base.clone()).is_ok());
        assert!(
            RuledSurfaceDefinition::from_recipe(RuledSurfaceRecipe {
                distance: 0.0,
                ..base.clone()
            })
            .is_err()
        );
        assert!(
            RuledSurfaceDefinition::from_recipe(RuledSurfaceRecipe {
                distance: -0.1,
                ..base.clone()
            })
            .is_err()
        );
        assert!(
            RuledSurfaceDefinition::from_recipe(RuledSurfaceRecipe {
                ruled_type: RuledSurfaceType::Sweep,
                vector: None,
                ..base.clone()
            })
            .is_err()
        );
        let sweep = RuledSurfaceDefinition::from_recipe(RuledSurfaceRecipe {
            ruled_type: RuledSurfaceType::Sweep,
            vector: Some(RuledSurfaceVector::OriginZ),
            distance: 2.0,
            angle_deg: 180.0,
            ..base
        })
        .unwrap();
        assert_eq!(sweep.distance(), 2.0);
        assert_eq!(sweep.angle_deg(), 180.0);
        assert_eq!(
            sweep
                .vector()
                .and_then(RuledSurfaceVector::origin_direction),
            Some([0.0, 0.0, 1.0])
        );
    }
}
