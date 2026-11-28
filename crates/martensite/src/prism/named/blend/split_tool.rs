// SPDX-License-Identifier: GPL-2.0-only
//! Construct cutting surfaces and section edges for Martensite B-rep Split (#3918).
use martensite_tkbo::brep_algo_api::BRepAlgoAPISection;
use martensite_tkbrep::{topexp, topods::Shape, toptools::IndexedMapOfShape};
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkmath::{
    bnd::BoundingBox3,
    gp::{Axis2, Circle3, Direction3, Point3, Transform3, Vector3},
};
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakePrism;
use martensite_tktopalgo::{
    brep_bnd_lib::BRepBndLib,
    brep_builder_api::{
        BRepBuilderAPIMakeEdge, BRepBuilderAPIMakeFace, BRepBuilderAPIMakePolygon,
        BRepBuilderAPIMakeShape, BRepBuilderAPITransform,
    },
};
use oblikovati_model::{
    feature::split::{ResolvedSplitCurve, ResolvedSplitTool},
    sketch::SketchPlane,
};

/// A built cutting tool surface along with its reference plane `(origin, normal)` for side classification.
pub(super) struct BuiltSplitTool {
    pub(super) surface: Shape,
    pub(super) plane_origin: [f64; 3],
    pub(super) plane_normal: [f64; 3],
}

/// Build an oversized planar or ruled cutting surface covering `body`'s bounding box.
pub(super) fn build_split_tool(
    body: &Shape,
    tool: &ResolvedSplitTool,
) -> Result<BuiltSplitTool, String> {
    let (center, extent) = body_bounds_center_and_extent(body)?;
    match tool {
        ResolvedSplitTool::Plane(plane) => build_plane_tool(*plane, center, extent),
        ResolvedSplitTool::Curve(curve) => build_curve_tool(curve, extent),
    }
}

/// Compute the 1D intersection edges between `target_faces` and `tool_surface`.
pub(super) fn build_section_edges(
    target_faces: &[Shape],
    tool_surface: &Shape,
) -> Result<Vec<Shape>, String> {
    let mut edges = Vec::new();
    for face in target_faces {
        let section = BRepAlgoAPISection::from_shapes(face, tool_surface, None, true)
            .map_err(|cause| format!("split face section: {cause}"))?;
        if !section.is_done() || section.has_errors() {
            return Err("split face section: expected completed intersection".into());
        }
        let mut map = IndexedMapOfShape::default();
        topexp::map_shapes_of_type(section.shape(), ShapeEnum::Edge, &mut map)
            .map_err(|cause| cause.to_string())?;
        for index in 1..=map.extent() {
            edges.push(
                map.find_key(index)
                    .cloned()
                    .map_err(|cause| cause.to_string())?,
            );
        }
    }
    if edges.is_empty() {
        return Err("split faces: splitting tool does not intersect the selected faces".into());
    }
    Ok(edges)
}

pub(super) fn body_bounds_center_and_extent(body: &Shape) -> Result<([f64; 3], f64), String> {
    let mut bounds = BoundingBox3::default();
    BRepBndLib::add_optimal(body, &mut bounds, false, false)
        .map_err(|cause| format!("split bounds: {cause}"))?;
    let [xmin, ymin, zmin, xmax, ymax, zmax] = bounds
        .get()
        .map_err(|cause| format!("split bounds extent: {cause}"))?;
    let center = [
        0.5 * (xmin + xmax),
        0.5 * (ymin + ymax),
        0.5 * (zmin + zmax),
    ];
    let dx = xmax - xmin;
    let dy = ymax - ymin;
    let dz = zmax - zmin;
    let diag = (dx * dx + dy * dy + dz * dz).sqrt();
    if !diag.is_finite() || diag <= 0.0 {
        return Err("split bounds: expected a nonempty target solid".into());
    }
    Ok((center, (diag * 2.5).max(10.0)))
}

pub(super) fn build_plane_tool(
    plane: SketchPlane,
    center: [f64; 3],
    extent: f64,
) -> Result<BuiltSplitTool, String> {
    let recipe = plane.recipe();
    let origin = recipe.origin;
    let normal = unit_vector(plane.normal(), "split plane normal")?;
    let u = unit_vector(recipe.x_axis, "split plane x-axis")?;
    let v = unit_vector(cross(normal, u), "split plane y-axis")?;
    // Project the body center onto the plane so the oversized face is centered on the body.
    let offset = sub(center, origin);
    let signed = dot(offset, normal);
    let anchor = sub(center, scale(normal, signed));
    let corners = [
        add(add(anchor, scale(u, -extent)), scale(v, -extent)),
        add(add(anchor, scale(u, extent)), scale(v, -extent)),
        add(add(anchor, scale(u, extent)), scale(v, extent)),
        add(add(anchor, scale(u, -extent)), scale(v, extent)),
    ];
    let mut poly = BRepBuilderAPIMakePolygon::new();
    for c in corners {
        poly.add_point(Point3::new(c[0], c[1], c[2]))
            .map_err(|cause| format!("split plane corner: {cause}"))?;
    }
    poly.close()
        .map_err(|cause| format!("split plane wire close: {cause}"))?;
    let wire = poly
        .wire()
        .map_err(|cause| format!("split plane wire: {cause}"))?;
    let surface: Shape = BRepBuilderAPIMakeFace::from_wire(&wire, true)
        .map_err(|cause| format!("split plane face: {cause}"))?
        .face()
        .map_err(|cause| format!("split plane face result: {cause}"))?
        .into();
    Ok(BuiltSplitTool {
        surface,
        plane_origin: origin,
        plane_normal: normal,
    })
}

fn build_curve_tool(curve: &ResolvedSplitCurve, extent: f64) -> Result<BuiltSplitTool, String> {
    let (edge, normal, plane_origin, plane_normal) = match curve {
        ResolvedSplitCurve::Line {
            start,
            end,
            plane_normal,
        } => {
            let edge: Shape = BRepBuilderAPIMakeEdge::from_points(
                Point3::new(start[0], start[1], start[2]),
                Point3::new(end[0], end[1], end[2]),
            )
            .map_err(|cause| format!("split line edge: {cause}"))?
            .edge()
            .map_err(|cause| format!("split line edge result: {cause}"))?
            .into();
            let line_dir = unit_vector(sub(*end, *start), "split line direction")?;
            let n = unit_vector(*plane_normal, "split sketch plane normal")?;
            let side_normal = unit_vector(cross(n, line_dir), "split line side normal")?;
            (edge, n, *start, side_normal)
        }
        ResolvedSplitCurve::Arc {
            center,
            start,
            end,
            counter_clockwise,
            plane,
        } => {
            let n = unit_vector(plane.normal(), "split arc normal")?;
            let dir = Direction3::new(n[0], n[1], n[2]).map_err(|cause| cause.to_string())?;
            let radius = length(sub(*start, *center));
            let axes = Axis2::new(Point3::new(center[0], center[1], center[2]), dir)
                .map_err(|cause| cause.to_string())?;
            let circle = Circle3::new(axes, radius).map_err(|cause| cause.to_string())?;
            let (p0, p1) = if *counter_clockwise {
                (*start, *end)
            } else {
                (*end, *start)
            };
            let edge: Shape = BRepBuilderAPIMakeEdge::from_circ_points(
                circle,
                Point3::new(p0[0], p0[1], p0[2]),
                Point3::new(p1[0], p1[1], p1[2]),
            )
            .map_err(|cause| format!("split arc edge: {cause}"))?
            .edge()
            .map_err(|cause| format!("split arc edge result: {cause}"))?
            .into();
            (edge, n, *center, n)
        }
        ResolvedSplitCurve::Circle {
            center,
            radius,
            plane,
        } => {
            let n = unit_vector(plane.normal(), "split circle normal")?;
            let dir = Direction3::new(n[0], n[1], n[2]).map_err(|cause| cause.to_string())?;
            let axes = Axis2::new(Point3::new(center[0], center[1], center[2]), dir)
                .map_err(|cause| cause.to_string())?;
            let circle = Circle3::new(axes, *radius).map_err(|cause| cause.to_string())?;
            let edge: Shape = BRepBuilderAPIMakeEdge::from_circ(circle)
                .map_err(|cause| format!("split circle edge: {cause}"))?
                .edge()
                .map_err(|cause| format!("split circle edge result: {cause}"))?
                .into();
            (edge, n, *center, n)
        }
    };
    let mut shift = Transform3::default();
    shift.set_translation(Vector3::new(
        -normal[0] * extent,
        -normal[1] * extent,
        -normal[2] * extent,
    ));
    let mut moved = BRepBuilderAPITransform::with_shape(&edge, shift, true, false)
        .map_err(|cause| format!("split curve shift: {cause}"))?;
    let shifted = moved
        .shape()
        .map_err(|cause| format!("split curve shift shape: {cause}"))?
        .clone();
    let sweep = Vector3::new(
        2.0 * normal[0] * extent,
        2.0 * normal[1] * extent,
        2.0 * normal[2] * extent,
    );
    let mut prism = BRepPrimAPIMakePrism::new(&shifted, sweep, false, true)
        .map_err(|cause| format!("split curve ribbon: {cause}"))?;
    let surface = prism
        .shape()
        .map_err(|cause| format!("split curve ribbon shape: {cause}"))?
        .clone();
    Ok(BuiltSplitTool {
        surface,
        plane_origin,
        plane_normal,
    })
}

fn dot(a: [f64; 3], b: [f64; 3]) -> f64 {
    a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}
fn cross(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}
fn sub(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [a[0] - b[0], a[1] - b[1], a[2] - b[2]]
}
fn add(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [a[0] + b[0], a[1] + b[1], a[2] + b[2]]
}
fn scale(a: [f64; 3], s: f64) -> [f64; 3] {
    [a[0] * s, a[1] * s, a[2] * s]
}
fn length(a: [f64; 3]) -> f64 {
    dot(a, a).sqrt()
}
fn unit_vector(v: [f64; 3], label: &str) -> Result<[f64; 3], String> {
    let len = length(v);
    if !len.is_finite() || len <= 1e-12 {
        return Err(format!("{label}: expected a nonzero finite vector"));
    }
    Ok([v[0] / len, v[1] / len, v[2] / len])
}
