// SPDX-License-Identifier: GPL-2.0-only
//! Certified Rib evaluation on top of Martensite solid construction and Boolean fuse.
use crate::{
    BlendNamedTopology, CertifiedNamedSolidEvaluator, EvaluationEntity, EvaluationFailure,
    EvaluationFailureKind, EvaluationPhase, NamedSolid,
};
use martensite_tkbrep::topods::Shape;
use martensite_tkmath::{
    bnd::BoundingBox3,
    gp::{Point3, Vector3},
};
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakePrism;
use martensite_tktopalgo::{
    brep_bnd_lib::BRepBndLib,
    brep_builder_api::{
        BRepBuilderAPIMakeFace, BRepBuilderAPIMakePolygon, BRepBuilderAPIMakeShape,
        BRepBuilderAPIMakeVertex,
    },
    brep_extrema::BRepExtremaDistShapeShape,
};
use oblikovati_model::{
    feature::{
        FeatureId,
        part::PartFeatureFailure,
        sketch_prism::PrismBoolean,
        sketch_rib::{ResolvedSketchRib, RibDirection, RibThickenSide},
    },
    sketch::{SketchEntityId, SketchPlane},
};

const BOUNDS_EPS: f64 = 1e-4;
const AIR_GAP_EPS: f64 = 1e-4;
const VOLUME_EPS: f64 = 1e-6;
const DIR_EPS: f64 = 1e-9;

impl CertifiedNamedSolidEvaluator<'_> {
    /// Evaluate an open-profile Rib and fuse it into a live named target solid.
    pub(crate) fn evaluate_named_rib(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedSketchRib<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let failure = |message: String| {
            PartFeatureFailure::Kernel(
                EvaluationPhase::Prism
                    .refused(
                        EvaluationFailureKind::InvalidResult,
                        EvaluationEntity::Body,
                        format!("Create rib feature failed: {message}"),
                    )
                    .for_feature(feature),
            )
        };
        let target_shape = body
            .native_shape(self.history, self.document)
            .map_err(failure)?;
        let bounds = shape_bounds(target_shape).map_err(failure)?;
        let plane = input.sketch.plane;
        let points_2d = extract_open_polyline(input.sketch.sketch.geometry(), input.profile_curves)
            .map_err(failure)?;
        let points_3d: Vec<[f64; 3]> = points_2d
            .iter()
            .map(|pt| plane_to_world(plane, *pt))
            .collect();
        for pt in &points_3d {
            if !point_in_bounds(*pt, bounds, BOUNDS_EPS) {
                return Err(failure(
                    "rib profile extends beyond the target solid boundary".into(),
                ));
            }
        }
        let midpoint = polyline_midpoint(&points_3d);
        if point_distance_to_shape(midpoint, target_shape).map_err(failure)? <= AIR_GAP_EPS {
            return Err(failure(
                "rib profile lies on or inside the target solid rather than spanning an open region"
                    .into(),
            ));
        }

        let tool_shape = match input.direction {
            RibDirection::ParallelToSketchPlane => {
                build_parallel_rib(plane, &points_2d, &points_3d, bounds, &input)
                    .map_err(failure)?
            }
            RibDirection::NormalToSketchPlane => {
                build_normal_rib(plane, &points_2d, bounds, &input).map_err(failure)?
            }
        };

        let resolution = match body {
            NamedSolid::Prism(prism) => prism.certification().map(|item| item.resolution),
            NamedSolid::Derived(derived) => derived.resolution(),
        };
        let tool_topology = BlendNamedTopology::from_derived(
            self.history,
            self.document,
            feature,
            tool_shape,
            resolution,
        )
        .map_err(failure)?;
        let tool = tool_topology
            .sweep_boolean_tool(self.history, self.document)
            .map_err(failure)?;
        let base_volume = body.volume(self.history, self.document).map_err(failure)?;
        let candidate = body
            .boolean_tools(
                self.history,
                self.document,
                vec![tool],
                PrismBoolean::Join,
                "rib",
            )
            .map_err(|cause| failure(cause.to_string()))?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(failure)?;
        let result = NamedSolid::Derived(Box::new(named));
        let result_volume = result
            .volume(self.history, self.document)
            .map_err(failure)?;
        if result_volume <= base_volume + VOLUME_EPS {
            return Err(failure(
                "rib did not add material to the target solid".into(),
            ));
        }
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}

fn build_parallel_rib(
    plane: SketchPlane,
    points_2d: &[[f64; 2]],
    points_3d: &[[f64; 3]],
    bounds: [f64; 6],
    input: &ResolvedSketchRib<'_>,
) -> Result<Shape, String> {
    let normal = plane.normal();
    let origin = plane.recipe().origin;
    let (min_n, max_n) = project_bounds_along(bounds, origin, normal);
    let (t0, t1) = thickness_offsets(input.thickness, input.thicken_side);
    if t0 < min_n - BOUNDS_EPS || t1 > max_n + BOUNDS_EPS {
        return Err("rib thickness extends beyond the target solid boundary".into());
    }
    let first = *points_2d.first().ok_or("rib profile: expected endpoints")?;
    let last = *points_2d.last().ok_or("rib profile: expected endpoints")?;
    let chord = [last[0] - first[0], last[1] - first[1]];
    let chord_len = (chord[0] * chord[0] + chord[1] * chord[1]).sqrt();
    if chord_len <= DIR_EPS {
        return Err("rib profile: expected non-degenerate open curve".into());
    }
    let mut in_plane_dir = [-chord[1] / chord_len, chord[0] / chord_len];
    let body_center = [
        0.5 * (bounds[0] + bounds[3]),
        0.5 * (bounds[1] + bounds[4]),
        0.5 * (bounds[2] + bounds[5]),
    ];
    let mid = polyline_midpoint(points_3d);
    let to_center = [
        body_center[0] - mid[0],
        body_center[1] - mid[1],
        body_center[2] - mid[2],
    ];
    let dir_3d = plane_vector(plane, in_plane_dir);
    if dot3(dir_3d, to_center) < 0.0 {
        in_plane_dir = [-in_plane_dir[0], -in_plane_dir[1]];
    }
    if input.flip_direction {
        in_plane_dir = [-in_plane_dir[0], -in_plane_dir[1]];
    }
    let sweep_3d = plane_vector(plane, in_plane_dir);
    let (_, max_s) = project_bounds_along(bounds, mid, sweep_3d);
    let depth = match input.depth {
        Some(value) if !input.to_next => value,
        _ => {
            if max_s <= AIR_GAP_EPS {
                return Err("no target solid lies in the selected rib direction".into());
            }
            max_s
        }
    };
    let mut spine = points_2d.to_vec();
    if input.extend_profile {
        extend_polyline_to_bounds(plane, &mut spine, bounds);
    }
    let mut polygon = Vec::with_capacity(spine.len() * 2);
    for pt in &spine {
        polygon.push(offset_world(plane_to_world(plane, *pt), normal, t0));
    }
    for pt in spine.iter().rev() {
        let shifted = [
            pt[0] + in_plane_dir[0] * depth,
            pt[1] + in_plane_dir[1] * depth,
        ];
        polygon.push(offset_world(plane_to_world(plane, shifted), normal, t0));
    }
    let extrude_vec = scale3(normal, t1 - t0);
    extrude_polygon(&polygon, extrude_vec)
}

fn build_normal_rib(
    plane: SketchPlane,
    points_2d: &[[f64; 2]],
    bounds: [f64; 6],
    input: &ResolvedSketchRib<'_>,
) -> Result<Shape, String> {
    let normal = plane.normal();
    let origin = plane.recipe().origin;
    let (pos_min, pos_max) = project_bounds_along(bounds, origin, normal);
    let mut ext_dir = if pos_max > -pos_min {
        normal
    } else {
        scale3(normal, -1.0)
    };
    if input.flip_direction {
        ext_dir = scale3(ext_dir, -1.0);
    }
    let (_, max_e) = project_bounds_along(bounds, origin, ext_dir);
    if max_e <= AIR_GAP_EPS {
        return Err("no target solid lies in the selected rib direction".into());
    }
    let depth = match input.depth {
        Some(value) if !input.to_next => value,
        _ => max_e,
    };
    let mut spine = points_2d.to_vec();
    if input.extend_profile {
        extend_polyline_to_bounds(plane, &mut spine, bounds);
    }
    if spine.len() < 2 {
        return Err("rib profile: expected at least two points".into());
    }
    let (t0, t1) = thickness_offsets(input.thickness, input.thicken_side);
    let mut left = Vec::with_capacity(spine.len());
    let mut right = Vec::with_capacity(spine.len());
    for idx in 0..spine.len() {
        let prev = spine[idx.saturating_sub(1)];
        let next = spine[(idx + 1).min(spine.len() - 1)];
        let dx = next[0] - prev[0];
        let dy = next[1] - prev[1];
        let len = (dx * dx + dy * dy).sqrt();
        if len <= DIR_EPS {
            return Err("rib profile: degenerate zero-length segment".into());
        }
        let perp = [-dy / len, dx / len];
        let p = spine[idx];
        left.push(plane_to_world(
            plane,
            [p[0] + perp[0] * t0, p[1] + perp[1] * t0],
        ));
        right.push(plane_to_world(
            plane,
            [p[0] + perp[0] * t1, p[1] + perp[1] * t1],
        ));
    }
    let mut polygon = left;
    polygon.extend(right.into_iter().rev());
    extrude_polygon(&polygon, scale3(ext_dir, depth))
}

fn extrude_polygon(points: &[[f64; 3]], vector: [f64; 3]) -> Result<Shape, String> {
    let mut poly = BRepBuilderAPIMakePolygon::new();
    for pt in points {
        poly.add_point(Point3::new(pt[0], pt[1], pt[2]))
            .map_err(|cause| format!("rib polygon vertex: {cause}"))?;
    }
    poly.close()
        .map_err(|cause| format!("rib polygon close: {cause}"))?;
    let wire = poly.wire().map_err(|cause| format!("rib wire: {cause}"))?;
    let face = BRepBuilderAPIMakeFace::from_wire(&wire, true)
        .map_err(|cause| format!("rib face: {cause}"))?
        .face()
        .map_err(|cause| format!("rib face build: {cause}"))?;
    let mut prism = BRepPrimAPIMakePrism::new(
        face.as_ref(),
        Vector3::new(vector[0], vector[1], vector[2]),
        true,
        true,
    )
    .map_err(|cause| format!("rib prism: {cause}"))?;
    let shape = prism
        .shape()
        .map_err(|cause| format!("rib prism shape: {cause}"))?
        .clone();
    crate::checked_shape::require_geometry(&shape, "rib tool")
        .map_err(|cause| cause.to_string())?;
    Ok(shape)
}

fn extract_open_polyline(
    geometry: &oblikovati_model::sketch::Sketch2D,
    curves: &[SketchEntityId],
) -> Result<Vec<[f64; 2]>, String> {
    let mut segments = Vec::with_capacity(curves.len());
    for &id in curves {
        let line = geometry
            .line(id)
            .map_err(|cause| format!("rib curve {}: expected a sketch line: {cause}", id.0))?;
        let start = geometry
            .point(line.endpoints[0])
            .map_err(|cause| format!("rib curve {}: missing start point: {cause}", id.0))?
            .position;
        let end = geometry
            .point(line.endpoints[1])
            .map_err(|cause| format!("rib curve {}: missing end point: {cause}", id.0))?
            .position;
        segments.push((start, end));
    }
    if segments.is_empty() {
        return Err("expected at least one open profile curve".into());
    }
    let mut chain = vec![segments[0].0, segments[0].1];
    for (a, b) in segments.into_iter().skip(1) {
        let tail = *chain.last().ok_or("empty chain")?;
        if dist2(tail, a) <= 1e-6 {
            chain.push(b);
        } else if dist2(tail, b) <= 1e-6 {
            chain.push(a);
        } else {
            return Err("rib profile curves must form a single connected open chain".into());
        }
    }
    Ok(chain)
}

fn extend_polyline_to_bounds(plane: SketchPlane, spine: &mut [[f64; 2]], bounds: [f64; 6]) {
    if spine.len() < 2 {
        return;
    }
    // Extend start along (spine[0] - spine[1]) until hitting bounds
    let d_start = [spine[0][0] - spine[1][0], spine[0][1] - spine[1][1]];
    if let Some(step) = ray_distance_to_bounds(plane, spine[0], d_start, bounds) {
        let len = (d_start[0] * d_start[0] + d_start[1] * d_start[1]).sqrt();
        if len > DIR_EPS {
            spine[0][0] += d_start[0] / len * step;
            spine[0][1] += d_start[1] / len * step;
        }
    }
    let last = spine.len() - 1;
    let d_end = [
        spine[last][0] - spine[last - 1][0],
        spine[last][1] - spine[last - 1][1],
    ];
    if let Some(step) = ray_distance_to_bounds(plane, spine[last], d_end, bounds) {
        let len = (d_end[0] * d_end[0] + d_end[1] * d_end[1]).sqrt();
        if len > DIR_EPS {
            spine[last][0] += d_end[0] / len * step;
            spine[last][1] += d_end[1] / len * step;
        }
    }
}

fn ray_distance_to_bounds(
    plane: SketchPlane,
    start_2d: [f64; 2],
    dir_2d: [f64; 2],
    bounds: [f64; 6],
) -> Option<f64> {
    let len = (dir_2d[0] * dir_2d[0] + dir_2d[1] * dir_2d[1]).sqrt();
    if len <= DIR_EPS {
        return None;
    }
    let origin = plane_to_world(plane, start_2d);
    let dir = plane_vector(plane, [dir_2d[0] / len, dir_2d[1] / len]);
    let mut t_max = f64::INFINITY;
    for axis in 0..3 {
        if dir[axis].abs() > DIR_EPS {
            let bound = if dir[axis] > 0.0 {
                bounds[axis + 3]
            } else {
                bounds[axis]
            };
            let t = (bound - origin[axis]) / dir[axis];
            if t >= 0.0 {
                t_max = t_max.min(t);
            }
        }
    }
    (t_max.is_finite() && t_max > 0.0).then_some(t_max)
}

fn thickness_offsets(thickness: f64, side: RibThickenSide) -> (f64, f64) {
    match side {
        RibThickenSide::Symmetric => (-0.5 * thickness, 0.5 * thickness),
        RibThickenSide::Side1 => (0.0, thickness),
        RibThickenSide::Side2 => (-thickness, 0.0),
    }
}

fn shape_bounds(shape: &Shape) -> Result<[f64; 6], String> {
    let mut bounds = BoundingBox3::default();
    BRepBndLib::add_optimal(shape, &mut bounds, false, false)
        .map_err(|cause| format!("rib target bounds: {cause}"))?;
    bounds
        .get()
        .map_err(|cause| format!("rib target bounds: {cause}"))
}

fn point_distance_to_shape(point: [f64; 3], shape: &Shape) -> Result<f64, String> {
    let mut vertex = BRepBuilderAPIMakeVertex::new(Point3::new(point[0], point[1], point[2]))
        .map_err(|cause| format!("rib probe vertex: {cause}"))?;
    let vertex_shape = vertex
        .shape()
        .map_err(|cause| format!("rib probe vertex shape: {cause}"))?;
    let extrema = BRepExtremaDistShapeShape::with_shapes(vertex_shape, shape, Default::default())
        .map_err(|cause| format!("rib clearance probe: {cause}"))?;
    if !extrema.is_done() {
        return Err("rib clearance probe did not converge".into());
    }
    extrema
        .value()
        .map_err(|cause| format!("rib clearance probe distance: {cause}"))
}

fn point_in_bounds(pt: [f64; 3], bounds: [f64; 6], eps: f64) -> bool {
    pt[0] >= bounds[0] - eps
        && pt[0] <= bounds[3] + eps
        && pt[1] >= bounds[1] - eps
        && pt[1] <= bounds[4] + eps
        && pt[2] >= bounds[2] - eps
        && pt[2] <= bounds[5] + eps
}

fn project_bounds_along(bounds: [f64; 6], origin: [f64; 3], dir: [f64; 3]) -> (f64, f64) {
    let mut min_t = 0.0;
    let mut max_t = 0.0;
    for i in 0..3 {
        let a = (bounds[i] - origin[i]) * dir[i];
        let b = (bounds[i + 3] - origin[i]) * dir[i];
        min_t += a.min(b);
        max_t += a.max(b);
    }
    (min_t, max_t)
}

fn plane_to_world(plane: SketchPlane, pt: [f64; 2]) -> [f64; 3] {
    let r = plane.recipe();
    [
        r.origin[0] + pt[0] * r.x_axis[0] + pt[1] * r.y_axis[0],
        r.origin[1] + pt[0] * r.x_axis[1] + pt[1] * r.y_axis[1],
        r.origin[2] + pt[0] * r.x_axis[2] + pt[1] * r.y_axis[2],
    ]
}

fn plane_vector(plane: SketchPlane, vec: [f64; 2]) -> [f64; 3] {
    let r = plane.recipe();
    [
        vec[0] * r.x_axis[0] + vec[1] * r.y_axis[0],
        vec[0] * r.x_axis[1] + vec[1] * r.y_axis[1],
        vec[0] * r.x_axis[2] + vec[1] * r.y_axis[2],
    ]
}

fn polyline_midpoint(points: &[[f64; 3]]) -> [f64; 3] {
    let first = points[0];
    let last = points[points.len() - 1];
    [
        0.5 * (first[0] + last[0]),
        0.5 * (first[1] + last[1]),
        0.5 * (first[2] + last[2]),
    ]
}

fn offset_world(pt: [f64; 3], dir: [f64; 3], dist: f64) -> [f64; 3] {
    [
        pt[0] + dir[0] * dist,
        pt[1] + dir[1] * dist,
        pt[2] + dir[2] * dist,
    ]
}

fn scale3(vec: [f64; 3], factor: f64) -> [f64; 3] {
    [vec[0] * factor, vec[1] * factor, vec[2] * factor]
}

fn dot3(a: [f64; 3], b: [f64; 3]) -> f64 {
    a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

fn dist2(a: [f64; 2], b: [f64; 2]) -> f64 {
    ((a[0] - b[0]).powi(2) + (a[1] - b[1]).powi(2)).sqrt()
}
