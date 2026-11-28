// SPDX-License-Identifier: GPL-2.0-only
//! Authored Part Surface Stitch feature recipe, validation, and resolution (#3918).
use super::{FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity};
use std::collections::BTreeSet;

fn default_maintain_as_surface() -> bool {
    true
}

/// Persisted Part Surface Stitch recipe (Part program version 74, source version 64).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct StitchRecipe {
    pub version: u32,
    /// Selected surface body keys in authored order (at least two distinct surfaces).
    pub surfaces: Vec<Vec<u8>>,
    /// Keep the stitched shell as a surface body even when watertight.
    #[serde(default = "default_maintain_as_surface")]
    pub maintain_as_surface: bool,
    /// Maximum stitching gap tolerance in millimeters (e.g. `0.01`, `0.5`).
    pub maximum_tolerance_mm: f64,
    /// Analyze tab option: colorize free and stitched edge conditions.
    #[serde(default)]
    pub show_edge_conditions: bool,
    /// Analyze tab option: highlight near-tangent stitched boundaries.
    #[serde(default)]
    pub show_near_tangent: bool,
}

/// Validated authored Stitch definition.
#[derive(Clone, Debug, PartialEq)]
pub struct StitchDefinition {
    recipe: StitchRecipe,
    surface_sources: Vec<FeatureId>,
}

/// Resolved Stitch inputs ready for Martensite sewing evaluation.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedStitch {
    pub maintain_as_surface: bool,
    /// Maximum stitching tolerance converted to model units (cm).
    pub maximum_tolerance_cm: f64,
    /// Maximum stitching tolerance in millimeters for diagnostics.
    pub maximum_tolerance_mm: f64,
    pub show_edge_conditions: bool,
    pub show_near_tangent: bool,
}

impl StitchDefinition {
    /// Validate and construct a Stitch definition from its serialized recipe.
    /// Example: `StitchDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: StitchRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "stitch recipe version {}: expected 1",
                recipe.version
            ));
        }
        if recipe.surfaces.len() < 2 {
            return Err(format!(
                "stitch surfaces count {}: expected at least two selected surfaces",
                recipe.surfaces.len()
            ));
        }
        let mut expected_doc = None;
        let mut seen = BTreeSet::new();
        let mut surface_sources = Vec::with_capacity(recipe.surfaces.len());
        for surface in &recipe.surfaces {
            let (doc, source) = reference_identity(surface, 4, "stitch surface")?;
            if let Some(first_doc) = expected_doc {
                if doc != first_doc {
                    return Err(format!(
                        "stitch surface document {doc:?}: expected document {first_doc:?}"
                    ));
                }
            } else {
                expected_doc = Some(doc);
            }
            if !seen.insert(surface.as_slice()) {
                return Err(format!(
                    "stitch surface {surface:?}: expected distinct selected surfaces"
                ));
            }
            surface_sources.push(source);
        }
        if !recipe.maximum_tolerance_mm.is_finite() || recipe.maximum_tolerance_mm <= 0.0 {
            return Err(format!(
                "stitch maximum_tolerance_mm {}: expected a finite positive tolerance in mm",
                recipe.maximum_tolerance_mm
            ));
        }
        Ok(Self {
            recipe,
            surface_sources,
        })
    }

    /// Copy the serializable recipe; example: `definition.recipe()`.
    pub fn recipe(&self) -> StitchRecipe {
        self.recipe.clone()
    }

    /// Selected surface body keys in authored order; example: `definition.surfaces()`.
    pub fn surfaces(&self) -> &[Vec<u8>] {
        &self.recipe.surfaces
    }

    /// Producer feature IDs of the selected surface bodies; example: `definition.surface_sources()`.
    pub fn surface_sources(&self) -> &[FeatureId] {
        &self.surface_sources
    }

    /// Whether the result must remain a surface body even if closed.
    pub fn maintain_as_surface(&self) -> bool {
        self.recipe.maintain_as_surface
    }

    /// Maximum stitching tolerance in millimeters.
    pub fn maximum_tolerance_mm(&self) -> f64 {
        self.recipe.maximum_tolerance_mm
    }

    /// Whether Show Edge Conditions is enabled on the Analyze tab.
    pub fn show_edge_conditions(&self) -> bool {
        self.recipe.show_edge_conditions
    }

    /// Whether Show Near Tangent is enabled on the Analyze tab.
    pub fn show_near_tangent(&self) -> bool {
        self.recipe.show_near_tangent
    }

    /// Attribute all read dependencies; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let keys: BTreeSet<FeatureReadKey> = self
            .surface_sources
            .iter()
            .copied()
            .map(FeatureReadKey::Feature)
            .collect();
        FeatureReadFootprint::new(keys, false)
    }

    /// Resolve authored fields for kernel sewing evaluation; example: `definition.resolve()`.
    pub fn resolve(&self) -> ResolvedStitch {
        ResolvedStitch {
            maintain_as_surface: self.recipe.maintain_as_surface,
            maximum_tolerance_cm: self.recipe.maximum_tolerance_mm * 0.1,
            maximum_tolerance_mm: self.recipe.maximum_tolerance_mm,
            show_edge_conditions: self.recipe.show_edge_conditions,
            show_near_tangent: self.recipe.show_near_tangent,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::feature::work_plane::{WorkPlaneDefinition, WorkPlaneProgramRecipe};
    use crate::feature::{AuthoredFeature, FeatureSequence};
    use crate::identity::{DocumentGuid, SemanticReferenceName};

    fn body_key(doc: DocumentGuid, feature: u64) -> Vec<u8> {
        SemanticReferenceName::new(doc, feature, 4, "thicken-offset", "body", &[1])
            .unwrap()
            .encode()
            .unwrap()
    }

    #[test]
    fn stitch_validates_inputs_and_round_trips_v74_v64() {
        let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let s1 = body_key(doc, 2);
        let s2 = body_key(doc, 3);
        let recipe = StitchRecipe {
            version: 1,
            surfaces: vec![s1.clone(), s2.clone()],
            maintain_as_surface: true,
            maximum_tolerance_mm: 0.5,
            show_edge_conditions: true,
            show_near_tangent: true,
        };
        let def = StitchDefinition::from_recipe(recipe.clone()).unwrap();
        assert_eq!(def.surfaces().len(), 2);
        assert_eq!(def.surface_sources(), &[FeatureId(2), FeatureId(3)]);
        assert!((def.resolve().maximum_tolerance_cm - 0.05).abs() < 1e-12);

        // Rejects fewer than 2 surfaces, duplicates, and non-positive tolerance.
        assert!(
            StitchDefinition::from_recipe(StitchRecipe {
                surfaces: vec![s1.clone()],
                ..recipe.clone()
            })
            .is_err()
        );
        assert!(
            StitchDefinition::from_recipe(StitchRecipe {
                surfaces: vec![s1.clone(), s1],
                ..recipe.clone()
            })
            .is_err()
        );
        assert!(
            StitchDefinition::from_recipe(StitchRecipe {
                maximum_tolerance_mm: 0.0,
                ..recipe.clone()
            })
            .is_err()
        );

        // Verify exact source version 64 and program version 74 round-trip.
        let wp_def = WorkPlaneDefinition::Stitch(Box::new(def.clone()));
        let wp_recipe = wp_def.recipe();
        assert_eq!(wp_recipe.version, 64);
        let restored = WorkPlaneDefinition::from_recipe(wp_recipe).unwrap();
        assert_eq!(restored, wp_def);

        let mut seq = FeatureSequence::default();
        let origin_plane = crate::sketch::SketchPlane::xy();
        seq.append(AuthoredFeature::new(
            FeatureId(2),
            "OffsetSrf1",
            WorkPlaneDefinition::Fixed(origin_plane),
            FeatureReadFootprint::default(),
        ))
        .unwrap();
        seq.append(AuthoredFeature::new(
            FeatureId(3),
            "OffsetSrf2",
            WorkPlaneDefinition::Fixed(origin_plane),
            FeatureReadFootprint::default(),
        ))
        .unwrap();
        seq.append(AuthoredFeature::new(
            FeatureId(4),
            "Stitch Surface1",
            wp_def,
            def.reads(),
        ))
        .unwrap();
        let program: WorkPlaneProgramRecipe = seq.part_recipe().unwrap();
        assert_eq!(program.version, 74);
        let restored_seq = FeatureSequence::from_part_recipe(&program).unwrap();
        assert_eq!(restored_seq.features().len(), 3);
    }
}
