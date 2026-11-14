// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use martensite_tkbrep::topods::Shape;
use martensite_tkmath::gp::{Direction3, Line3, Point3};
use martensite_tktopalgo::int_curves_face::IntCurvesFaceShapeIntersector;
use std::{marker::PhantomData, rc::Rc};
/// How near a face's boundary a line of sight still lands on the face.
const PICK_TOLERANCE: f64 = 1e-7; // tol:pick
fn sight_line([x, y, z]: [f64; 3], [dx, dy, dz]: [f64; 3]) -> Result<Line3, String> {
    let direction = Direction3::new(dx, dy, dz).map_err(|cause| {
        format!(
            "line direction {:?}: expected a nonzero vector: {cause}",
            [dx, dy, dz]
        )
    })?;
    Ok(Line3::new(Point3::new(x, y, z), direction))
}
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
    let (shape, faces, _) = copied_prism_entities(shape, lineage, &[])?;
    Ok((shape, faces))
}
type PrismCopy = (Shape, Vec<(PrismFaceRole, Shape)>, Vec<Shape>);
fn copied_prism_entities(
    shape: &Shape,
    lineage: &[(PrismFaceRole, Shape)],
    subentities: &[Shape],
) -> Result<PrismCopy, EvaluationFailure> {
    let mut copier = crate::checked_shape::copy_shape(shape, "prism copy")?;
    let copied = copier
        .shape()
        .map_err(|error| Phase::Copy.native(Entity::Body, String::new(), error))?
        .clone();
    let faces = lineage
        .iter()
        .map(|(role, source)| Ok((*role, copied_role(&mut copier, *role, source)?)))
        .collect::<Result<Vec<_>, EvaluationFailure>>()?;
    let subentities = subentities
        .iter()
        .enumerate()
        .map(|(index, source)| copied_subentity(&mut copier, index, source))
        .collect::<Result<Vec<_>, _>>()?;
    crate::checked_shape::require_geometry(&copied, "copied prism")?;
    report::checked_report(&copied, &faces)?;
    Ok((copied, faces, subentities))
}
fn copied_subentity(
    copier: &mut martensite_tktopalgo::brep_builder_api::BRepBuilderAPICopy,
    index: usize,
    source: &Shape,
) -> Result<Shape, EvaluationFailure> {
    let mapped = copier.modified(source).map_err(|cause| {
        Phase::Copy.native(
            Entity::Body,
            format!("prism subentity {index} copy history"),
            cause,
        )
    })?;
    if mapped.len() != 1 {
        return Err(Phase::Copy.refused(
            Kind::IncompleteHistory,
            Entity::Body,
            format!(
                "prism subentity {index} copied {} shapes: expected exactly one",
                mapped.len()
            ),
        ));
    }
    Ok(mapped.first().expect("one copy").clone())
}

fn copied_role(
    copier: &mut martensite_tktopalgo::brep_builder_api::BRepBuilderAPICopy,
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
pub(super) fn capture_prism_with_subentities(
    shape: &Shape,
    lineage: &[(PrismFaceRole, Shape)],
    subentities: &[Shape],
) -> Result<(PrismProbeSnapshot, Vec<Shape>), EvaluationFailure> {
    let (shape, faces, copied) = copied_prism_entities(shape, lineage, subentities)?;
    verify_subentities(&shape, &copied)?;
    let report = report::checked_report(&shape, &faces)?;
    Ok((
        PrismProbeSnapshot {
            shape,
            faces,
            report,
            owner: PhantomData,
        },
        copied,
    ))
}
fn verify_subentities(shape: &Shape, subentities: &[Shape]) -> Result<(), EvaluationFailure> {
    use martensite_tkbrep::{topexp, toptools::IndexedMapOfShape};
    use martensite_tkg3d::top_abs::ShapeEnum;
    for entity in subentities {
        let kind = entity.shape_type().map_err(|cause| {
            Phase::Copy.native(Entity::Body, "copied prism subentity kind".into(), cause)
        })?;
        if !matches!(kind, ShapeEnum::Edge | ShapeEnum::Vertex) {
            return Err(Phase::Copy.refused(
                Kind::IncompleteHistory,
                Entity::Body,
                format!("copied prism subentity {kind:?}: expected edge or vertex"),
            ));
        }
    }
    for kind in [ShapeEnum::Edge, ShapeEnum::Vertex] {
        let mut census = IndexedMapOfShape::default();
        topexp::map_shapes_of_type(shape, kind, &mut census).map_err(|cause| {
            Phase::Copy.native(Entity::Body, "copied prism topology".into(), cause)
        })?;
        let typed = subentities
            .iter()
            .map(|entity| {
                entity
                    .shape_type()
                    .map(|actual| (entity, actual))
                    .map_err(|cause| {
                        Phase::Copy.native(
                            Entity::Body,
                            "copied prism subentity kind".into(),
                            cause,
                        )
                    })
            })
            .collect::<Result<Vec<_>, _>>()?;
        let matching: Vec<_> = typed
            .into_iter()
            .filter_map(|(entity, actual)| (actual == kind).then_some(entity))
            .collect();
        if matching.len() != census.extent() {
            return Err(Phase::Copy.refused(
                Kind::IncompleteHistory,
                Entity::Body,
                format!(
                    "copied prism {kind:?} names={}, topology={}: expected complete coverage",
                    matching.len(),
                    census.extent()
                ),
            ));
        }
        for index in 1..=census.extent() {
            let entity = census.find_key(index).map_err(|cause| {
                Phase::Copy.native(Entity::Body, "copied prism topology member".into(), cause)
            })?;
            if matching
                .iter()
                .filter(|candidate| candidate.is_same(entity))
                .count()
                != 1
            {
                return Err(Phase::Copy.refused(
                    Kind::IncompleteHistory,
                    Entity::Body,
                    format!("copied prism {kind:?} {index}: expected exactly one mapped source"),
                ));
            }
        }
    }
    Ok(())
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
    /// The nearest face a forward ray meets, and its distance: exact, from the kernel's
    /// faces, never from a display mesh. Example: `body.face_on_line([point, direction])?`.
    pub fn face_on_line(
        &self,
        [point, direction]: [[f64; 3]; 2],
    ) -> Result<Option<(PrismFaceRole, f64)>, String> {
        let line = sight_line(point, direction)?;
        let mut nearest: Option<(PrismFaceRole, f64)> = None;
        for (role, face) in &self.faces {
            let mut intersector = IntCurvesFaceShapeIntersector::new();
            let reach = (|| -> Result<Option<f64>, martensite_tkernel::standard::Failure> {
                intersector.load(face, PICK_TOLERANCE)?;
                intersector.perform_nearest(&line, 0., f64::MAX / 2.)?;
                if intersector.nb_pnt() == 0 {
                    return Ok(None);
                }
                intersector.w_parameter(1).map(Some)
            })()
            .map_err(|cause| format!("face on line {point:?} {direction:?}: {cause}"))?;
            if let Some(reach) = reach
                && nearest.is_none_or(|(_, best)| reach < best)
            {
                nearest = Some((*role, reach));
            }
        }
        Ok(nearest)
    }
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
