// SPDX-License-Identifier: GPL-2.0-only
//! Fixed and offset construction frames from Go work_plane.go/work_plane_defs.go, aa8297d1.

mod error;
mod evaluate;
mod recipe;
use super::{AuthoredFeature, FeatureId, FeatureReadFootprint, FeatureReadKey};
use crate::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    sketch::SketchPlane,
};
pub use error::WorkPlaneError;
pub use evaluate::WorkPlaneEvaluation;
pub use recipe::{WorkPlaneDefinitionRecipe, WorkPlaneRecipe};

/// Origin construction frames, independent of collection positions or wire strings.
#[derive(Clone, Copy, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum WorkPlaneOrigin {
    XY,
    XZ,
    YZ,
}

/// References resolve by stable feature identity, never by nearest geometry.
#[derive(Clone, Copy, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum WorkPlaneReference {
    Origin(WorkPlaneOrigin),
    Feature(FeatureId),
}

/// Offset length in document working units, or a selected parameter model value.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum WorkPlaneDistance {
    Constant(f64),
    Parameter(u64),
}

/// Concrete construction definitions; geometry consumers receive validated frames.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum WorkPlaneDefinition {
    Fixed(SketchPlane),
    Offset {
        base: WorkPlaneReference,
        distance: WorkPlaneDistance,
    },
}

/// Project-owned seam for resolving current work-plane results.
pub trait WorkPlaneResolver {
    /// Resolve one available current frame; example: `resolver.plane(FeatureId(1))?`.
    fn plane(&self, feature: FeatureId) -> Result<SketchPlane, WorkPlaneError>;
}

impl WorkPlaneOrigin {
    /// Use the validated origin frame; example: `WorkPlaneOrigin::XZ.plane()`.
    pub fn plane(self) -> SketchPlane {
        match self {
            Self::XY => SketchPlane::xy(),
            Self::XZ => SketchPlane::xz(),
            Self::YZ => SketchPlane::yz(),
        }
    }
}

impl WorkPlaneDistance {
    fn value(self, parameters: &ParameterCollection<u64>) -> Result<f64, WorkPlaneError> {
        let (parameter, quantity) = match self {
            Self::Constant(value) => (None, Quantity::new(value, Dimension::Length)),
            Self::Parameter(id) => (
                Some(id),
                parameters
                    .model_value(id)
                    .map_err(WorkPlaneError::Parameter)?,
            ),
        };
        if quantity.dimension != Dimension::Length || !quantity.value.is_finite() {
            return Err(WorkPlaneError::Offset {
                parameter,
                quantity,
            });
        }
        Ok(quantity.value)
    }
}

impl WorkPlaneDefinition {
    /// Derive all declared input reads; example: `definition.reads()` before adding it to the program.
    pub fn reads(self) -> FeatureReadFootprint {
        let Self::Offset { base, distance } = self else {
            return FeatureReadFootprint::default();
        };
        let mut keys = Vec::new();
        if let WorkPlaneReference::Feature(id) = base {
            keys.push(FeatureReadKey::Feature(id));
        }
        if let WorkPlaneDistance::Parameter(id) = distance {
            keys.push(FeatureReadKey::Parameter(id));
        }
        FeatureReadFootprint::new(keys, false)
    }

    /// Evaluate exact construction coordinates without consulting display geometry; example: `definition.evaluate(&parameters, &resolver)?`.
    pub fn evaluate(
        self,
        parameters: &ParameterCollection<u64>,
        resolver: &impl WorkPlaneResolver,
    ) -> Result<SketchPlane, WorkPlaneError> {
        match self {
            Self::Fixed(plane) => Ok(plane),
            Self::Offset { base, distance } => {
                let plane = match base {
                    WorkPlaneReference::Origin(origin) => origin.plane(),
                    WorkPlaneReference::Feature(id) => resolver.plane(id)?,
                };
                shifted_frame(plane, distance.value(parameters)?)
            }
        }
    }
}

impl AuthoredFeature<WorkPlaneDefinition> {
    /// Build an entry with definition-derived footprints; example: `AuthoredFeature::work_plane(id, "Offset", definition)`.
    pub fn work_plane(id: FeatureId, name: &str, definition: WorkPlaneDefinition) -> Self {
        let reads = definition.reads();
        Self::new(id, name, definition, reads)
    }
}

fn shifted_frame(plane: SketchPlane, distance: f64) -> Result<SketchPlane, WorkPlaneError> {
    let mut recipe = plane.recipe();
    let normal = plane.normal();
    recipe.origin = std::array::from_fn(|axis| recipe.origin[axis] + normal[axis] * distance);
    SketchPlane::from_recipe(recipe, plane.options()).map_err(WorkPlaneError::Frame)
}
