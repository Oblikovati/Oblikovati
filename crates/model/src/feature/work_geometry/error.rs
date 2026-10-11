// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    feature::FeatureId, parameter::ParameterError, quantity::Quantity, sketch::SketchError,
};
use std::fmt::{Display, Formatter};

/// A failed construction frame has no usable current geometry.
#[derive(Clone, Debug, PartialEq)]
pub enum WorkGeometryError {
    Frame(SketchError),
    Kind {
        feature: FeatureId,
        expected: &'static str,
        actual: &'static str,
    },
    Axis(crate::feature::work_axis::WorkAxisError),
    AxisInsteadOfPlane(crate::feature::work_axis::WorkAxisLine),
    Parameter(ParameterError<u64>),
    Unavailable(FeatureId),
    Offset {
        parameter: Option<u64>,
        quantity: Quantity,
    },
    RecipeVersion(u32),
}

impl Display for WorkGeometryError {
    fn fmt(&self, formatter: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Frame(error) => Display::fmt(error, formatter),
            Self::Kind {
                feature,
                expected,
                actual,
            } => write_kind(*feature, expected, actual, formatter),
            Self::Axis(error) => Display::fmt(error, formatter),
            Self::AxisInsteadOfPlane(axis) => write_axis(*axis, formatter),
            Self::Parameter(error) => Display::fmt(error, formatter),
            Self::Unavailable(id) => write_unavailable(*id, formatter),
            Self::Offset {
                parameter,
                quantity,
            } => write_offset(*parameter, *quantity, formatter),
            Self::RecipeVersion(version) => write_recipe_version(*version, formatter),
        }
    }
}
fn write_kind(
    feature: FeatureId,
    expected: &str,
    actual: &str,
    formatter: &mut Formatter<'_>,
) -> std::fmt::Result {
    write!(
        formatter,
        "datum {feature:?} kind {actual}: expected an available current {expected}"
    )
}
fn write_axis(
    axis: crate::feature::work_axis::WorkAxisLine,
    formatter: &mut Formatter<'_>,
) -> std::fmt::Result {
    write!(formatter, "datum axis {axis:?}: expected a plane frame")
}
fn write_unavailable(id: FeatureId, formatter: &mut Formatter<'_>) -> std::fmt::Result {
    write!(
        formatter,
        "work plane {id:?}: expected an available current frame"
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
        "work plane recipe version {version}: expected 1 for planes or 2 for axes"
    )
}

impl std::error::Error for WorkGeometryError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Frame(error) => Some(error),
            Self::Axis(error) => Some(error),
            Self::Parameter(error) => Some(error),
            _ => None,
        }
    }
}
