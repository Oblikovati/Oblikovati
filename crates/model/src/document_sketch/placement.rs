// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches, PlacedSketch, PlacedSketchRecipe};
use crate::{
    feature::{
        FeatureId, FeatureReadFootprint, FeatureReadKey,
        work_plane::{WorkPlaneError, WorkPlaneFaceReference, WorkPlaneOrigin, WorkPlaneResolver},
    },
    sketch::{Sketch2D, SketchError, SketchPlane},
};

impl DocumentSketches {
    /// Report inputs that can require ordered recompute after a part feature changes.
    /// Example: `sketches.requires_part_dependency_recompute()` before reusing a part result.
    pub fn requires_part_dependency_recompute(&self) -> bool {
        self.sketches.iter().any(|sketch| {
            !matches!(
                sketch.placement,
                SketchPlacement::Fixed(_) | SketchPlacement::OriginPlane(_)
            ) || sketch.orientation.is_some()
                || sketch
                    .frame_edit
                    .as_ref()
                    .is_some_and(|edit| edit.requires_datums())
                || !sketch.point_projections.is_empty()
        })
    }
}

/// Authored placement; a feature reference stores no derived frame.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchPlacement {
    OriginPlane(WorkPlaneOrigin),
    Fixed(SketchPlane),
    WorkPlane(FeatureId),
    Face(WorkPlaneFaceReference),
}
impl SketchPlacement {
    pub(super) fn same_recipe(&self, other: &Self) -> bool {
        match (self, other) {
            (Self::OriginPlane(first), Self::OriginPlane(second)) => first == second,
            (Self::Fixed(first), Self::Fixed(second)) => first.same_recipe(*second),
            (Self::WorkPlane(first), Self::WorkPlane(second)) => first == second,
            (Self::Face(first), Self::Face(second)) => first == second,
            _ => false,
        }
    }
    pub(super) fn from_recipe(
        recipe: &PlacedSketchRecipe,
        version: u32,
    ) -> Result<Self, SketchError> {
        match (
            recipe.plane,
            recipe.plane_options,
            recipe.plane_feature,
            &recipe.plane_face,
            recipe.plane_origin,
        ) {
            (Some(frame), Some(options), None, None, None) => {
                SketchPlane::from_recipe(frame, options).map(Self::Fixed)
            }
            (None, None, Some(feature), None, None) if version >= 2 && feature.0 != 0 => {
                Ok(Self::WorkPlane(feature))
            }
            (None, None, None, Some(reference), None) if version >= 12 => {
                Ok(Self::Face(reference.clone()))
            }
            (None, None, None, None, Some(origin)) if version >= 13 => {
                Ok(Self::OriginPlane(origin))
            }
            shape => Err(SketchError::Definition(format!(
                "sketch {:?} placement {shape:?} in recipe version {version}: expected one fixed frame, work-plane feature, or version 12 face or version 13 origin-plane reference",
                recipe.id
            ))),
        }
    }
}
impl PlacedSketch {
    /// Inspect the authored reference; example: `placed.placement()` for dependency queries.
    pub fn placement(&self) -> SketchPlacement {
        self.placement.clone()
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
        keys.extend(
            self.point_projections
                .iter()
                .filter_map(|binding| binding.read_key()),
        );
        FeatureReadFootprint::new(keys, false)
    }
    pub(crate) fn placement_read_keys(&self) -> Vec<FeatureReadKey> {
        let mut keys = Vec::new();
        match &self.placement {
            SketchPlacement::WorkPlane(feature) => keys.push(FeatureReadKey::Feature(*feature)),
            SketchPlacement::Face(reference) => {
                keys.push(FeatureReadKey::Feature(reference.source()))
            }
            SketchPlacement::Fixed(_) | SketchPlacement::OriginPlane(_) => {}
        }
        if let Some(orientation) = self.orientation {
            keys.extend(orientation.read_keys());
        }
        if let Some(edit) = &self.frame_edit {
            keys.extend(edit.read_keys());
        }
        keys
    }
    /// Resolve with caller-supplied current planes; the caller must verify their source freshness.
    /// Example: `placed.plane(Some(&current_planes))?`.
    pub fn plane(
        &self,
        planes: Option<&dyn WorkPlaneResolver>,
    ) -> Result<SketchPlane, WorkPlaneError> {
        self.plane_with_axes(planes, &super::orientation::UnavailableSketchAxes)
    }
    /// Resolve using caller-supplied current resolvers; callers must ensure their data is fresh for this sketch.
    /// Example: `placed.plane_with_axes(Some(&planes), &axes)?`.
    pub fn plane_with_axes(
        &self,
        planes: Option<&dyn WorkPlaneResolver>,
        axes: &dyn super::SketchAxisResolver,
    ) -> Result<SketchPlane, WorkPlaneError> {
        let base = self.base_plane(planes)?;
        let frame = super::orientation::orient_current(base, self.orientation, axes)
            .map_err(WorkPlaneError::Frame)?;
        match &self.frame_edit {
            Some(edit) => edit.resolve(frame, axes).map_err(WorkPlaneError::Frame),
            None => Ok(frame),
        }
    }
    fn base_plane(
        &self,
        planes: Option<&dyn WorkPlaneResolver>,
    ) -> Result<SketchPlane, WorkPlaneError> {
        match &self.placement {
            SketchPlacement::Fixed(plane) => Ok(*plane),
            SketchPlacement::OriginPlane(origin) => Ok(origin.plane()),
            SketchPlacement::WorkPlane(id) => planes
                .ok_or(WorkPlaneError::Unavailable {
                    feature: *id,
                    expected: "plane",
                })?
                .plane(*id),
            SketchPlacement::Face(reference) => planes
                .ok_or_else(|| crate::feature::work_plane::unavailable_face(reference))?
                .face(reference),
        }
    }
}
impl DocumentSketches {
    /// Create a sketch attached to a durable face name; example: `sketches.add_on_face(id, "Sketch", face, geometry)?`.
    pub fn add_on_face(
        &mut self,
        id: DocumentSketchId,
        name: &str,
        face: WorkPlaneFaceReference,
        geometry: Sketch2D,
    ) -> Result<(), SketchError> {
        self.insert_placed(id, name, SketchPlacement::Face(face), geometry)
    }
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
    /// Create on an origin plane by stable identity; example: `sketches.add_on_origin_plane(id, "Sketch", origin, geometry)?`.
    pub fn add_on_origin_plane(
        &mut self,
        id: DocumentSketchId,
        name: &str,
        origin: WorkPlaneOrigin,
        geometry: Sketch2D,
    ) -> Result<(), SketchError> {
        self.insert_placed(id, name, SketchPlacement::OriginPlane(origin), geometry)
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
