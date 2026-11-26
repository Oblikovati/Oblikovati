// SPDX-License-Identifier: GPL-2.0-only

use super::{WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneError, WorkPlaneReference};
use crate::sketch::{SketchPlane, SketchPlaneOptions, SketchPlaneRecipe};

/// Versioned authored construction data; no evaluated caches are stored.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct WorkPlaneRecipe {
    pub version: u32,
    pub definition: WorkPlaneDefinitionRecipe,
}

/// One serialized definition selected by variant; scalar closures are not saved.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub enum WorkPlaneDefinitionRecipe {
    DeleteBody(Vec<u8>),
    Chamfer(crate::feature::chamfer::ChamferRecipe),
    Fillet(crate::feature::fillet::FilletRecipe),
    Hole(crate::feature::hole::HoleRecipe),
    MoveBody(crate::feature::move_body::MoveBodyRecipe),
    Shell(crate::feature::shell::ShellRecipe),
    Mirror(crate::feature::mirror::MirrorRecipe),
    CircularPattern(crate::feature::circular_pattern::CircularPatternRecipe),
    RectangularPattern(crate::feature::rectangular_pattern::RectangularPatternRecipe),
    SketchPrism(crate::feature::sketch_prism::SketchPrismRecipe),
    SketchSweep(crate::feature::sketch_sweep::SketchSweepRecipe),
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
    pub fn recipe(&self) -> WorkPlaneRecipe {
        let definition = self.source_recipe();
        WorkPlaneRecipe {
            version: definition.source_version(),
            definition,
        }
    }

    fn source_recipe(&self) -> WorkPlaneDefinitionRecipe {
        use WorkPlaneDefinitionRecipe as Source;
        match self {
            Self::DeleteBody(definition) => Source::DeleteBody(definition.key().to_vec()),
            Self::Chamfer(definition) => Source::Chamfer(definition.recipe()),
            Self::Fillet(definition) => Source::Fillet(definition.recipe()),
            Self::Hole(definition) => Source::Hole(definition.recipe()),
            Self::MoveBody(definition) => Source::MoveBody(definition.recipe()),
            Self::Shell(definition) => Source::Shell(definition.recipe()),
            Self::Mirror(mirror) => Source::Mirror(mirror.recipe().clone()),
            Self::CircularPattern(pattern) => Source::CircularPattern(pattern.recipe().clone()),
            Self::RectangularPattern(pattern) => {
                Source::RectangularPattern(pattern.recipe().clone())
            }
            Self::SketchSweep(definition) => Source::SketchSweep(definition.recipe().clone()),
            Self::SketchPrism(definition) => Source::SketchPrism(definition.recipe()),
            Self::AtAngleAboutAxis(source) => Source::AtAngleAboutAxis(source.clone()),
            Self::ThroughTwoAxes(axes) => Source::ThroughTwoAxes(*axes),
            Self::NormalToAxis(source) => Source::NormalToAxis(*source),
            Self::LineThroughPoint(source) => Source::LineThroughPoint(*source),
            Self::ThroughThreePoints(points) => Source::ThroughThreePoints(*points),
            Self::ParallelThroughPoint(source) => Source::ParallelThroughPoint(source.clone()),
            Self::Axis(axis) => Source::Axis(axis.recipe()),
            Self::Point(coordinates) => Source::Point(*coordinates),
            Self::PointAtReference(reference) => Source::PointAtReference(*reference),
            Self::AxisThroughPoints(points) => Source::AxisThroughPoints(*points),
            Self::Fixed(plane) => fixed_frame_recipe(*plane),
            Self::Offset { base, distance } => offset_source_recipe(base.clone(), *distance),
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
        Mirror(recipe) => crate::feature::mirror::MirrorDefinition::new(recipe)
            .map(|definition| WorkPlaneDefinition::Mirror(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        CircularPattern(recipe) => {
            crate::feature::circular_pattern::CircularPatternDefinition::new(recipe)
                .map(|definition| WorkPlaneDefinition::CircularPattern(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        RectangularPattern(recipe) => {
            crate::feature::rectangular_pattern::RectangularPatternDefinition::new(recipe)
                .map(|definition| WorkPlaneDefinition::RectangularPattern(Box::new(definition)))
                .map_err(WorkPlaneError::BodyReference)
        }
        Shell(recipe) => crate::feature::shell::ShellDefinition::from_recipe(&recipe)
            .map(|definition| WorkPlaneDefinition::Shell(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        MoveBody(recipe) => crate::feature::move_body::MoveBodyDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::MoveBody(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        Hole(recipe) => crate::feature::hole::HoleDefinition::from_recipe(recipe)
            .map(|definition| WorkPlaneDefinition::Hole(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        Chamfer(recipe) => crate::feature::chamfer::ChamferDefinition::from_recipe(&recipe)
            .map(|definition| WorkPlaneDefinition::Chamfer(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        Fillet(recipe) => crate::feature::fillet::FilletDefinition::from_recipe(&recipe)
            .map(|definition| WorkPlaneDefinition::Fillet(Box::new(definition)))
            .map_err(WorkPlaneError::BodyReference),
        DeleteBody(key) => crate::feature::part::DeleteBodyDefinition::new(key)
            .map(WorkPlaneDefinition::DeleteBody)
            .map_err(WorkPlaneError::BodyReference),
        SketchSweep(recipe) => {
            crate::feature::sketch_sweep::SketchSweepDefinition::from_recipe(recipe)
                .map(|definition| WorkPlaneDefinition::SketchSweep(Box::new(definition)))
                .map_err(|cause| WorkPlaneError::Sweep(Box::new(cause)))
        }
        SketchPrism(recipe) => restore_prism_inputs(recipe),
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
    fn base_reference(&self) -> Option<&WorkPlaneReference> {
        match self {
            Self::Offset { base, .. } => Some(base),
            Self::ParallelThroughPoint(source) => Some(&source.base),
            Self::AtAngleAboutAxis(source) => Some(&source.base),
            _ => None,
        }
    }
    fn has_go_face_reference(&self) -> bool {
        matches!(
            self.base_reference(),
            Some(WorkPlaneReference::GoPrismFace(_))
        )
    }
    fn has_face_reference(&self) -> bool {
        matches!(self.base_reference(), Some(WorkPlaneReference::Face(_)))
    }
    pub(super) fn source_version(&self) -> u32 {
        if self.has_go_face_reference() {
            return 14;
        }
        if self.has_face_reference() {
            return 13;
        }
        match self {
            Self::Chamfer(_) => 16,
            Self::Hole(_) => 17,
            Self::MoveBody(_) => 18,
            Self::Shell(_) => 19,
            Self::SketchSweep(recipe) => match recipe.version {
                5 => 27,
                4 => 26,
                3 => 25,
                2 => 24,
                _ => 23,
            },
            Self::Mirror(_) => 22,
            Self::CircularPattern(_) => 21,
            Self::RectangularPattern(_) => 20,
            Self::Fillet(_) => 15,
            Self::DeleteBody(_) => 12,
            Self::SketchPrism(_) => 11,
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
    pub(super) fn minimum_program_version(&self) -> u32 {
        if self.has_go_face_reference() {
            return 19;
        }
        if self.has_face_reference() {
            return 18;
        }
        match self {
            Self::Chamfer(_) => 21,
            Self::Hole(_) => 22,
            Self::MoveBody(_) => 23,
            Self::Shell(_) => 24,
            Self::SketchSweep(recipe) => match recipe.version {
                5 => 32,
                4 => 31,
                3 => 30,
                2 => 29,
                _ => 28,
            },
            Self::Mirror(_) => 27,
            Self::CircularPattern(_) => 26,
            Self::RectangularPattern(_) => 25,
            Self::Fillet(_) => 20,
            Self::DeleteBody(_) => 17,
            Self::SketchPrism(_) => 16,
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

fn restore_prism_inputs(
    recipe: crate::feature::sketch_prism::SketchPrismRecipe,
) -> Result<WorkPlaneDefinition, WorkPlaneError> {
    {
        crate::feature::sketch_prism::SketchPrismDefinition::from_recipe(&recipe)
            .map(|definition| WorkPlaneDefinition::SketchPrism(Box::new(definition)))
            .map_err(WorkPlaneError::Prism)
    }
}

fn offset_source_recipe(
    base: WorkPlaneReference,
    distance: WorkPlaneDistance,
) -> WorkPlaneDefinitionRecipe {
    WorkPlaneDefinitionRecipe::Offset { base, distance }
}
