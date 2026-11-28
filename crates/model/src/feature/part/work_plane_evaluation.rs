// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate topology-aware extended Work Plane features during ordered Part replay (#3918).

use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_lineage::BodyLineageIndex, body_resolution::resolve_body_face,
};
use crate::{
    feature::{
        FeatureId,
        work_axis::WorkAxisLine,
        work_geometry::{WorkGeometryError, WorkGeometryEvaluation},
        work_plane::{
            WorkPlaneExtendedDefinition, WorkPlaneExtendedResolver, WorkPlaneFaceReference,
            WorkPlaneResolver, WorkPlaneSurfaceReference, WorkPlaneTopologyRef,
        },
    },
    history::DocumentSnapshot,
    sketch::SketchPlane,
};
use std::collections::BTreeMap;

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_work_plane_extended<
        B: PartFeatureEvaluator<Body = Body, Failure = Failure>,
    >(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &B,
        id: FeatureId,
        definition: &WorkPlaneExtendedDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let feature = snapshot.work_geometry.get(id).map_err(|_| {
            PartFeatureFailure::Datum(Box::new(WorkGeometryError::Unavailable {
                feature: id,
                expected: "work plane",
            }))
        })?;
        feature
            .validate_plane_metadata()
            .map_err(|cause| PartFeatureFailure::Metadata(Box::new(cause)))?;
        let resolver = PartWorkPlaneResolver {
            datums: &self.datums,
            bodies: &self.bodies,
            lineage: &self.body_lineage,
            backend,
        };
        let plane = definition
            .evaluate(&snapshot.parameters, &resolver)
            .map_err(|cause| PartFeatureFailure::Datum(Box::new(cause)))?;
        self.datums
            .insert_evaluated_plane(id, feature, plane)
            .map_err(|cause| PartFeatureFailure::Datum(Box::new(cause)))
    }
}

struct PartWorkPlaneResolver<'a, B: PartFeatureEvaluator> {
    datums: &'a WorkGeometryEvaluation,
    bodies: &'a BTreeMap<super::body_lineage::BodySlot, B::Body>,
    lineage: &'a BodyLineageIndex,
    backend: &'a B,
}

impl<B: PartFeatureEvaluator> WorkPlaneResolver for PartWorkPlaneResolver<'_, B> {
    fn axis(&self, feature: FeatureId) -> Result<WorkAxisLine, WorkGeometryError> {
        self.datums.axis(feature)
    }
    fn plane(&self, feature: FeatureId) -> Result<SketchPlane, WorkGeometryError> {
        WorkPlaneResolver::plane(self.datums, feature)
    }
    fn face(&self, reference: &WorkPlaneFaceReference) -> Result<SketchPlane, WorkGeometryError> {
        resolve_body_face(self.bodies, self.lineage, self.backend, reference)
    }
    fn point(&self, feature: FeatureId) -> Result<[f64; 3], WorkGeometryError> {
        self.datums.point(feature)
    }
}

impl<B: PartFeatureEvaluator> WorkPlaneExtendedResolver for PartWorkPlaneResolver<'_, B> {
    fn topology_vertex(
        &self,
        reference: &WorkPlaneTopologyRef,
    ) -> Result<[f64; 3], WorkGeometryError> {
        resolve_body_entity(self, reference.source(), reference.key(), |body| {
            self.backend.resolve_topology_point(body, reference.key())
        })
    }

    fn topology_edge(
        &self,
        reference: &WorkPlaneTopologyRef,
    ) -> Result<WorkAxisLine, WorkGeometryError> {
        resolve_body_entity(self, reference.source(), reference.key(), |body| {
            self.backend.resolve_straight_edge(body, reference.key())
        })
    }

    fn torus_midplane(
        &self,
        surface: &WorkPlaneSurfaceReference,
    ) -> Result<SketchPlane, WorkGeometryError> {
        resolve_body_entity(self, surface.face.source(), surface.face.key(), |body| {
            self.backend
                .resolve_torus_midplane(body, surface.face.key())
        })
    }

    fn tangent_through_edge(
        &self,
        surface: &WorkPlaneSurfaceReference,
        line: WorkAxisLine,
    ) -> Result<SketchPlane, WorkGeometryError> {
        resolve_body_entity(self, surface.face.source(), surface.face.key(), |body| {
            self.backend
                .resolve_tangent_plane_through_line(body, surface.face.key(), line)
        })
    }

    fn tangent_through_point(
        &self,
        surface: &WorkPlaneSurfaceReference,
        point: [f64; 3],
    ) -> Result<SketchPlane, WorkGeometryError> {
        resolve_body_entity(self, surface.face.source(), surface.face.key(), |body| {
            self.backend
                .resolve_tangent_plane_through_point(body, surface.face.key(), point)
        })
    }

    fn tangent_parallel_to_plane(
        &self,
        surface: &WorkPlaneSurfaceReference,
        plane: SketchPlane,
    ) -> Result<SketchPlane, WorkGeometryError> {
        resolve_body_entity(self, surface.face.source(), surface.face.key(), |body| {
            self.backend
                .resolve_tangent_plane_parallel(body, surface.face.key(), plane)
        })
    }
}

fn resolve_body_entity<B: PartFeatureEvaluator, T>(
    resolver: &PartWorkPlaneResolver<'_, B>,
    source: FeatureId,
    key: &[u8],
    query: impl Fn(&B::Body) -> Result<T, String>,
) -> Result<T, WorkGeometryError> {
    super::body_resolution::resolve_named_entity(
        resolver.bodies,
        resolver.lineage,
        source,
        key,
        query,
    )
    .map_err(WorkGeometryError::BodyReference)
}
