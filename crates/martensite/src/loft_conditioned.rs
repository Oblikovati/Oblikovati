// SPDX-License-Identifier: GPL-2.0-only
//! Construct a circular surface with a weighted tangent at its first section.
use super::*;
use martensite_tkbrep::{
    topexp,
    topods::{Builder, Shell},
};
use martensite_tkernel::{ncollection::Array1, standard::Handle};
use martensite_tkg3d::geom::{GeomBezierCurve, GeomCurve};
use martensite_tkmath::gp::Axis1;
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeRevol;
use oblikovati_model::feature::sketch_loft::LoftTakeoffCondition;

pub(super) fn build(
    first: &ResolvedLoftCircle,
    last: &ResolvedLoftCircle,
    condition: LoftTakeoffCondition,
    separation: f64,
    resolution: PrismResolution,
) -> Result<LoftCandidate, EvaluationFailure> {
    let axis = axis_for_sections(first, last, separation, resolution.weld)?;
    let radial = first.x_direction;
    let (sin, cos) = condition.angle_degrees.to_radians().sin_cos();
    let handle = condition.weight * separation / 3.;
    let section = |radius: f64, height: f64| {
        let [x, y, z] =
            std::array::from_fn(|i| first.center[i] + radial[i] * radius + axis[i] * height);
        Point3::new(x, y, z)
    };
    let poles = [
        section(first.radius, 0.),
        section(first.radius + handle * cos, handle * sin),
        section((first.radius + 2. * last.radius) / 3., 2. * separation / 3.),
        section(last.radius, separation),
    ];
    let make_meridian = || -> Result<Edge, Failure> {
        let poles = Array1::from_shared(&poles, 1);
        let curve: Handle<dyn GeomCurve> = Handle::new(GeomBezierCurve::new(&poles)?).into_upcast();
        BRepBuilderAPIMakeEdge::from_curve(&curve)?.edge()
    };
    let meridian =
        make_meridian().map_err(|cause| native(Phase::Profile, "tangent meridian", cause))?;
    let [x, y, z] = first.center;
    let [dx, dy, dz] = axis;
    let direction = Direction3::new(dx, dy, dz)
        .map_err(|cause| native(Phase::Profile, "section axis", cause))?;
    let axis = Axis1::new(Point3::new(x, y, z), direction);
    let mut revolved = BRepPrimAPIMakeRevol::full(meridian.as_ref(), axis, true)
        .map_err(|cause| native(Phase::Loft, "conditioned surface", cause))?;
    let wall = revolved
        .shape()
        .cloned()
        .map_err(|cause| native(Phase::Loft, "conditioned wall", cause))?;
    let shape =
        one_face_shell(&wall).map_err(|cause| native(Phase::Loft, "conditioned shell", cause))?;
    require_body_kind(&shape, false)?;
    crate::checked_shape::require_geometry(&shape, "conditioned Loft result")?;
    let achieved =
        crate::checked_shape::measured_tolerances(&shape, resolution.maximum_tolerance, "loft")?;
    verify_faces(&shape, &[&wall])?;
    let [first, last] = [
        topexp::first_vertex(&meridian, true),
        topexp::last_vertex(&meridian, true),
    ];
    let trims = surface_trims(
        &shape,
        [
            generated_trim(&mut revolved, first.as_ref())?,
            generated_trim(&mut revolved, last.as_ref())?,
        ]
        .each_ref(),
    )?;
    Ok(LoftCandidate {
        shape,
        history: LoftTopology::Surface { trims, wall },
        resolution,
        achieved,
    })
}

fn axis_for_sections(
    first: &ResolvedLoftCircle,
    last: &ResolvedLoftCircle,
    separation: f64,
    tolerance: f64,
) -> Result<[f64; 3], EvaluationFailure> {
    let delta: [f64; 3] = std::array::from_fn(|i| last.center[i] - first.center[i]);
    let off_axis = |normal: [f64; 3]| {
        let projected: f64 = delta.into_iter().zip(normal).map(|(a, b)| a * b).sum();
        delta
            .into_iter()
            .zip(normal)
            .map(|(offset, direction)| offset - projected * direction)
            .fold(0.0_f64, f64::hypot)
    };
    if separation <= tolerance || off_axis(first.normal).max(off_axis(last.normal)) > tolerance {
        return Err(Phase::Profile.refused(
            Kind::Unsupported,
            Entity::Body,
            "A weighted Loft takeoff requires separate, coaxial circular sections.".into(),
        ));
    }
    Ok(delta.map(|component| component / separation))
}

fn one_face_shell(face: &Shape) -> Result<Shape, Failure> {
    let mut shell = Shell::default();
    let builder = Builder;
    builder.make_shell(&mut shell);
    let mut shape = shell.into_shape();
    builder.add(&mut shape, face)?;
    Ok(shape)
}

fn generated_trim(
    revolved: &mut BRepPrimAPIMakeRevol,
    vertex: &Shape,
) -> Result<Shape, EvaluationFailure> {
    let generated = revolved
        .generated(vertex)
        .map_err(|cause| native(Phase::History, "conditioned section trim", cause))?;
    generated
        .first()
        .filter(|_| generated.len() == 1)
        .cloned()
        .ok_or_else(|| history_failure("expected one generated trim per section"))
}
