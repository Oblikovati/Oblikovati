// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    EvaluationEntity as Entity, EvaluationFailure, EvaluationFailureKind as Kind,
    EvaluationPhase as Phase,
};
mod certified;
pub(crate) use certified::{retain_certified_tapered_prism, retain_certified_tapered_prisms};
pub(crate) mod input;
pub use certified::{
    CertifiedPrismSnapshot, PrismAchievedTolerance, retain_certified_profile_prism,
    retain_certified_profile_prisms,
};
mod named;
pub use named::{
    AnalyticEdgeAxis, AnalyticEdgeAxisKind, BlendCandidate, BlendHistoryRecord, BlendNamedTopology,
    CapturedNamedSolid, CertifiedNamedSolidEvaluator, CertifiedPrismEvidence,
    DrawingBaseProjection, DrawingProjectedCurve, DrawingProjection, DrawingProjectionCurveKind,
    DrawingProjectionFrame, MarkEligibility, NamedEdgeHit, NamedFaceMesh, NamedPrismSnapshot,
    NamedSolid, NamedVertexHit, PrismFaceTopology, PrismSurfaceKind, RigidMotion,
    SharedFilletVertex, SolidFilletEdgeGroups, ThreadableFaceInfo, capture_full_named_candidate,
    check_mark_eligibility, complete_named_candidate, complete_named_checkpoints,
    drawing_base_view_bodies, drawing_base_view_solids, drawing_projection_bodies,
    drawing_projection_solids, finish_appearance_rgb_u8, finish_appearance_rgba,
    named_part_properties, named_solid_part_properties, prepare_dependent_named_publication,
    prepare_full_named_publication, retain_certified_named_sketch_prism, retain_named_sketch_prism,
    validate_full_named_snapshot,
};
mod mesh;
pub(crate) mod placement;
pub(crate) mod profile;
mod provenance;
pub use provenance::{
    PrismHistoryRelation, PrismHistorySource, PrismSubentityHistory, PrismSubentityProvenance,
    PrismSubentityRole,
};
pub(crate) mod report;
mod resolution;
pub use resolution::{PrismResolution, PrismResolutionPolicy};
mod snapshot;
#[cfg(test)]
#[path = "../tests/unit/prism_snapshot.rs"]
mod snapshot_tests;
mod taper;
mod types;
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakePrism;
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;
pub use mesh::{DisplayTriangle, PrismFaceMesh, PrismMeshOptions, PrismPresentation};
use oblikovati_model::sketch::SketchPlane;
pub use snapshot::{PrismProbeSnapshot, retain_planar_prism, retain_prism_on_plane};
pub(crate) use taper::PrismTaper;
pub use types::{
    FaceProbe, PlanarProfileLoop, PrismFaceProbe, PrismFaceRole, PrismProbeDefinition,
    PrismProbeLoop, PrismProbeReport, PrismTopologySummary, TopologySummary,
};
/// Check the released planar profile/prism path and generated-face lineage in working units.
/// Example: `probe_planar_prism(&definition)?` reports analytic geometry without publishing a feature result.
pub fn probe_planar_prism(definition: &PrismProbeDefinition) -> Result<PrismProbeReport, String> {
    Ok(retain_planar_prism(definition)?.report().clone())
}
/// Build the common retained prism and preserve each failure stage; example: `build_prism(definition, plane, 0.)?`.
pub(crate) fn build_prism(
    definition: &PrismProbeDefinition,
    plane: SketchPlane,
    taper_radians: f64,
) -> Result<PrismProbeSnapshot, EvaluationFailure> {
    input::validate_definition(definition)?;
    let vector = placement::model_displacement(plane, definition.displacement)?;
    let profile = profile::build_profile(&definition.outer, &definition.holes, plane)?;
    let mut maker = construct_prism(&profile.face, vector, definition.displacement)?;
    let shape = completed_prism(&mut maker)?;
    crate::checked_shape::require_geometry(&shape, "prism result")?;
    let lineage = report::construction_lineage(&mut maker, &profile, &shape)?;
    if let Some(taper) = PrismTaper::new(plane, vector, taper_radians) {
        let tapered = taper.apply(&shape, &lineage, &[])?;
        return snapshot::capture_prism(&tapered.shape, &tapered.faces);
    }
    snapshot::capture_prism(&shape, &lineage)
}

fn construct_prism(
    face: &martensite_tkbrep::topods::Face,
    vector: martensite_tkmath::gp::Vector3,
    displacement: [f64; 3],
) -> Result<BRepPrimAPIMakePrism, EvaluationFailure> {
    BRepPrimAPIMakePrism::new(face, vector, true, true).map_err(|cause| {
        Phase::Prism.native(
            Entity::Body,
            format!("prism displacement {displacement:?}: expected constructed solid"),
            cause,
        )
    })
}
fn completed_prism(
    maker: &mut BRepPrimAPIMakePrism,
) -> Result<martensite_tkbrep::topods::Shape, EvaluationFailure> {
    maker.shape().cloned().map_err(|cause| {
        Phase::Prism.native(
            Entity::Body,
            "prism result: expected completed shape".into(),
            cause,
        )
    })
}

/// A derived display mesh source; exact reference queries remain on the guarded owner.
pub trait PrismPresentationSource {
    /// Discretize retained geometry for display; example: `body.presentation(options)?`.
    fn presentation(&self, options: PrismMeshOptions) -> Result<PrismPresentation, String>;
}

/// Display-only triangles for a current solid body.
pub trait BodyPresentationSource {
    /// Tessellate the body for display; example: `body.body_triangles(options)?`.
    fn body_triangles(&self, options: PrismMeshOptions) -> Result<Vec<[[f64; 3]; 3]>, String> {
        Ok(self
            .body_face_triangles(options)?
            .into_iter()
            .flatten()
            .collect())
    }
    /// Tessellate a body in face order; example: `body.body_face_triangles(options)?`.
    fn body_face_triangles(
        &self,
        options: PrismMeshOptions,
    ) -> Result<Vec<Vec<[[f64; 3]; 3]>>, String>;
    /// Tessellate a body in face order with analytical per-vertex normals when available.
    fn body_face_display_triangles(
        &self,
        options: PrismMeshOptions,
    ) -> Result<Vec<Vec<DisplayTriangle>>, String> {
        Ok(self
            .body_face_triangles(options)?
            .into_iter()
            .map(|face| {
                face.into_iter()
                    .map(|positions| DisplayTriangle {
                        positions,
                        normals: None,
                    })
                    .collect()
            })
            .collect())
    }
}
impl<T: PrismPresentationSource> BodyPresentationSource for T {
    fn body_face_triangles(
        &self,
        options: PrismMeshOptions,
    ) -> Result<Vec<Vec<[[f64; 3]; 3]>>, String> {
        Ok(self
            .presentation(options)?
            .faces
            .into_iter()
            .map(|face| face.triangles)
            .collect())
    }
    fn body_face_display_triangles(
        &self,
        options: PrismMeshOptions,
    ) -> Result<Vec<Vec<DisplayTriangle>>, String> {
        Ok(self
            .presentation(options)?
            .faces
            .into_iter()
            .map(|face| {
                let has_normals = face.normals.len() == face.triangles.len();
                face.triangles
                    .into_iter()
                    .enumerate()
                    .map(|(index, positions)| DisplayTriangle {
                        positions,
                        normals: has_normals.then(|| face.normals[index]),
                    })
                    .collect()
            })
            .collect())
    }
}
impl PrismPresentationSource for PrismProbeSnapshot {
    fn presentation(&self, options: PrismMeshOptions) -> Result<PrismPresentation, String> {
        PrismProbeSnapshot::presentation(self, options)
    }
}

#[cfg(test)]
#[path = "../tests/unit/fillet_prism_history_probe.rs"]
mod fillet_prism_history_probe;
#[cfg(test)]
#[path = "../tests/unit/hole_prism_history_probe.rs"]
mod hole_prism_history_probe;
