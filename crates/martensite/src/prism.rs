// SPDX-License-Identifier: GPL-2.0-only
mod input;
mod named;
pub use named::{NamedPrismEvaluator, NamedPrismSnapshot, retain_named_sketch_prism};
mod mesh;
mod placement;
mod profile;
mod report;
mod snapshot;
#[cfg(test)]
#[path = "../tests/unit/prism_snapshot.rs"]
mod snapshot_tests;
mod types;
pub use mesh::{PrismFaceMesh, PrismMeshOptions, PrismPresentation};
use oblikovati_model::sketch::SketchPlane;
use rustycad_tkprim::brep_prim_api::BRepPrimAPIMakePrism;
use rustycad_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;
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
fn build_prism(
    definition: &PrismProbeDefinition,
    plane: SketchPlane,
) -> Result<PrismProbeSnapshot, String> {
    input::validate_definition(definition)?;
    let vector = placement::model_displacement(plane, definition.displacement)?;
    let profile = profile::build_profile(definition, plane)?;
    let mut maker =
        BRepPrimAPIMakePrism::new(&profile.face, vector, true, true).map_err(|error| {
            format!(
                "prism displacement {:?}: expected constructed solid: {error}",
                definition.displacement
            )
        })?;
    let shape = maker
        .shape()
        .map_err(|error| format!("prism result: expected completed shape: {error}"))?
        .clone();
    crate::require_geometric_shape(&shape, "prism result")?;
    let lineage = report::construction_lineage(&mut maker, &profile, &shape)?;
    snapshot::capture_prism(&shape, &lineage)
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
