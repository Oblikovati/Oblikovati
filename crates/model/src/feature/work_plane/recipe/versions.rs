// SPDX-License-Identifier: GPL-2.0-only

use super::WorkPlaneDefinitionRecipe;
use crate::feature::work_plane::WorkPlaneReference;

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

    pub(in crate::feature::work_plane) fn source_version(&self) -> u32 {
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
            Self::SketchLoft(recipe) if recipe.placed.is_some() => 33,
            Self::SketchLoft(recipe) if recipe.center_line.is_some() => 32,
            Self::SketchLoft(recipe) => match recipe.output {
                crate::feature::sketch_loft::SketchLoftOutput::Solid => 30,
                crate::feature::sketch_loft::SketchLoftOutput::Surface => 31,
            },
            Self::SketchRevolve(_) => 38,
            Self::SketchCoil(_) => 39,
            Self::SketchEmboss(_) => 40,
            Self::DerivedPart(_) => 41,
            Self::SketchRib(_) => 42,
            Self::SketchDecal(_) => 43,
            Self::ImportedModel(_) => 44,
            Self::Unwrap(_) => 45,
            Self::Slot(_) => 46,
            Self::Combine(_) => 49,
            Self::DirectEdit(_) => 52,
            Self::Draft(_) => 47,
            Self::Thread(_) => 48,
            Self::DeleteFace(_) => 53,
            Self::ThickenOffset(_) => 50,
            Self::Split(_) => 51,
            Self::Mark(_) => 54,
            Self::WorkPlaneExtended(_) => 58,
            Self::SketchSweep(recipe) => match recipe.version {
                7 => 29,
                6 => 28,
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
            Self::SketchPrism(recipe) if recipe.version == 5 => 37,
            Self::SketchPrism(recipe) if recipe.version == 4 => 36,
            Self::SketchPrism(recipe) if recipe.intersection_target.is_some() => 35,
            Self::SketchPrism(recipe) if recipe.taper.is_some() => 34,
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

    pub(in crate::feature::work_plane) fn minimum_program_version(&self) -> u32 {
        if self.has_go_face_reference() {
            return 19;
        }
        if self.has_face_reference() {
            return 18;
        }
        match self {
            Self::Chamfer(recipe) if recipe.version == 2 => 55,
            Self::Chamfer(_) => 21,
            Self::Hole(recipe) if recipe.version == 2 => 52,
            Self::Hole(_) => 22,
            Self::MoveBody(_) => 23,
            Self::Shell(recipe) if recipe.version == 2 => 56,
            Self::Shell(_) => 24,
            Self::SketchLoft(recipe) if recipe.takeoff.is_some() => 39,
            Self::SketchLoft(recipe) if recipe.placed.is_some() => 38,
            Self::SketchLoft(recipe) if recipe.center_line.is_some() => 37,
            Self::SketchLoft(recipe) => match recipe.output {
                crate::feature::sketch_loft::SketchLoftOutput::Solid => 35,
                crate::feature::sketch_loft::SketchLoftOutput::Surface => 36,
            },
            Self::SketchRevolve(_) => 44,
            Self::SketchCoil(_) => 45,
            Self::SketchEmboss(_) => 46,
            Self::DerivedPart(_) => 47,
            Self::SketchRib(_) => 48,
            Self::SketchDecal(_) => 49,
            Self::ImportedModel(_) => 50,
            Self::Unwrap(_) => 51,
            Self::Slot(_) => 53,
            Self::Combine(_) => 59,
            Self::DirectEdit(_) => 62,
            Self::Draft(_) => 57,
            Self::Thread(_) => 58,
            Self::DeleteFace(_) => 63,
            Self::ThickenOffset(_) => 60,
            Self::Split(_) => 61,
            Self::Mark(_) => 64,
            Self::WorkPlaneExtended(_) => 68,
            Self::SketchSweep(recipe) => match recipe.version {
                7 => 34,
                6 => 33,
                5 => 32,
                4 => 31,
                3 => 30,
                2 => 29,
                _ => 28,
            },
            Self::Mirror(_) => 27,
            Self::CircularPattern(_) => 26,
            Self::RectangularPattern(_) => 25,
            Self::Fillet(recipe) if recipe.version == 2 => 54,
            Self::Fillet(_) => 20,
            Self::DeleteBody(_) => 17,
            Self::SketchPrism(recipe) if recipe.version == 5 => 43,
            Self::SketchPrism(recipe) if recipe.version == 4 => 42,
            Self::SketchPrism(recipe) if recipe.intersection_target.is_some() => 41,
            Self::SketchPrism(recipe) if recipe.taper.is_some() => 40,
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
