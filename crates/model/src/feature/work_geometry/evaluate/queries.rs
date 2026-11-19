// SPDX-License-Identifier: GPL-2.0-only
use super::{WorkGeometryEvaluation, WorkPlaneError};
use crate::feature::work_geometry::{WorkGeometryKindMismatch, result::WorkGeometryResult};
use crate::feature::{FeatureId, work_axis::WorkAxisLine, work_plane::WorkPlaneResolver};
use crate::sketch::SketchPlane;

impl WorkPlaneResolver for WorkGeometryEvaluation {
    fn axis(&self, feature: FeatureId) -> Result<WorkAxisLine, WorkPlaneError> {
        WorkGeometryEvaluation::axis(self, feature)
    }
    fn point(&self, feature: FeatureId) -> Result<[f64; 3], WorkPlaneError> {
        WorkGeometryEvaluation::point(self, feature)
    }
    fn plane(&self, feature: FeatureId) -> Result<SketchPlane, WorkPlaneError> {
        self.resolve_kind(feature, "plane", project_plane)
    }
    fn face(
        &self,
        reference: &crate::feature::work_plane::WorkPlaneFaceReference,
    ) -> Result<SketchPlane, WorkPlaneError> {
        self.sketch_faces
            .get(reference.key())
            .cloned()
            .unwrap_or_else(|| Err(crate::feature::work_plane::unavailable_face(reference)))
    }
}
impl WorkGeometryEvaluation {
    /// Resolve a current exact frame-origin reference; example: `evaluation.sketch_frame_origin(&reference)?`.
    pub fn sketch_frame_origin(
        &self,
        reference: &crate::document_sketch::SketchFrameTopologyOrigin,
    ) -> Result<[f64; 3], WorkPlaneError> {
        self.sketch_frame_origins
            .get(reference.key())
            .cloned()
            .unwrap_or_else(|| {
                Err(WorkPlaneError::BodyReference(format!(
                    "frame origin {:?}: expected current named topology",
                    reference.key()
                )))
            })
    }
    /// Iterate successful current planes by identity; example: `evaluation.planes().count()`.
    pub fn planes(&self) -> impl Iterator<Item = (FeatureId, SketchPlane)> + '_ {
        self.datums
            .iter()
            .filter_map(|(&feature, &datum)| project_plane(datum).map(|plane| (feature, plane)))
    }
    /// Resolve one current line by stable identity; example: `evaluation.axis(FeatureId(3))?`.
    pub fn axis(&self, feature: FeatureId) -> Result<WorkAxisLine, WorkPlaneError> {
        self.resolve_kind(feature, "axis", project_axis)
    }
    /// Resolve one current position by stable identity; example: `evaluation.point(FeatureId(1))?`.
    pub fn point(&self, feature: FeatureId) -> Result<[f64; 3], WorkPlaneError> {
        self.resolve_kind(feature, "point", project_point)
    }
    fn resolve_kind<T>(
        &self,
        feature: FeatureId,
        expected: &'static str,
        project: fn(WorkGeometryResult) -> Option<T>,
    ) -> Result<T, WorkPlaneError> {
        let datum = self
            .datums
            .get(&feature)
            .copied()
            .ok_or(WorkPlaneError::Unavailable { feature, expected })?;
        project(datum).ok_or(WorkPlaneError::Kind(WorkGeometryKindMismatch {
            feature,
            expected,
            actual: datum.kind(),
        }))
    }
}
fn project_plane(datum: WorkGeometryResult) -> Option<SketchPlane> {
    match datum {
        WorkGeometryResult::Plane(plane) => Some(plane),
        _ => None,
    }
}
fn project_axis(datum: WorkGeometryResult) -> Option<WorkAxisLine> {
    match datum {
        WorkGeometryResult::Axis(axis) => Some(axis),
        _ => None,
    }
}
fn project_point(datum: WorkGeometryResult) -> Option<[f64; 3]> {
    match datum {
        WorkGeometryResult::Point(point) => Some(point),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn planes_include_only_evaluated_plane_results_in_identity_order() {
        let mut result = WorkGeometryEvaluation::default();
        result
            .datums
            .insert(FeatureId(9), WorkGeometryResult::Plane(SketchPlane::xz()));
        result
            .datums
            .insert(FeatureId(3), WorkGeometryResult::Point([0.; 3]));
        result
            .datums
            .insert(FeatureId(2), WorkGeometryResult::Plane(SketchPlane::xy()));
        let planes: Vec<_> = result.planes().collect();
        assert_eq!(
            planes,
            vec![
                (FeatureId(2), SketchPlane::xy()),
                (FeatureId(9), SketchPlane::xz())
            ]
        );
    }
}
