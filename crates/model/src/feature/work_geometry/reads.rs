// SPDX-License-Identifier: GPL-2.0-only
use super::{DatumLength, WorkGeometryDefinition, WorkPointReference};
use crate::feature::work_plane::WorkPlaneReference;
use crate::feature::{FeatureReadFootprint, FeatureReadKey};

impl WorkGeometryDefinition {
    /// Attribute every authored datum input; example: `definition.reads()` before program insertion.
    pub fn reads(self) -> FeatureReadFootprint {
        let keys = match self {
            Self::Offset { base, distance } => offset_keys(base, distance),
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
fn offset_keys(base: WorkPlaneReference, distance: DatumLength) -> Vec<FeatureReadKey> {
    let mut keys = Vec::new();
    if let WorkPlaneReference::Feature(id) = base {
        keys.push(FeatureReadKey::Feature(id));
    }
    keys.extend(length_key(distance));
    keys
}
