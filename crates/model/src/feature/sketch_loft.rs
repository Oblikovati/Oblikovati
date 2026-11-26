// SPDX-License-Identifier: GPL-2.0-only
//! The ordered authored circle sections of an independent Loft solid.
use super::{FeatureReadFootprint, FeatureReadKey, work_geometry::WorkGeometryEvaluation};
use crate::{
    document_sketch::DocumentSketchId,
    history::{DocumentSnapshot, ResolvedDocumentSketch},
    sketch::SketchEntityId,
};

/// An exact authored circle in its source sketch.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchLoftSection {
    pub sketch: DocumentSketchId,
    pub circle: SketchEntityId,
}

/// Persisted section identities in the order in which the loft passes through them.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchLoftRecipe {
    pub sections: [SketchLoftSection; 2],
}

/// A Loft whose sources are structurally valid and can be resolved for the current Part.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchLoftDefinition(SketchLoftRecipe);

/// A solved section in database coordinates with its source identity.
pub struct ResolvedLoftCircle {
    pub source: SketchLoftSection,
    pub center: [f64; 3],
    pub normal: [f64; 3],
    pub x_direction: [f64; 3],
    pub radius: f64,
}

impl SketchLoftDefinition {
    /// Validate the authored section pair. Example: `SketchLoftDefinition::new([base, tip])?`.
    pub fn new(sections: [SketchLoftSection; 2]) -> Result<Self, String> {
        if sections[0].sketch == sections[1].sketch
            || sections
                .iter()
                .any(|section| section.sketch.0 == 0 || section.circle.0 == 0)
        {
            return Err(format!(
                "Loft sections {sections:?}: expected two distinct, named sketch circles"
            ));
        }
        Ok(Self(SketchLoftRecipe { sections }))
    }

    /// Restore ordered section IDs. Example: `SketchLoftDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: SketchLoftRecipe) -> Result<Self, String> {
        Self::new(recipe.sections)
    }

    /// Read the authored sources without geometry. Example: `loft.recipe().sections`.
    pub fn recipe(&self) -> &SketchLoftRecipe {
        &self.0
    }

    pub(crate) fn source_reads(&self) -> FeatureReadFootprint {
        FeatureReadFootprint::new(
            self.0
                .sections
                .map(|section| FeatureReadKey::Sketch(section.sketch.0)),
            false,
        )
    }

    /// Collect source sketches and their datum dependencies. Example: `loft.reads(snapshot)?`.
    pub fn reads(&self, snapshot: &DocumentSnapshot) -> Result<FeatureReadFootprint, String> {
        let mut keys = self.source_reads().keys().clone();
        let mut unattributed = false;
        for section in self.0.sections {
            let sketch = snapshot
                .sketches
                .get(section.sketch)
                .map_err(|cause| cause.to_string())?;
            keys.extend(sketch.read_footprint().keys());
            unattributed |= sketch.read_footprint().has_unattributed_reads();
        }
        Ok(FeatureReadFootprint::new(keys, unattributed))
    }

    pub(super) fn resolve(
        &self,
        snapshot: &DocumentSnapshot,
        datums: &WorkGeometryEvaluation,
    ) -> Result<[ResolvedLoftCircle; 2], String> {
        let resolve = |section: SketchLoftSection| {
            let source = snapshot
                .require_converged_sketch_with_datums(section.sketch, datums)
                .map_err(|cause| cause.to_string())?;
            circle_in_part(&source, section)
        };
        let [first, last] = self.0.sections;
        Ok([resolve(first)?, resolve(last)?])
    }
}

fn circle_in_part(
    source: &ResolvedDocumentSketch<'_>,
    section: SketchLoftSection,
) -> Result<ResolvedLoftCircle, String> {
    let geometry = source.sketch.geometry();
    if geometry
        .is_construction(section.circle)
        .map_err(|cause| cause.to_string())?
    {
        return Err(format!(
            "Loft section {section:?}: expected a regular circle"
        ));
    }
    let circle = geometry
        .circle(section.circle)
        .map_err(|cause| cause.to_string())?;
    let center = geometry
        .point(circle.center)
        .map_err(|cause| cause.to_string())?;
    Ok(ResolvedLoftCircle {
        source: section,
        center: source
            .plane
            .to_model(center.position)
            .map_err(|cause| cause.to_string())?,
        normal: source.plane.normal(),
        x_direction: source.plane.recipe().x_axis,
        radius: circle.radius,
    })
}
