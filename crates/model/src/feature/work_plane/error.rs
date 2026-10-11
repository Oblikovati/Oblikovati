// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    feature::FeatureId, parameter::ParameterError, quantity::Quantity, sketch::SketchError,
};
use std::fmt::{Display, Formatter};

/// A failed construction frame has no usable current geometry.
#[derive(Clone, Debug, PartialEq)]
pub enum WorkPlaneError {
    Frame(SketchError),
    Parameter(ParameterError<u64>),
    Unavailable(FeatureId),
    Offset {
        parameter: Option<u64>,
        quantity: Quantity,
    },
    RecipeVersion(u32),
}

impl Display for WorkPlaneError {
    fn fmt(&self, formatter: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Frame(error) => Display::fmt(error, formatter),
            Self::Parameter(error) => Display::fmt(error, formatter),
            Self::Unavailable(id) => write!(
                formatter,
                "work plane {id:?}: expected an available current frame"
            ),
            Self::Offset {
                parameter,
                quantity,
            } => write!(
                formatter,
                "work plane offset parameter {parameter:?} value {quantity:?}: expected finite Length in document working units"
            ),
            Self::RecipeVersion(version) => {
                write!(formatter, "work plane recipe version {version}: expected 1")
            }
        }
    }
}

impl std::error::Error for WorkPlaneError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Frame(error) => Some(error),
            Self::Parameter(error) => Some(error),
            _ => None,
        }
    }
}
