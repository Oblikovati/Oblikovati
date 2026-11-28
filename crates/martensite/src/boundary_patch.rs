// SPDX-License-Identifier: GPL-2.0-only
//! Certified Surface Boundary Patch B-rep face/shell construction (#3918).
#[path = "boundary_patch_chain.rs"]
mod chain;

pub use chain::{chain_sketch_boundary_loop, is_interior_diagonal_curve};

use crate::{
    BlendNamedTopology, CertifiedNamedSolidEvaluator, EvaluationEntity, EvaluationFailure,
    EvaluationFailureKind, EvaluationPhase, NamedSolid,
};
use martensite_tkbrep::topods::{Builder, Shape, Shell};
use martensite_tkmath::gp::{Axis2, Circle3, Direction3, Ellipse3, Point3};
use martensite_tktopalgo::brep_builder_api::{
    BRepBuilderAPIMakeEdge, BRepBuilderAPIMakeFace, BRepBuilderAPIMakePolygon,
    BRepBuilderAPIMakeWire,
};
use oblikovati_model::feature::{
    FeatureId,
    boundary_patch::{BoundaryCurveRef, ResolvedBoundaryPatch},
    part::PartFeatureFailure,
};

const POINT_TOL: f64 = 1e-5;

impl CertifiedNamedSolidEvaluator<'_> {
    /// Evaluate a Surface Boundary Patch feature into a named Surface body (`SrfN`).
    pub(crate) fn evaluate_named_boundary_patch(
        &mut self,
        feature: FeatureId,
        input: ResolvedBoundaryPatch<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let failure = |kind: EvaluationFailureKind, message: String| {
            PartFeatureFailure::Kernel(
                EvaluationPhase::Prism
                    .refused(kind, EvaluationEntity::Body, message)
                    .for_feature(feature),
            )
        };
        let shell = build_boundary_patch_shell(&input)
            .map_err(|msg| failure(EvaluationFailureKind::InvalidResult, msg))?;
        let topology = BlendNamedTopology::from_boundary_patch_shell(
            self.history,
            self.document,
            feature,
            shell,
            None,
        )
        .map_err(|msg| failure(EvaluationFailureKind::InvalidReference, msg))?;
        Ok(NamedSolid::Derived(Box::new(topology)))
    }
}

fn build_boundary_patch_shell(input: &ResolvedBoundaryPatch<'_>) -> Result<Shape, String> {
    let boundary_refs: Vec<BoundaryCurveRef> = input
        .edge_sets
        .iter()
        .flat_map(|set| set.edges.iter().copied())
        .collect();
    if boundary_refs.is_empty() {
        return Err("boundary patch: expected at least one boundary curve".into());
    }
    if boundary_refs.len() == 1
        && input.guide_rails.is_empty()
        && let Some(face) = try_single_closed_conic(input, boundary_refs[0])?
    {
        return wrap_face_as_shell(face);
    }
    if boundary_refs.len() < 2 {
        return Err("boundary patch: expected a closed loop or at least two boundary edges".into());
    }
    let mut segments_3d = Vec::new();
    for curve_ref in boundary_refs.iter().chain(input.guide_rails.iter()) {
        segments_3d.push(resolve_curve_segment_3d(input, *curve_ref)?);
    }
    let ordered_points = order_boundary_polygon_points(&segments_3d)?;
    let face = build_planar_face_from_points(&ordered_points)?;
    wrap_face_as_shell(face)
}

fn try_single_closed_conic(
    input: &ResolvedBoundaryPatch<'_>,
    curve_ref: BoundaryCurveRef,
) -> Result<Option<Shape>, String> {
    let resolved = input.sketches.get(&curve_ref.sketch().0).ok_or_else(|| {
        format!(
            "boundary patch sketch {:?}: not resolved",
            curve_ref.sketch()
        )
    })?;
    let geom = resolved.sketch.geometry();
    let entity = curve_ref.entity();
    if let Some(circle) = geom.circles().iter().find(|c| c.id == entity) {
        let center_2d = geom
            .point(circle.center)
            .map_err(|e| e.to_string())?
            .position;
        let axes = sketch_axis2(resolved.plane, center_2d)?;
        let circ = Circle3::new(axes, circle.radius).map_err(|e| e.to_string())?;
        let edge = BRepBuilderAPIMakeEdge::from_circ(circ)
            .map_err(|e| e.to_string())?
            .edge()
            .map_err(|e| e.to_string())?;
        let wire = BRepBuilderAPIMakeWire::from_edge(&edge)
            .map_err(|e| e.to_string())?
            .wire()
            .map_err(|e| e.to_string())?;
        let face = BRepBuilderAPIMakeFace::from_wire(&wire, true)
            .map_err(|e| e.to_string())?
            .face()
            .map_err(|e| e.to_string())?;
        return Ok(Some(face.into_shape()));
    }
    if let Some(ellipse) = geom.ellipses().iter().find(|e| e.id() == entity) {
        let center_2d = geom
            .point(ellipse.center())
            .map_err(|e| e.to_string())?
            .position;
        let axis_2d = ellipse.major_axis();
        let [r_maj, r_min] = ellipse.radii();
        let angle = axis_2d[1].atan2(axis_2d[0]);
        let axes = sketch_rotated_axis2(resolved.plane, center_2d, angle)?;
        let elips = Ellipse3::new(axes, r_maj, r_min).map_err(|e| e.to_string())?;
        let edge = BRepBuilderAPIMakeEdge::from_elips(elips)
            .map_err(|e| e.to_string())?
            .edge()
            .map_err(|e| e.to_string())?;
        let wire = BRepBuilderAPIMakeWire::from_edge(&edge)
            .map_err(|e| e.to_string())?
            .wire()
            .map_err(|e| e.to_string())?;
        let face = BRepBuilderAPIMakeFace::from_wire(&wire, true)
            .map_err(|e| e.to_string())?
            .face()
            .map_err(|e| e.to_string())?;
        return Ok(Some(face.into_shape()));
    }
    Ok(None)
}

fn resolve_curve_segment_3d(
    input: &ResolvedBoundaryPatch<'_>,
    curve_ref: BoundaryCurveRef,
) -> Result<Vec<[f64; 3]>, String> {
    let resolved = input.sketches.get(&curve_ref.sketch().0).ok_or_else(|| {
        format!(
            "boundary patch sketch {:?}: not resolved",
            curve_ref.sketch()
        )
    })?;
    let geom = resolved.sketch.geometry();
    let entity = curve_ref.entity();
    if let Some(line) = geom.lines().iter().find(|l| l.id == entity) {
        let a = geom
            .point(line.endpoints[0])
            .map_err(|e| e.to_string())?
            .position;
        let b = geom
            .point(line.endpoints[1])
            .map_err(|e| e.to_string())?
            .position;
        let p0 = resolved.plane.to_model(a).map_err(|e| e.to_string())?;
        let p1 = resolved.plane.to_model(b).map_err(|e| e.to_string())?;
        return Ok(vec![p0, p1]);
    }
    if let Some(arc) = geom.arcs().iter().find(|a| a.id == entity) {
        let center = geom.point(arc.center).map_err(|e| e.to_string())?.position;
        let start = geom.point(arc.start).map_err(|e| e.to_string())?.position;
        let end = geom.point(arc.end).map_err(|e| e.to_string())?.position;
        let r = (start[0] - center[0]).hypot(start[1] - center[1]);
        let a0 = (start[1] - center[1]).atan2(start[0] - center[0]);
        let mut a1 = (end[1] - center[1]).atan2(end[0] - center[0]);
        if a1 <= a0 {
            a1 += std::f64::consts::TAU;
        }
        let steps = 16usize;
        let mut pts = Vec::with_capacity(steps + 1);
        for i in 0..=steps {
            let t = i as f64 / steps as f64;
            let ang = a0 + (a1 - a0) * t;
            let p2 = [center[0] + r * ang.cos(), center[1] + r * ang.sin()];
            pts.push(resolved.plane.to_model(p2).map_err(|e| e.to_string())?);
        }
        return Ok(pts);
    }
    Err(format!(
        "boundary patch curve {curve_ref:?}: expected a sketch line, arc, circle, or ellipse"
    ))
}

fn order_boundary_polygon_points(polylines: &[Vec<[f64; 3]>]) -> Result<Vec<[f64; 3]>, String> {
    if polylines.is_empty() {
        return Err("boundary patch: expected boundary segments".into());
    }
    let mut used = vec![false; polylines.len()];
    let mut ordered = polylines[0].clone();
    used[0] = true;
    loop {
        let mut extended = false;
        let tail = *ordered
            .last()
            .ok_or("boundary patch: empty initial polyline")?;
        let head = ordered[0];
        for (idx, poly) in polylines.iter().enumerate() {
            if used[idx] || poly.len() < 2 {
                continue;
            }
            let p_start = poly[0];
            let p_end = poly[poly.len() - 1];
            if dist3(tail, p_start) <= POINT_TOL {
                ordered.extend_from_slice(&poly[1..]);
            } else if dist3(tail, p_end) <= POINT_TOL {
                ordered.extend(poly.iter().rev().skip(1).copied());
            } else if dist3(head, p_end) <= POINT_TOL {
                let mut prefix = poly[..poly.len() - 1].to_vec();
                prefix.extend(ordered);
                ordered = prefix;
            } else if dist3(head, p_start) <= POINT_TOL {
                let mut prefix: Vec<_> = poly.iter().rev().take(poly.len() - 1).copied().collect();
                prefix.extend(ordered);
                ordered = prefix;
            } else {
                continue;
            }
            used[idx] = true;
            extended = true;
            break;
        }
        if !extended {
            let Some((idx, reverse)) = next_disconnected_chain(&ordered, polylines, &used) else {
                break;
            };
            used[idx] = true;
            if reverse {
                ordered.extend(polylines[idx].iter().rev().copied());
            } else {
                ordered.extend_from_slice(&polylines[idx]);
            }
        }
    }
    let mut cleaned: Vec<[f64; 3]> = Vec::with_capacity(ordered.len());
    for pt in ordered {
        if cleaned
            .last()
            .is_none_or(|prev| dist3(*prev, pt) > POINT_TOL)
        {
            cleaned.push(pt);
        }
    }
    if cleaned.len() >= 2 && dist3(cleaned[0], cleaned[cleaned.len() - 1]) <= POINT_TOL {
        cleaned.pop();
    }
    if cleaned.len() < 3 {
        return Err(
            "boundary patch: expected at least three non-collinear boundary vertices".into(),
        );
    }
    if polygon_area_3d(&cleaned) <= 1e-9 {
        return Err("boundary patch: degenerate zero-area boundary".into());
    }
    Ok(cleaned)
}

fn next_disconnected_chain(
    ordered: &[[f64; 3]],
    polylines: &[Vec<[f64; 3]>],
    used: &[bool],
) -> Option<(usize, bool)> {
    let tail = *ordered.last()?;
    let mut best: Option<(usize, bool, f64)> = None;
    for (idx, poly) in polylines.iter().enumerate() {
        if used[idx] || poly.len() < 2 {
            continue;
        }
        let d_start = dist3(tail, poly[0]);
        let d_end = dist3(tail, poly[poly.len() - 1]);
        let (reverse, dist) = if d_end < d_start {
            (true, d_end)
        } else {
            (false, d_start)
        };
        if best.is_none_or(|(_, _, best_d)| dist < best_d) {
            best = Some((idx, reverse, dist));
        }
    }
    best.map(|(idx, rev, _)| (idx, rev))
}

fn build_planar_face_from_points(points: &[[f64; 3]]) -> Result<Shape, String> {
    let mut poly = BRepBuilderAPIMakePolygon::new();
    for &[x, y, z] in points {
        poly.add_point(Point3::new(x, y, z))
            .map_err(|e| e.to_string())?;
    }
    poly.close().map_err(|e| e.to_string())?;
    let wire = poly.wire().map_err(|e| e.to_string())?;
    let face = BRepBuilderAPIMakeFace::from_wire(&wire, true)
        .map_err(|e| e.to_string())?
        .face()
        .map_err(|e| e.to_string())?;
    Ok(face.into_shape())
}

fn wrap_face_as_shell(face: Shape) -> Result<Shape, String> {
    let mut shell = Shell::default();
    let builder = Builder;
    builder.make_shell(&mut shell);
    let mut shape = shell.into_shape();
    builder.add(&mut shape, &face).map_err(|e| e.to_string())?;
    Ok(shape)
}

fn sketch_axis2(
    plane: oblikovati_model::sketch::SketchPlane,
    center_2d: [f64; 2],
) -> Result<Axis2, String> {
    let origin = plane.to_model(center_2d).map_err(|e| e.to_string())?;
    let n = plane.normal();
    let x = plane.recipe().x_axis;
    Axis2::with_x_direction(
        Point3::new(origin[0], origin[1], origin[2]),
        Direction3::new(n[0], n[1], n[2]).map_err(|e| e.to_string())?,
        Direction3::new(x[0], x[1], x[2]).map_err(|e| e.to_string())?,
    )
    .map_err(|e| e.to_string())
}

fn sketch_rotated_axis2(
    plane: oblikovati_model::sketch::SketchPlane,
    center_2d: [f64; 2],
    angle: f64,
) -> Result<Axis2, String> {
    let origin = plane.to_model(center_2d).map_err(|e| e.to_string())?;
    let n = plane.normal();
    let recipe = plane.recipe();
    let px = recipe.x_axis;
    let py = recipe.y_axis;
    let (s, c) = angle.sin_cos();
    let rx = [
        px[0] * c + py[0] * s,
        px[1] * c + py[1] * s,
        px[2] * c + py[2] * s,
    ];
    Axis2::with_x_direction(
        Point3::new(origin[0], origin[1], origin[2]),
        Direction3::new(n[0], n[1], n[2]).map_err(|e| e.to_string())?,
        Direction3::new(rx[0], rx[1], rx[2]).map_err(|e| e.to_string())?,
    )
    .map_err(|e| e.to_string())
}

fn dist3(a: [f64; 3], b: [f64; 3]) -> f64 {
    ((a[0] - b[0]).powi(2) + (a[1] - b[1]).powi(2) + (a[2] - b[2]).powi(2)).sqrt()
}

fn polygon_area_3d(points: &[[f64; 3]]) -> f64 {
    let origin = points[0];
    let mut area = [0.0_f64; 3];
    for pair in points[1..].windows(2) {
        let u = [
            pair[0][0] - origin[0],
            pair[0][1] - origin[1],
            pair[0][2] - origin[2],
        ];
        let v = [
            pair[1][0] - origin[0],
            pair[1][1] - origin[1],
            pair[1][2] - origin[2],
        ];
        area[0] += u[1] * v[2] - u[2] * v[1];
        area[1] += u[2] * v[0] - u[0] * v[2];
        area[2] += u[0] * v[1] - u[1] * v[0];
    }
    0.5 * (area[0].powi(2) + area[1].powi(2) + area[2].powi(2)).sqrt()
}
