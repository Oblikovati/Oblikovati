// SPDX-License-Identifier: GPL-2.0-only
mod input;
mod mesh;
mod profile;
mod report;
mod snapshot;
#[cfg(test)]
#[path = "../tests/unit/prism_snapshot.rs"]
mod snapshot_tests;
mod types;
pub use mesh::{PrismFaceMesh, PrismMeshOptions, PrismPresentation};
use rustycad_tkmath::gp::Vector3;
use rustycad_tkprim::brep_prim_api::BRepPrimAPIMakePrism;
use rustycad_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;
pub use snapshot::{PrismProbeSnapshot, retain_planar_prism};
pub use types::{
    PrismFaceProbe, PrismFaceRole, PrismProbeDefinition, PrismProbeLoop, PrismProbeReport,
};
/// Check the released planar profile/prism path and generated-face lineage in working units.
/// Example: `probe_planar_prism(&definition)?` reports analytic geometry without publishing a feature result.
pub fn probe_planar_prism(definition: &PrismProbeDefinition) -> Result<PrismProbeReport, String> {
    Ok(retain_planar_prism(definition)?.report().clone())
}
fn build_prism(definition: &PrismProbeDefinition) -> Result<PrismProbeSnapshot, String> {
    input::validate_definition(definition)?;
    let profile = profile::build_profile(definition)?;
    let [x, y, z] = definition.displacement;
    let mut maker = BRepPrimAPIMakePrism::new(&profile.face, Vector3::new(x, y, z), true, true)
        .map_err(|error| {
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
