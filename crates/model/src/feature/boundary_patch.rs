// SPDX-License-Identifier: GPL-2.0-only
//! Authored Surface Boundary Patch feature recipe, validation, and resolution (#3918).
use super::{FeatureReadFootprint, FeatureReadKey};
use crate::{
    document_sketch::DocumentSketchId,
    feature::work_geometry::WorkGeometryEvaluation,
    history::{DocumentSnapshot, ResolvedDocumentSketch},
    sketch::SketchEntityId,
};
use std::collections::{BTreeMap, BTreeSet};

/// Continuity condition applied to one boundary edge set.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum BoundaryCondition {
    /// G0 positional contact (1st condition icon).
    #[default]
    Free,
    /// G1 tangency condition (2nd condition icon).
    Tangent,
    /// G2 curvature smoothness condition (3rd condition icon).
    Smooth,
}

impl BoundaryCondition {
    /// Human-readable label shown in Properties tooltips and dropdowns.
    pub const fn label(self) -> &'static str {
        match self {
            Self::Free => "Contact (G0)",
            Self::Tangent => "Tangent (G1)",
            Self::Smooth => "Smooth (G2)",
        }
    }

    /// Short caption for compact UI buttons.
    pub const fn short_label(self) -> &'static str {
        match self {
            Self::Free => "G0",
            Self::Tangent => "G1",
            Self::Smooth => "G2",
        }
    }

    /// 0-based icon slot matching the three Boundary Patch condition icons.
    pub const fn index(self) -> usize {
        match self {
            Self::Free => 0,
            Self::Tangent => 1,
            Self::Smooth => 2,
        }
    }

    /// Map a 0-based icon slot to its condition; example: `BoundaryCondition::from_index(1)`.
    pub const fn from_index(index: usize) -> Option<Self> {
        match index {
            0 => Some(Self::Free),
            1 => Some(Self::Tangent),
            2 => Some(Self::Smooth),
            _ => None,
        }
    }

    /// Whether a numeric weight factor applies to this condition.
    pub const fn supports_weight(self) -> bool {
        matches!(self, Self::Tangent | Self::Smooth)
    }
}

/// One curve referenced by a Boundary Patch edge set or guide rail.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum BoundaryCurveRef {
    SketchCurve {
        sketch: DocumentSketchId,
        entity: SketchEntityId,
    },
}

impl BoundaryCurveRef {
    /// Owning sketch identifier for sketch curve references.
    pub const fn sketch(self) -> DocumentSketchId {
        match self {
            Self::SketchCurve { sketch, .. } => sketch,
        }
    }

    /// Curve entity identifier inside the owning sketch.
    pub const fn entity(self) -> SketchEntityId {
        match self {
            Self::SketchCurve { entity, .. } => entity,
        }
    }

    /// Deterministic ordering key for duplicate detection.
    pub const fn sort_key(self) -> (u64, u64) {
        (self.sketch().0, self.entity().0)
    }
}

/// One grouped boundary edge selection (`Edges1`, `Edges2`, ...) with continuity condition and weight.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct BoundaryEdgeSet {
    pub edges: Vec<BoundaryCurveRef>,
    #[serde(default)]
    pub condition: BoundaryCondition,
    #[serde(default = "default_weight")]
    pub weight: f64,
}

const fn default_weight() -> f64 {
    1.0
}

/// Persisted Surface Boundary Patch recipe (Part program version 70, source version 60).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct BoundaryPatchRecipe {
    pub version: u32,
    #[serde(default = "default_auto_chain")]
    pub automatic_edge_chain: bool,
    pub edge_sets: Vec<BoundaryEdgeSet>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub guide_rails: Vec<BoundaryCurveRef>,
}

const fn default_auto_chain() -> bool {
    true
}

/// Validated authored Surface Boundary Patch definition.
#[derive(Clone, Debug, PartialEq)]
pub struct BoundaryPatchDefinition {
    recipe: BoundaryPatchRecipe,
}

/// Resolved sketches and boundary curve sets ready for Martensite evaluation.
pub struct ResolvedBoundaryPatch<'a> {
    pub sketches: BTreeMap<u64, ResolvedDocumentSketch<'a>>,
    pub edge_sets: Vec<BoundaryEdgeSet>,
    pub guide_rails: Vec<BoundaryCurveRef>,
    pub automatic_edge_chain: bool,
}

impl BoundaryPatchDefinition {
    /// Validate and construct a Boundary Patch definition from its serialized recipe.
    /// Example: `BoundaryPatchDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: BoundaryPatchRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "boundary patch recipe version {}: expected 1",
                recipe.version
            ));
        }
        if recipe.edge_sets.is_empty() {
            return Err("boundary patch edge_sets []: expected at least one boundary set".into());
        }
        let mut seen = BTreeSet::new();
        for (set_index, set) in recipe.edge_sets.iter().enumerate() {
            if set.edges.is_empty() {
                return Err(format!(
                    "boundary patch edge_sets[{set_index}]: expected at least one edge"
                ));
            }
            if !set.weight.is_finite() || set.weight < 0.0 {
                return Err(format!(
                    "boundary patch edge_sets[{set_index}] weight {}: expected a finite non-negative weight",
                    set.weight
                ));
            }
            for &curve in &set.edges {
                if !seen.insert(curve.sort_key()) {
                    return Err(format!(
                        "boundary patch curve {curve:?}: expected distinct boundary curves"
                    ));
                }
            }
        }
        for &rail in &recipe.guide_rails {
            if !seen.insert(rail.sort_key()) {
                return Err(format!(
                    "boundary patch guide rail {rail:?}: expected a curve distinct from boundary edges"
                ));
            }
        }
        Ok(Self { recipe })
    }

    /// Copy the serializable recipe; example: `definition.recipe()`.
    pub fn recipe(&self) -> BoundaryPatchRecipe {
        self.recipe.clone()
    }

    /// Whether Automatic Edge Chain was active when authored.
    pub fn automatic_edge_chain(&self) -> bool {
        self.recipe.automatic_edge_chain
    }

    /// Authored boundary edge sets (`Edges1`, `Edges2`, ...).
    pub fn edge_sets(&self) -> &[BoundaryEdgeSet] {
        &self.recipe.edge_sets
    }

    /// Authored guide rail curves.
    pub fn guide_rails(&self) -> &[BoundaryCurveRef] {
        &self.recipe.guide_rails
    }

    /// Distinct sketches referenced by boundary edges or guide rails.
    pub fn referenced_sketches(&self) -> Vec<DocumentSketchId> {
        let ids: BTreeSet<u64> = self
            .recipe
            .edge_sets
            .iter()
            .flat_map(|set| set.edges.iter().map(|curve| curve.sketch().0))
            .chain(self.recipe.guide_rails.iter().map(|rail| rail.sketch().0))
            .collect();
        ids.into_iter().map(DocumentSketchId).collect()
    }

    /// Attribute all sketch read dependencies; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let keys: Vec<FeatureReadKey> = self
            .referenced_sketches()
            .into_iter()
            .map(|sketch| FeatureReadKey::Sketch(sketch.0))
            .collect();
        FeatureReadFootprint::new(keys, false)
    }

    /// Resolve every referenced sketch plane and converged geometry during ordered part replay.
    /// Example: `definition.resolve_for_replay(snapshot, &datums)?`.
    pub(in crate::feature) fn resolve_for_replay<'a>(
        &self,
        snapshot: &'a DocumentSnapshot,
        datums: &WorkGeometryEvaluation,
    ) -> Result<ResolvedBoundaryPatch<'a>, String> {
        let mut sketches = BTreeMap::new();
        for sketch_id in self.referenced_sketches() {
            let resolved = snapshot
                .require_converged_sketch_with_datums(sketch_id, datums)
                .map_err(|cause| cause.to_string())?;
            sketches.insert(sketch_id.0, resolved);
        }
        Ok(ResolvedBoundaryPatch {
            sketches,
            edge_sets: self.recipe.edge_sets.clone(),
            guide_rails: self.recipe.guide_rails.clone(),
            automatic_edge_chain: self.recipe.automatic_edge_chain,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn curve(sketch: u64, entity: u64) -> BoundaryCurveRef {
        BoundaryCurveRef::SketchCurve {
            sketch: DocumentSketchId(sketch),
            entity: SketchEntityId(entity),
        }
    }

    #[test]
    fn validates_recipe_and_tracks_sketch_reads() {
        let recipe = BoundaryPatchRecipe {
            version: 1,
            automatic_edge_chain: true,
            edge_sets: vec![BoundaryEdgeSet {
                edges: vec![curve(1, 6), curve(1, 7), curve(1, 8), curve(1, 9)],
                condition: BoundaryCondition::Tangent,
                weight: 0.7,
            }],
            guide_rails: vec![],
        };
        let def = BoundaryPatchDefinition::from_recipe(recipe.clone()).unwrap();
        assert_eq!(def.recipe(), recipe);
        assert_eq!(def.edge_sets()[0].condition, BoundaryCondition::Tangent);
        assert!((def.edge_sets()[0].weight - 0.7).abs() < 1e-12);
        assert!(def.reads().keys().contains(&FeatureReadKey::Sketch(1)));
    }

    #[test]
    fn rejects_invalid_version_empty_sets_negative_weight_and_duplicates() {
        let base = BoundaryPatchRecipe {
            version: 1,
            automatic_edge_chain: false,
            edge_sets: vec![BoundaryEdgeSet {
                edges: vec![curve(1, 6)],
                condition: BoundaryCondition::Free,
                weight: 1.0,
            }],
            guide_rails: vec![],
        };
        assert!(
            BoundaryPatchDefinition::from_recipe(BoundaryPatchRecipe {
                version: 2,
                ..base.clone()
            })
            .is_err()
        );
        assert!(
            BoundaryPatchDefinition::from_recipe(BoundaryPatchRecipe {
                edge_sets: vec![],
                ..base.clone()
            })
            .is_err()
        );
        assert!(
            BoundaryPatchDefinition::from_recipe(BoundaryPatchRecipe {
                edge_sets: vec![BoundaryEdgeSet {
                    edges: vec![curve(1, 6)],
                    condition: BoundaryCondition::Tangent,
                    weight: -0.1,
                }],
                ..base.clone()
            })
            .is_err()
        );
        assert!(
            BoundaryPatchDefinition::from_recipe(BoundaryPatchRecipe {
                guide_rails: vec![curve(1, 6)],
                ..base
            })
            .is_err()
        );
    }
}
