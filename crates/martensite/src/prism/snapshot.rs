// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use rustycad_tkbrep::topods::Shape;
use std::{marker::PhantomData, rc::Rc};
/// Retained diagnostic prism with immutable native geometry and copied construction roles.
/// ```compile_fail
/// fn require_send<T: Send>() {}
/// require_send::<oblikovati_martensite::PrismProbeSnapshot>();
/// ```
pub struct PrismProbeSnapshot {
    pub(super) shape: Shape,
    pub(super) faces: Vec<(PrismFaceRole, Shape)>,
    report: PrismProbeReport,
    owner: PhantomData<Rc<()>>,
}
/// Construct and retain the same checked prism path; example: `retain_planar_prism(&definition)?`.
pub fn retain_planar_prism(
    definition: &PrismProbeDefinition,
) -> Result<PrismProbeSnapshot, String> {
    retain_prism_on_plane(definition, SketchPlane::xy())
}
/// Retain a prism on a validated model plane; displacement components follow its X, Y and normal axes.
/// Example: `retain_prism_on_plane(&definition, SketchPlane::xz())?`.
pub fn retain_prism_on_plane(
    definition: &PrismProbeDefinition,
    plane: SketchPlane,
) -> Result<PrismProbeSnapshot, String> {
    build_prism(definition, plane)
}
pub(super) fn copied_prism(
    shape: &Shape,
    lineage: &[(PrismFaceRole, Shape)],
) -> Result<(Shape, Vec<(PrismFaceRole, Shape)>), String> {
    let mut copier = crate::snapshot::copied_kernel_shape(shape, "prism copy")?;
    let copied = copier.shape().map_err(|error| error.to_string())?.clone();
    let mut faces = Vec::new();
    for (role, source) in lineage {
        let mapped = copier
            .modified(source)
            .map_err(|error| format!("prism {role:?}: expected copy history: {error}"))?;
        if mapped.len() != 1 {
            return Err(format!(
                "prism {role:?} copied {} shapes: expected one preserved role",
                mapped.len()
            ));
        }
        faces.push((*role, mapped.first().expect("one copy").clone()));
    }
    crate::require_geometric_shape(&copied, "copied prism")?;
    report::checked_report(&copied, &faces)?;
    Ok((copied, faces))
}
pub(super) fn capture_prism(
    shape: &Shape,
    lineage: &[(PrismFaceRole, Shape)],
) -> Result<PrismProbeSnapshot, String> {
    let (shape, faces) = copied_prism(shape, lineage)?;
    let report = report::checked_report(&shape, &faces)?;
    Ok(PrismProbeSnapshot {
        shape,
        faces,
        report,
        owner: PhantomData,
    })
}
impl PrismProbeSnapshot {
    /// Read only the checked analytic report; example: `snapshot.report().properties.volume`.
    pub fn report(&self) -> &PrismProbeReport {
        &self.report
    }
    /// Copy geometry and role lineage together; example: `snapshot.detached()?`.
    pub fn detached(&self) -> Result<Self, String> {
        capture_prism(&self.shape, &self.faces)
    }
    /// Derive presentation from a detached B-rep copy; example: `snapshot.presentation(options)?`.
    pub fn presentation(&self, options: PrismMeshOptions) -> Result<PrismPresentation, String> {
        mesh::derive_presentation(self, options)
    }
}
