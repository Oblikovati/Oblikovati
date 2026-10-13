// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    feature::FeatureId, parameter::ParameterError, quantity::Quantity, sketch::SketchError,
};
use std::fmt::{Display, Formatter};

/// A failed construction frame has no usable current geometry.
#[derive(Clone, Debug, PartialEq)]
pub enum WorkGeometryError {
    BodyReference(String),
    Prism(crate::feature::sketch_prism::SketchPrismError),
    Angle(super::DatumAngleError),
    DisplayCorners([[f64; 3]; 2]),
    Frame(SketchError),
    Kind(WorkGeometryKindMismatch),
    Coordinate(super::point::WorkPointCoordinateError),
    Axis(crate::feature::work_axis::WorkAxisError),
    NonPlane(Box<super::WorkGeometryDefinition>),
    Parameter(ParameterError<u64>),
    Unavailable {
        feature: FeatureId,
        expected: &'static str,
    },
    Offset {
        parameter: Option<u64>,
        quantity: Quantity,
    },
    RecipeVersion(u32),
}

impl Display for WorkGeometryError {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        use WorkGeometryError::*;
        match self {
            BodyReference(cause) => f.write_str(cause),
            Angle(error) => Display::fmt(error, f),
            DisplayCorners(corners) => write_display_corners(*corners, f),
            Frame(error) => Display::fmt(error, f),
            Kind(error) => Display::fmt(error, f),
            Coordinate(error) => Display::fmt(error, f),
            Axis(error) => Display::fmt(error, f),
            NonPlane(definition) => write_non_plane(definition, f),
            Prism(cause) => Display::fmt(cause, f),
            Parameter(error) => Display::fmt(error, f),
            Unavailable { feature, expected } => write_unavailable(*feature, expected, f),
            Offset {
                parameter,
                quantity,
            } => write_offset(*parameter, *quantity, f),
            RecipeVersion(version) => write_recipe_version(*version, f),
        }
    }
}
/// A typed kind decline names the selected feature and the expected current view.
#[derive(Clone, Debug, PartialEq)]
pub struct WorkGeometryKindMismatch {
    pub feature: FeatureId,
    pub expected: &'static str,
    pub actual: &'static str,
}
impl Display for WorkGeometryKindMismatch {
    fn fmt(&self, formatter: &mut Formatter<'_>) -> std::fmt::Result {
        let Self {
            feature,
            expected,
            actual,
        } = self;
        write!(
            formatter,
            "datum {feature:?} kind {actual}: expected an available current {expected}"
        )
    }
}
fn write_non_plane(
    definition: &super::WorkGeometryDefinition,
    formatter: &mut Formatter<'_>,
) -> std::fmt::Result {
    write!(
        formatter,
        "datum definition {definition:?}: expected a plane frame"
    )
}
fn write_unavailable(
    id: FeatureId,
    expected: &str,
    formatter: &mut Formatter<'_>,
) -> std::fmt::Result {
    write!(
        formatter,
        "datum {id:?}: expected an available current {expected}"
    )
}
fn write_offset(
    parameter: Option<u64>,
    quantity: Quantity,
    formatter: &mut Formatter<'_>,
) -> std::fmt::Result {
    write!(
        formatter,
        "work plane offset parameter {parameter:?} value {quantity:?}: expected finite Length in document working units"
    )
}
fn write_recipe_version(version: u32, formatter: &mut Formatter<'_>) -> std::fmt::Result {
    write!(
        formatter,
        "work plane recipe version {version}: expected 1 for planes, 2 for line axes, 3 for positions and two-point axes, 4 for associative points, 5 for plane-and-point definitions, 6 for three-point definitions, 7 for line-and-point definitions, 8 for normal-to-axis definitions, 9 for two-axis definitions, 10 for line-angle definitions, 11 for sketch prism inputs, or 12 for body deletion"
    )
}

impl std::error::Error for WorkGeometryError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Prism(error) => Some(error),
            Self::Angle(error) => Some(error),
            Self::Frame(error) => Some(error),
            Self::Axis(error) => Some(error),
            Self::Coordinate(error) => Some(error),
            Self::Parameter(error) => Some(error),
            _ => None,
        }
    }
}

fn write_display_corners(
    corners: [[f64; 3]; 2],
    formatter: &mut Formatter<'_>,
) -> std::fmt::Result {
    write!(
        formatter,
        "plane display corners {corners:?}: expected two finite three-component points in centimetres"
    )
}
