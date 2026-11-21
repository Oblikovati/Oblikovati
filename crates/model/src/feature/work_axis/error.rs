// SPDX-License-Identifier: GPL-2.0-only
use crate::sketch::SketchError;
use std::{error::Error, fmt};

/// The offending axis input and the checked metric cause remain available to the owner.
#[derive(Clone, Debug, PartialEq)]
pub enum WorkAxisError {
    Origin([f64; 3]),
    Direction {
        direction: [f64; 3],
        cause: SketchError,
    },
    RecipeVersion(u32),
}
impl fmt::Display for WorkAxisError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Origin(origin) => write!(
                output,
                "work axis origin {origin:?}: expected three finite document working-unit components"
            ),
            Self::Direction { direction, cause } => write!(
                output,
                "work axis direction {direction:?}: expected a finite nonzero direction and checked unit components: {cause}"
            ),
            Self::RecipeVersion(version) => write!(
                output,
                "work axis line recipe version {version}: expected version 1"
            ),
        }
    }
}
impl Error for WorkAxisError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Direction { cause, .. } => Some(cause),
            _ => None,
        }
    }
}
