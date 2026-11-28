// SPDX-License-Identifier: GPL-2.0-only
use super::{WorkGeometryDefinition, WorkGeometryError, point::resolve_position};
use crate::{
    feature::{ucs::ResolvedUcs, work_axis::WorkAxisLine},
    parameter::ParameterCollection,
    sketch::SketchPlane,
};

/// Kind views share one current result table, never separate owned feature collections.
#[derive(Clone, Copy, Debug)]
pub(super) enum WorkGeometryResult {
    Plane(SketchPlane),
    Axis(WorkAxisLine),
    Point([f64; 3]),
    Ucs(ResolvedUcs),
}
impl WorkGeometryResult {
    pub(super) fn with_plane_orientation(
        self,
        feature: &super::AuthoredFeature<WorkGeometryDefinition>,
    ) -> Result<Self, WorkGeometryError> {
        // The replay boundary validates plane-only metadata before evaluating the source (#3748).
        if let (Self::Plane(plane), true) = (self, feature.is_normal_flipped()) {
            return plane
                .flipped_normal()
                .map(Self::Plane)
                .map_err(WorkGeometryError::Frame);
        }
        Ok(self)
    }
    pub(super) fn evaluate(
        definition: WorkGeometryDefinition,
        parameters: &ParameterCollection<u64>,
        current: &impl crate::feature::work_plane::WorkPlaneResolver,
    ) -> Result<Self, WorkGeometryError> {
        match definition {
            WorkGeometryDefinition::Axis(axis) => Ok(Self::Axis(axis)),
            WorkGeometryDefinition::PointAtReference(reference) => {
                reference.position(current).map(Self::Point)
            }
            WorkGeometryDefinition::Point(coordinates) => {
                resolve_position(coordinates, parameters).map(Self::Point)
            }
            WorkGeometryDefinition::AxisThroughPoints(points) => axis_between(points, current),
            WorkGeometryDefinition::Ucs(definition) => Ok(Self::Ucs(definition.resolved())),
            plane => plane.evaluate(parameters, current).map(Self::Plane),
        }
    }
    pub(super) fn kind(self) -> &'static str {
        match self {
            Self::Plane(_) => "plane",
            Self::Axis(_) => "axis",
            Self::Point(_) => "point",
            Self::Ucs(_) => "ucs",
        }
    }
}

fn axis_between(
    points: super::WorkAxisPoints,
    current: &impl crate::feature::work_plane::WorkPlaneResolver,
) -> Result<WorkGeometryResult, WorkGeometryError> {
    WorkAxisLine::through_points(
        points.first.position(current)?,
        points.second.position(current)?,
    )
    .map(WorkGeometryResult::Axis)
    .map_err(WorkGeometryError::Axis)
}
