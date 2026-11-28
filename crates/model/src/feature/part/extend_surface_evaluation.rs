// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Part Surface Extend features during ordered part replay (#3918).
use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::resolve_body_slot,
};
use crate::feature::{
    FeatureId,
    extend_surface::{
        ExtendSurfaceDefinition, ExtendSurfaceExtent, ExtendSurfaceTermination,
        ResolvedExtendSurface, ResolvedExtendSurfaceExtent,
    },
    fillet::reference_identity,
    work_plane::WorkPlaneResolver,
};

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_extend_surface<
        B: PartFeatureEvaluator<Body = Body, Failure = Failure>,
    >(
        &mut self,
        backend: &mut B,
        feature: FeatureId,
        definition: &ExtendSurfaceDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let target_slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            definition.body_source(),
            definition.target_body(),
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        let resolved_extent = self.resolve_extend_surface_extent(backend, definition)?;
        let target = self.bodies.get(&target_slot).ok_or_else(|| {
            PartFeatureFailure::BodyReference(format!(
                "extend surface target slot {target_slot:?}: expected an active surface body"
            ))
        })?;
        let input = ResolvedExtendSurface {
            edges: definition.edges().to_vec(),
            edge_chain: definition.edge_chain(),
            extent: resolved_extent,
            method: definition.method(),
        };
        let result = backend.evaluate_extend_surface(feature, target, input)?;
        backend
            .verify_body_reference(&result, definition.target_body())
            .map_err(PartFeatureFailure::BodyReference)?;
        self.body_lineage
            .replace(target_slot, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&target_slot);
        self.bodies.insert(feature.into(), result);
        Ok(())
    }

    fn resolve_extend_surface_extent<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &self,
        backend: &B,
        definition: &ExtendSurfaceDefinition,
    ) -> Result<ResolvedExtendSurfaceExtent, PartFeatureFailure<Failure>> {
        match definition.extent() {
            ExtendSurfaceExtent::Distance(distance) => {
                Ok(ResolvedExtendSurfaceExtent::Distance(*distance))
            }
            ExtendSurfaceExtent::To(ExtendSurfaceTermination::OriginPlane(origin)) => {
                Ok(ResolvedExtendSurfaceExtent::ToPlane(origin.sketch_plane()))
            }
            ExtendSurfaceExtent::To(ExtendSurfaceTermination::WorkPlane(plane_id)) => {
                let plane = self.datums.plane(*plane_id).map_err(|cause| {
                    PartFeatureFailure::BodyReference(format!(
                        "extend surface termination plane {plane_id:?}: {cause}"
                    ))
                })?;
                Ok(ResolvedExtendSurfaceExtent::ToPlane(plane))
            }
            ExtendSurfaceExtent::To(ExtendSurfaceTermination::Face { body, face }) => {
                let (_, source) = reference_identity(body, 4, "extend surface termination body")
                    .map_err(PartFeatureFailure::BodyReference)?;
                let slot =
                    resolve_body_slot(&self.bodies, &self.body_lineage, backend, source, body)
                        .map_err(PartFeatureFailure::BodyReference)?;
                let term_body = self.bodies.get(&slot).ok_or_else(|| {
                    PartFeatureFailure::BodyReference(format!(
                        "extend surface termination slot {slot:?}: expected an active body"
                    ))
                })?;
                let plane = backend
                    .resolve_planar_face(term_body, face)
                    .map_err(PartFeatureFailure::BodyReference)?;
                Ok(ResolvedExtendSurfaceExtent::ToPlane(plane))
            }
        }
    }
}
