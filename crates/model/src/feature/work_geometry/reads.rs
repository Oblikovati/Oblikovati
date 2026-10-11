// SPDX-License-Identifier: GPL-2.0-only
use super::{DatumLength, WorkGeometryDefinition, WorkPointReference};
use crate::feature::work_plane::WorkPlaneReference;
use crate::feature::{FeatureReadFootprint, FeatureReadKey};

impl WorkGeometryDefinition {
    /// Attribute every authored datum input; example: `definition.reads()` before program insertion.
    pub fn reads(self) -> FeatureReadFootprint {
        let keys = match self {
            Self::Offset { base, distance } => offset_keys(base, distance),
            Self::ParallelThroughPoint(source) => parallel_keys(source),
            Self::LineThroughPoint(source) | Self::NormalToAxis(source) => line_point_keys(source),
            Self::ThroughTwoAxes(axes) => axes.into_iter().filter_map(axis_key).collect(),
            Self::ThroughThreePoints(points) => points.into_iter().filter_map(point_key).collect(),
            Self::PointAtReference(reference) => point_key(reference).into_iter().collect(),
            Self::Point(coordinates) => coordinates.into_iter().filter_map(length_key).collect(),
            Self::AxisThroughPoints(points) => [points.first, points.second]
                .into_iter()
                .filter_map(point_key)
                .collect(),
            _ => Vec::new(),
        };
        FeatureReadFootprint::new(keys, false)
    }
}
fn length_key(length: DatumLength) -> Option<FeatureReadKey> {
    length.parameter().map(FeatureReadKey::Parameter)
}
fn point_key(reference: WorkPointReference) -> Option<FeatureReadKey> {
    match reference {
        WorkPointReference::Origin => None,
        WorkPointReference::Feature(id) => Some(FeatureReadKey::Feature(id)),
    }
}
fn plane_key(base: WorkPlaneReference) -> Option<FeatureReadKey> {
    match base {
        WorkPlaneReference::Origin(_) => None,
        WorkPlaneReference::Feature(id) => Some(FeatureReadKey::Feature(id)),
    }
}
fn offset_keys(base: WorkPlaneReference, distance: DatumLength) -> Vec<FeatureReadKey> {
    plane_key(base)
        .into_iter()
        .chain(length_key(distance))
        .collect()
}
fn parallel_keys(source: super::WorkPlaneThroughPoint) -> Vec<FeatureReadKey> {
    plane_key(source.base)
        .into_iter()
        .chain(point_key(source.point))
        .collect()
}

fn line_point_keys(source: super::WorkPlaneLinePoint) -> Vec<FeatureReadKey> {
    axis_key(source.line)
        .into_iter()
        .chain(point_key(source.point))
        .collect()
}
fn axis_key(reference: crate::feature::work_axis::WorkAxisReference) -> Option<FeatureReadKey> {
    match reference {
        crate::feature::work_axis::WorkAxisReference::Feature(id) => {
            Some(FeatureReadKey::Feature(id))
        }
        _ => None,
    }
}
