// SPDX-License-Identifier: GPL-2.0-only
//! Resolve evaluated construction geometry through the part result.
use super::PartFeatureEvaluation;
use crate::feature::{FeatureId, work_geometry::WorkGeometryError, work_plane::WorkPlaneResolver};

impl<Body, Failure> WorkPlaneResolver for PartFeatureEvaluation<Body, Failure> {
    fn axis(
        &self,
        feature: FeatureId,
    ) -> Result<crate::feature::work_axis::WorkAxisLine, WorkGeometryError> {
        self.datums.axis(feature)
    }
    fn plane(&self, feature: FeatureId) -> Result<crate::sketch::SketchPlane, WorkGeometryError> {
        WorkPlaneResolver::plane(&self.datums, feature)
    }
    fn face(
        &self,
        reference: &crate::feature::work_plane::WorkPlaneFaceReference,
    ) -> Result<crate::sketch::SketchPlane, WorkGeometryError> {
        WorkPlaneResolver::face(&self.datums, reference)
    }
    fn point(&self, feature: FeatureId) -> Result<[f64; 3], WorkGeometryError> {
        self.datums.point(feature)
    }
}
