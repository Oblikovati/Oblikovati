// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches, PlacedSketch, PlacedSketchRecipe};
use crate::{
    feature::{
        FeatureId, FeatureReadFootprint, FeatureReadKey,
        work_plane::{WorkPlaneError, WorkPlaneResolver},
    },
    sketch::{Sketch2D, SketchError, SketchPlane},
};

/// Authored placement; a feature reference stores no derived frame.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum SketchPlacement {
    Fixed(SketchPlane),
    WorkPlane(FeatureId),
}
impl SketchPlacement {
    pub(super) fn same_recipe(self, other: Self) -> bool {
        match (self, other) {
            (Self::Fixed(first), Self::Fixed(second)) => first.same_recipe(second),
            (Self::WorkPlane(first), Self::WorkPlane(second)) => first == second,
            _ => false,
        }
    }
    pub(super) fn from_recipe(
        recipe: &PlacedSketchRecipe,
        version: u32,
    ) -> Result<Self, SketchError> {
        match (recipe.plane, recipe.plane_options, recipe.plane_feature) {
            (Some(frame), Some(options), None) => {
                SketchPlane::from_recipe(frame, options).map(Self::Fixed)
            }
            (None, None, Some(feature)) if version >= 2 && feature.0 != 0 => {
                Ok(Self::WorkPlane(feature))
            }
            shape => Err(SketchError::Definition(format!(
                "sketch {:?} placement {shape:?} in recipe version {version}: expected a fixed frame with options, or one nonzero feature ID in version 2 or 3",
                recipe.id
            ))),
        }
    }
}
impl PlacedSketch {
    /// Inspect the authored reference; example: `placed.placement()` for dependency queries.
    pub fn placement(&self) -> SketchPlacement {
        self.placement
    }
    /// Attribute local dimensions and the host feature; example: `placed.read_footprint()` for feature replay.
    pub fn read_footprint(&self) -> FeatureReadFootprint {
        let mut keys: Vec<FeatureReadKey> = self
            .geometry
            .parameter_dependencies()
            .into_iter()
            .map(FeatureReadKey::Parameter)
            .collect();
        keys.extend(self.placement_read_keys());
        FeatureReadFootprint::new(keys, false)
    }
    pub(crate) fn placement_read_keys(&self) -> Vec<FeatureReadKey> {
        let mut keys = Vec::new();
        if let SketchPlacement::WorkPlane(feature) = self.placement {
            keys.push(FeatureReadKey::Feature(feature));
        }
        if let Some(orientation) = self.orientation {
            keys.extend(orientation.read_keys());
        }
        keys
    }
    /// Resolve the frame without a stale fallback; example: `placed.plane(Some(&current_planes))?`.
    pub fn plane(
        &self,
        planes: Option<&dyn WorkPlaneResolver>,
    ) -> Result<SketchPlane, WorkPlaneError> {
        self.plane_with_axes(planes, &super::orientation::UnavailableSketchAxes)
    }
    /// Resolve the current base and axis before orientation; example: `placed.plane_with_axes(Some(&planes), &axes)?`.
    pub fn plane_with_axes(
        &self,
        planes: Option<&dyn WorkPlaneResolver>,
        axes: &dyn super::SketchAxisResolver,
    ) -> Result<SketchPlane, WorkPlaneError> {
        let base = self.base_plane(planes)?;
        super::orientation::orient_current(base, self.orientation, axes)
            .map_err(WorkPlaneError::Frame)
    }
    fn base_plane(
        &self,
        planes: Option<&dyn WorkPlaneResolver>,
    ) -> Result<SketchPlane, WorkPlaneError> {
        match self.placement {
            SketchPlacement::Fixed(plane) => Ok(plane),
            SketchPlacement::WorkPlane(id) => planes
                .ok_or(WorkPlaneError::Unavailable {
                    feature: id,
                    expected: "plane",
                })?
                .plane(id),
        }
    }
}
impl DocumentSketches {
    /// Attach without changing local geometry or retaining a frame; example: `sketches.attach_work_plane(sketch, feature)?`.
    pub fn attach_work_plane(
        &mut self,
        sketch: DocumentSketchId,
        feature: FeatureId,
    ) -> Result<(), SketchError> {
        require_work_plane_id(sketch, feature)?;
        let index = self.index(sketch)?;
        self.sketches[index].placement = SketchPlacement::WorkPlane(feature);
        Ok(())
    }
    /// Create directly on an authored reference; example: `sketches.add_on_work_plane(id, "Sketch", feature, geometry)?`.
    pub fn add_on_work_plane(
        &mut self,
        id: DocumentSketchId,
        name: &str,
        feature: FeatureId,
        geometry: Sketch2D,
    ) -> Result<(), SketchError> {
        require_work_plane_id(id, feature)?;
        self.insert_placed(id, name, SketchPlacement::WorkPlane(feature), geometry)
    }
}
fn require_work_plane_id(sketch: DocumentSketchId, feature: FeatureId) -> Result<(), SketchError> {
    if feature.0 == 0 {
        return Err(SketchError::Definition(format!(
            "sketch {sketch:?} work plane {feature:?}: expected a nonzero feature ID"
        )));
    }
    Ok(())
}
