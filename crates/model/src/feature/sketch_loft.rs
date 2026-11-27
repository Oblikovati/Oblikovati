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

/// One authored straight curve through the Loft section centers.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchLoftCenterLine {
    pub sketch: DocumentSketchId,
    pub line: SketchEntityId,
}

/// A closed solid or a shell lofted through the current section boundaries.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum SketchLoftOutput {
    #[default]
    Solid,
    Surface,
}

impl SketchLoftOutput {
    fn is_solid(&self) -> bool {
        *self == Self::Solid
    }
}

/// Persisted section identities in the order in which the loft passes through them.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchLoftRecipe {
    pub sections: [SketchLoftSection; 2],
    #[serde(default, skip_serializing_if = "SketchLoftOutput::is_solid")]
    pub output: SketchLoftOutput,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub center_line: Option<SketchLoftCenterLine>,
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

/// Linear path geometry in the current Part coordinates.
pub struct ResolvedLoftCenterLine {
    pub source: SketchLoftCenterLine,
    pub endpoints: [[f64; 3]; 2],
}

/// Current resolved source geometry and the authored result kind.
pub struct ResolvedSketchLoft {
    pub sections: [ResolvedLoftCircle; 2],
    pub output: SketchLoftOutput,
    pub center_line: Option<ResolvedLoftCenterLine>,
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
        Ok(Self(SketchLoftRecipe {
            sections,
            output: SketchLoftOutput::Solid,
            center_line: None,
        }))
    }

    /// Restore ordered section IDs. Example: `SketchLoftDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: SketchLoftRecipe) -> Result<Self, String> {
        let loft = Self::new(recipe.sections)?.with_output(recipe.output);
        match recipe.center_line {
            Some(source) => loft.with_center_line(source),
            None => Ok(loft),
        }
    }

    /// Choose the result kind over the same sections. Example: `loft.with_output(SketchLoftOutput::Surface)`.
    pub fn with_output(mut self, output: SketchLoftOutput) -> Self {
        self.0.output = output;
        self
    }

    /// Bind the guide by authored identity. Example: `loft.with_center_line(path)?`.
    pub fn with_center_line(mut self, path: SketchLoftCenterLine) -> Result<Self, String> {
        if path.sketch.0 == 0
            || path.line.0 == 0
            || self
                .0
                .sections
                .iter()
                .any(|section| section.sketch == path.sketch)
        {
            return Err(format!(
                "Loft center line {path:?}: expected a named line in a separate sketch"
            ));
        }
        self.0.center_line = Some(path);
        Ok(self)
    }

    /// Visit the sketches the definition consumes. Example: `loft.source_sketches().count()`.
    pub fn source_sketches(&self) -> impl Iterator<Item = DocumentSketchId> + '_ {
        self.0
            .sections
            .iter()
            .map(|section| section.sketch)
            .chain(self.0.center_line.map(|path| path.sketch))
    }

    /// Read the authored sources without geometry. Example: `loft.recipe().sections`.
    pub fn recipe(&self) -> &SketchLoftRecipe {
        &self.0
    }

    pub(crate) fn source_reads(&self) -> FeatureReadFootprint {
        FeatureReadFootprint::new(
            self.source_sketches()
                .map(|sketch| FeatureReadKey::Sketch(sketch.0)),
            false,
        )
    }

    /// Collect source sketches and their datum dependencies. Example: `loft.reads(snapshot)?`.
    pub fn reads(&self, snapshot: &DocumentSnapshot) -> Result<FeatureReadFootprint, String> {
        let mut keys = self.source_reads().keys().clone();
        let mut unattributed = false;
        for source in self.source_sketches() {
            let sketch = snapshot
                .sketches
                .get(source)
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
    ) -> Result<ResolvedSketchLoft, String> {
        let resolve = |section: SketchLoftSection| {
            let source = snapshot
                .require_converged_sketch_with_datums(section.sketch, datums)
                .map_err(|cause| cause.to_string())?;
            circle_in_part(&source, section)
        };
        let [first, last] = self.0.sections;
        let center_line = self
            .0
            .center_line
            .map(|path| {
                let source = snapshot
                    .require_converged_sketch_with_datums(path.sketch, datums)
                    .map_err(|cause| cause.to_string())?;
                path_in_part(&source, path)
            })
            .transpose()?;
        Ok(ResolvedSketchLoft {
            sections: [resolve(first)?, resolve(last)?],
            output: self.0.output,
            center_line,
        })
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

fn path_in_part(
    source: &ResolvedDocumentSketch<'_>,
    path: SketchLoftCenterLine,
) -> Result<ResolvedLoftCenterLine, String> {
    let geometry = source.sketch.geometry();
    if geometry
        .is_construction(path.line)
        .map_err(|cause| cause.to_string())?
    {
        return Err(format!(
            "Loft center line {path:?}: expected a regular straight curve"
        ));
    }
    let line = geometry
        .line(path.line)
        .map_err(|cause| cause.to_string())?;
    let project = |endpoint| {
        let point = geometry
            .point(endpoint)
            .map_err(|cause| cause.to_string())?;
        source
            .plane
            .to_model(point.position)
            .map_err(|cause| cause.to_string())
    };
    Ok(ResolvedLoftCenterLine {
        source: path,
        endpoints: [project(line.endpoints[0])?, project(line.endpoints[1])?],
    })
}
