// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    EvaluationEntity as Entity, EvaluationFailure, EvaluationFailureKind as Kind,
    EvaluationPhase as Phase,
};
pub(crate) mod input;
mod named;
pub use named::{
    NamedPrismEvaluator, NamedPrismSnapshot, named_part_properties, retain_named_sketch_prism,
};
mod mesh;
mod placement;
pub(crate) mod profile;
mod report;
mod snapshot;
#[cfg(test)]
#[path = "../tests/unit/prism_snapshot.rs"]
mod snapshot_tests;
mod types;
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakePrism;
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;
pub use mesh::{PrismFaceMesh, PrismMeshOptions, PrismPresentation};
use oblikovati_model::sketch::SketchPlane;
pub use snapshot::{PrismProbeSnapshot, retain_planar_prism, retain_prism_on_plane};
pub use types::{
    PrismFaceProbe, PrismFaceRole, PrismProbeDefinition, PrismProbeLoop, PrismProbeReport,
    PrismTopologySummary,
};
/// Check the released planar profile/prism path and generated-face lineage in working units.
/// Example: `probe_planar_prism(&definition)?` reports analytic geometry without publishing a feature result.
pub fn probe_planar_prism(definition: &PrismProbeDefinition) -> Result<PrismProbeReport, String> {
    Ok(retain_planar_prism(definition)?.report().clone())
}
/// Build the common retained prism and preserve each failure stage; example: `build_prism(definition, plane)?`.
pub(crate) fn build_prism(
    definition: &PrismProbeDefinition,
    plane: SketchPlane,
) -> Result<PrismProbeSnapshot, EvaluationFailure> {
    input::validate_definition(definition)?;
    let vector = placement::model_displacement(plane, definition.displacement)?;
    let profile = profile::build_profile(&definition.outer, &definition.holes, plane)?;
    let mut maker = construct_prism(&profile.face, vector, definition.displacement)?;
    let shape = completed_prism(&mut maker)?;
    crate::checked_shape::require_geometry(&shape, "prism result")?;
    let lineage = report::construction_lineage(&mut maker, &profile, &shape)?;
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
impl PrismPresentationSource for PrismProbeSnapshot {
    fn presentation(&self, options: PrismMeshOptions) -> Result<PrismPresentation, String> {
        PrismProbeSnapshot::presentation(self, options)
    }
}
