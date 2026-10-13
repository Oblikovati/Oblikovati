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
    build_prism(definition, plane).map_err(|cause| cause.to_string())
}
pub(super) fn copied_prism(
    shape: &Shape,
    lineage: &[(PrismFaceRole, Shape)],
) -> Result<(Shape, Vec<(PrismFaceRole, Shape)>), EvaluationFailure> {
    let mut copier = crate::checked_shape::copy_shape(shape, "prism copy")?;
    let copied = copier
        .shape()
        .map_err(|error| Phase::Copy.native(Entity::Body, String::new(), error))?
        .clone();
    let mut faces = Vec::new();
    for (role, source) in lineage {
        faces.push((*role, copied_role(&mut copier, *role, source)?));
    }
    crate::checked_shape::require_geometry(&copied, "copied prism")?;
    report::checked_report(&copied, &faces)?;
    Ok((copied, faces))
}
fn copied_role(
    copier: &mut rustycad_tktopalgo::brep_builder_api::BRepBuilderAPICopy,
    role: PrismFaceRole,
    source: &Shape,
) -> Result<Shape, EvaluationFailure> {
    let mapped = copier.modified(source).map_err(|cause| {
        Phase::Copy.native(
            Entity::Face(role),
            format!("prism {role:?}: expected copy history"),
            cause,
        )
    })?;
    if mapped.len() != 1 {
        return Err(Phase::Copy.refused(
            Kind::IncompleteHistory,
            Entity::Face(role),
            format!(
                "prism {role:?} copied {} shapes: expected one preserved role",
                mapped.len()
            ),
        ));
    }
    Ok(mapped.first().expect("one copy").clone())
}
pub(super) fn capture_prism(
    shape: &Shape,
    lineage: &[(PrismFaceRole, Shape)],
) -> Result<PrismProbeSnapshot, EvaluationFailure> {
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
        capture_prism(&self.shape, &self.faces).map_err(|cause| cause.to_string())
    }
    /// Derive presentation from a detached B-rep copy; example: `snapshot.presentation(options)?`.
    pub fn presentation(&self, options: PrismMeshOptions) -> Result<PrismPresentation, String> {
        mesh::derive_presentation(self, options)
    }
}
impl PrismProbeSnapshot {
    /// Integrate this retained diagnostic solid; example: `snapshot.physical_properties()?` uses no display mesh.
    pub fn physical_properties(&self) -> Result<crate::AnalyticBodyProperties, String> {
        crate::analytic_body::integrate_body(&self.shape)
    }
}
