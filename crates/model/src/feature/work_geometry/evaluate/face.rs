// SPDX-License-Identifier: GPL-2.0-only
//! Combine current datum results with the owning part's named-face resolver.
use super::*;
use crate::feature::work_plane::{WorkPlaneFaceReference, WorkPlaneResolver};

pub(crate) struct DatumFaceResolver<'a, F> {
    pub datums: &'a WorkGeometryEvaluation,
    pub resolve_face: &'a F,
}
impl<F> WorkPlaneResolver for DatumFaceResolver<'_, F>
where
    F: Fn(&WorkPlaneFaceReference) -> Result<crate::sketch::SketchPlane, WorkPlaneError>,
{
    fn axis(
        &self,
        id: FeatureId,
    ) -> Result<crate::feature::work_axis::WorkAxisLine, WorkPlaneError> {
        self.datums.axis(id)
    }
    fn plane(&self, id: FeatureId) -> Result<crate::sketch::SketchPlane, WorkPlaneError> {
        WorkPlaneResolver::plane(self.datums, id)
    }
    fn point(&self, id: FeatureId) -> Result<[f64; 3], WorkPlaneError> {
        self.datums.point(id)
    }
    fn face(
        &self,
        reference: &WorkPlaneFaceReference,
    ) -> Result<crate::sketch::SketchPlane, WorkPlaneError> {
        (self.resolve_face)(reference)
    }
}
