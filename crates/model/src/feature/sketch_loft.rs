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

/// The first section's takeoff tangent and dimensionless influence on a Loft surface.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct LoftTakeoffCondition {
    pub angle_degrees: f64,
    pub weight: f64,
}

impl LoftTakeoffCondition {
    /// Validate a weighted tangent. Example: `LoftTakeoffCondition::new(90., 1.)?`.
    pub fn new(angle_degrees: f64, weight: f64) -> Result<Self, String> {
        if !angle_degrees.is_finite()
            || !(0. ..=180.).contains(&angle_degrees)
            || !weight.is_finite()
            || weight <= 0.
        {
            return Err(format!(
                "Loft takeoff ({angle_degrees}, {weight}): expected 0 to 180 degrees and a finite positive weight"
            ));
        }
        Ok(Self {
            angle_degrees,
            weight,
        })
    }
}

/// An intermediate section's normalized position and area multiplier.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchLoftPlacedSection {
    pub position: f64,
    pub area_scale: f64,
}

impl SketchLoftPlacedSection {
    /// Validate one intermediate section. Example: `SketchLoftPlacedSection::new(0.5, 1.0)?`.
    pub fn new(position: f64, area_scale: f64) -> Result<Self, String> {
        if !position.is_finite()
            || position <= 0.
            || position >= 1.
            || !area_scale.is_finite()
            || area_scale <= 0.
        {
            return Err(format!(
                "placed Loft section ({position}, {area_scale}): expected a position strictly between 0 and 1 and a finite positive area multiplier"
            ));
        }
        Ok(Self {
            position,
            area_scale,
        })
    }
}

/// The fitted circle of an intermediate section in Part coordinates.
pub struct LoftPlacedCircle {
    pub center: [f64; 3],
    pub normal: [f64; 3],
    pub x_direction: [f64; 3],
    pub radius: f64,
}

/// Persisted section identities in the order in which the loft passes through them.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchLoftRecipe {
    pub sections: [SketchLoftSection; 2],
    #[serde(default, skip_serializing_if = "SketchLoftOutput::is_solid")]
    pub output: SketchLoftOutput,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub center_line: Option<SketchLoftCenterLine>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub placed: Option<SketchLoftPlacedSection>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub takeoff: Option<LoftTakeoffCondition>,
}

/// A Loft whose sources are structurally valid and can be resolved for the current Part.
#[derive(Clone, Debug, PartialEq)]
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
    pub placed: Option<SketchLoftPlacedSection>,
    pub takeoff: Option<LoftTakeoffCondition>,
}

impl ResolvedSketchLoft {
    /// Compute the section sent to the builder. Example: `input.placed_circle()`.
    pub fn placed_circle(&self) -> Option<LoftPlacedCircle> {
        let dimensions = self.placed?;
        let [first, last] = &self.sections;
        let normal = std::array::from_fn(|axis| last.center[axis] - first.center[axis]);
        let center =
            std::array::from_fn(|axis| first.center[axis] + dimensions.position * normal[axis]);
        let radius = (first.radius + dimensions.position * (last.radius - first.radius))
            * dimensions.area_scale.sqrt();
        Some(LoftPlacedCircle {
            center,
            normal,
            x_direction: first.x_direction,
            radius,
        })
    }
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
            placed: None,
            takeoff: None,
        }))
    }

    /// Restore ordered section IDs. Example: `SketchLoftDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: SketchLoftRecipe) -> Result<Self, String> {
        let mut loft = Self::new(recipe.sections)?.with_output(recipe.output);
        if let Some(path) = recipe.center_line {
            loft = loft.with_center_line(path)?;
        }
        loft.with_placed_section(recipe.placed)?
            .with_takeoff(recipe.takeoff)
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

    /// Set the intermediate section. Example: `loft.with_placed_section(Some(section))?`.
    pub fn with_placed_section(
        mut self,
        placed: Option<SketchLoftPlacedSection>,
    ) -> Result<Self, String> {
        if let Some(section) = placed {
            if self.0.center_line.is_none() {
                return Err("A placed Loft section requires a Center Line.".into());
            }
            SketchLoftPlacedSection::new(section.position, section.area_scale)?;
        }
        self.0.placed = placed;
        Ok(self)
    }

    /// Set the first section condition. Example: `loft.with_takeoff(Some(condition))?`.
    pub fn with_takeoff(mut self, takeoff: Option<LoftTakeoffCondition>) -> Result<Self, String> {
        if let Some(condition) = takeoff {
            if self.0.output != SketchLoftOutput::Surface || self.0.center_line.is_some() {
                return Err(
                    "A weighted takeoff requires a two-section Loft surface without a guide."
                        .into(),
                );
            }
            LoftTakeoffCondition::new(condition.angle_degrees, condition.weight)?;
        }
        self.0.takeoff = takeoff;
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

    /// Resolve the current geometry for preview or replay. Example: `loft.resolve(snapshot, datums)?`.
    pub fn resolve(
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
            placed: self.0.placed,
            takeoff: self.0.takeoff,
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
