// SPDX-License-Identifier: GPL-2.0-only

use super::{WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneError, WorkPlaneReference};
use crate::sketch::{SketchPlane, SketchPlaneOptions, SketchPlaneRecipe};

/// Versioned authored construction data; no evaluated caches are stored.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct WorkPlaneRecipe {
    pub version: u32,
    pub definition: WorkPlaneDefinitionRecipe,
}

/// One serialized definition selected by variant; scalar closures are not saved.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub enum WorkPlaneDefinitionRecipe {
    AtAngleAboutAxis(crate::feature::work_geometry::WorkPlaneLineAngle),
    ThroughTwoAxes([crate::feature::work_axis::WorkAxisReference; 2]),
    NormalToAxis(crate::feature::work_geometry::WorkPlaneLinePoint),
    LineThroughPoint(crate::feature::work_geometry::WorkPlaneLinePoint),
    ThroughThreePoints([crate::feature::work_geometry::WorkPointReference; 3]),
    ParallelThroughPoint(crate::feature::work_geometry::WorkPlaneThroughPoint),
    Axis(crate::feature::work_axis::WorkAxisLineRecipe),
    Point([crate::feature::work_geometry::DatumLength; 3]),
    AxisThroughPoints(crate::feature::work_geometry::WorkAxisPoints),
    PointAtReference(crate::feature::work_geometry::WorkPointReference),
    Fixed {
        frame: SketchPlaneRecipe,
        options: SketchPlaneOptions,
    },
    Offset {
        base: WorkPlaneReference,
        distance: WorkPlaneDistance,
    },
}

impl WorkPlaneDefinition {
    /// Capture authored source without evaluated output; example: `definition.recipe()`.
    pub fn recipe(self) -> WorkPlaneRecipe {
        let definition = self.source_recipe();
        WorkPlaneRecipe {
            version: definition.source_version(),
            definition,
        }
    }

    fn source_recipe(self) -> WorkPlaneDefinitionRecipe {
        use WorkPlaneDefinitionRecipe as Source;
        match self {
            Self::AtAngleAboutAxis(source) => Source::AtAngleAboutAxis(source),
            Self::ThroughTwoAxes(axes) => Source::ThroughTwoAxes(axes),
            Self::NormalToAxis(source) => Source::NormalToAxis(source),
            Self::LineThroughPoint(source) => Source::LineThroughPoint(source),
            Self::ThroughThreePoints(points) => Source::ThroughThreePoints(points),
            Self::ParallelThroughPoint(source) => Source::ParallelThroughPoint(source),
            Self::Axis(axis) => Source::Axis(axis.recipe()),
            Self::Point(coordinates) => Source::Point(coordinates),
            Self::PointAtReference(reference) => Source::PointAtReference(reference),
            Self::AxisThroughPoints(points) => Source::AxisThroughPoints(points),
            Self::Fixed(plane) => fixed_frame_recipe(plane),
            Self::Offset { base, distance } => Source::Offset { base, distance },
        }
    }

    /// Restore and validate a supported recipe; example: `WorkPlaneDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: WorkPlaneRecipe) -> Result<Self, WorkPlaneError> {
        let expected = recipe.definition.source_version();
        if recipe.version != expected {
            return Err(WorkPlaneError::RecipeVersion(recipe.version));
        }
        restore_definition(recipe.definition)
    }
}

fn restore_definition(
    definition: WorkPlaneDefinitionRecipe,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    use WorkPlaneDefinitionRecipe::*;
    match definition {
        AtAngleAboutAxis(source) => restore_line_angle(source),
        ThroughTwoAxes(axes) => Ok(WorkPlaneDefinition::ThroughTwoAxes(axes)),
        NormalToAxis(source) => Ok(WorkPlaneDefinition::NormalToAxis(source)),
        LineThroughPoint(source) => Ok(WorkPlaneDefinition::LineThroughPoint(source)),
        ThroughThreePoints(points) => Ok(WorkPlaneDefinition::ThroughThreePoints(points)),
        ParallelThroughPoint(source) => Ok(WorkPlaneDefinition::ParallelThroughPoint(source)),
        Axis(axis) => restore_line_axis(axis),
        Fixed { frame, options } => restore_frame(frame, options),
        Offset { base, distance } => validate_offset(base, distance),
        Point(coordinates) => restore_point(coordinates),
        PointAtReference(reference) => Ok(WorkPlaneDefinition::PointAtReference(reference)),
        AxisThroughPoints(points) => Ok(WorkPlaneDefinition::AxisThroughPoints(points)),
    }
}

fn validate_offset(
    base: WorkPlaneReference,
    distance: WorkPlaneDistance,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    distance
        .validate_source()
        .map_err(super::plane_length_error)?;
    Ok(WorkPlaneDefinition::Offset { base, distance })
}
fn restore_point(
    coordinates: [crate::feature::work_geometry::DatumLength; 3],
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    crate::feature::work_geometry::validate_position(coordinates)?;
    Ok(WorkPlaneDefinition::Point(coordinates))
}
impl WorkPlaneDefinitionRecipe {
    pub(super) fn source_version(self) -> u32 {
        match self {
            Self::AtAngleAboutAxis(_) => 10,
            Self::ThroughTwoAxes(_) => 9,
            Self::NormalToAxis(_) => 8,
            Self::LineThroughPoint(_) => 7,
            Self::ThroughThreePoints(_) => 6,
            Self::ParallelThroughPoint(_) => 5,
            Self::PointAtReference(_) => 4,
            Self::Axis(_) => 2,
            Self::Point(_) | Self::AxisThroughPoints(_) => 3,
            _ => 1,
        }
    }
    pub(super) fn minimum_program_version(self) -> u32 {
        match self {
            Self::AtAngleAboutAxis(_) => 15,
            Self::ThroughTwoAxes(_) => 14,
            Self::NormalToAxis(_) => 13,
            Self::LineThroughPoint(_) => 12,
            Self::ThroughThreePoints(_) => 11,
            Self::ParallelThroughPoint(_) => 10,
            Self::PointAtReference(_) => 6,
            Self::Axis(_) => 3,
            Self::Point(_) | Self::AxisThroughPoints(_) => 4,
            _ => 1,
        }
    }
}

fn restore_line_axis(
    recipe: crate::feature::work_axis::WorkAxisLineRecipe,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    crate::feature::work_axis::WorkAxisLine::from_recipe(recipe)
        .map(WorkPlaneDefinition::Axis)
        .map_err(WorkPlaneError::Axis)
}

fn fixed_frame_recipe(plane: SketchPlane) -> WorkPlaneDefinitionRecipe {
    WorkPlaneDefinitionRecipe::Fixed {
        frame: plane.recipe(),
        options: plane.options(),
    }
}

fn restore_frame(
    frame: SketchPlaneRecipe,
    options: SketchPlaneOptions,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    SketchPlane::from_recipe(frame, options)
        .map(WorkPlaneDefinition::Fixed)
        .map_err(WorkPlaneError::Frame)
}

fn restore_line_angle(
    source: crate::feature::work_geometry::WorkPlaneLineAngle,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    source.angle.validate_source()?;
    Ok(WorkPlaneDefinition::AtAngleAboutAxis(source))
}
