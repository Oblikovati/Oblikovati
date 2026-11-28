// SPDX-License-Identifier: GPL-2.0-only
//! Exact analytic vertex, edge, and revolved-surface axis queries for Work Axis (#3918).
use super::*;
use martensite_tkbrep::{
    brep::BRepTool,
    brep_adaptor::{BRepAdaptorCurve, BRepAdaptorSurface},
    topexp::Explorer,
    topods,
};
use martensite_tkg3d::{
    adaptor3d::{Adaptor3dCurve, Adaptor3dSurface},
    top_abs::ShapeEnum,
};
use martensite_tkmath::geom_abs::{CurveType, SurfaceType};
use oblikovati_model::feature::work_axis::WorkAxisLine;

/// One exact solid vertex hit against a forward viewport ray.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct NamedVertexHit {
    pub position: [f64; 3],
    pub distance: f64,
    pub reach: f64,
}

/// Distinguishes straight edges from circular/elliptical edges when resolving an axis.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum AnalyticEdgeAxisKind {
    Straight,
    CircularOrElliptical,
}

/// Exact analytic axis extracted from a named solid edge.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct AnalyticEdgeAxis {
    pub kind: AnalyticEdgeAxisKind,
    pub line: WorkAxisLine,
}

impl NamedSolid {
    /// Query exact solid vertices within `tolerance` of a forward view ray, ordered nearest first.
    /// Example: `body.vertex_hits_on_ray(&history, document, [origin, direction], tolerance)?`.
    pub fn vertex_hits_on_ray(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        [origin, direction]: [[f64; 3]; 2],
        tolerance: f64,
    ) -> Result<Vec<NamedVertexHit>, String> {
        if !tolerance.is_finite() || tolerance <= 0.0 {
            return Err(format!(
                "vertex pick tolerance {tolerance}: expected a finite positive length"
            ));
        }
        let shape = self.native_shape(history, document)?;
        vertex_hits_on_shape(shape, [origin, direction], tolerance)
    }

    /// Return all unique exact vertex positions on this solid in deterministic order.
    /// Example: `body.vertex_positions(&history, document)?`.
    pub fn vertex_positions(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Vec<[f64; 3]>, String> {
        let shape = self.native_shape(history, document)?;
        unique_shape_vertices(shape)
    }

    /// Extract the analytic axis line of a straight, circular, or elliptical named edge.
    /// Example: `body.edge_axis(&history, document, key)?`.
    pub fn edge_axis(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<Option<AnalyticEdgeAxis>, String> {
        let shape = self.native_edge(history, document, key)?;
        analytic_edge_axis(shape)
    }

    /// Extract the exact revolution axis of a cylindrical, conical, toroidal, or revolved face.
    /// Example: `body.revolved_face_axis(&history, document, key)?`.
    pub fn revolved_face_axis(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<Option<WorkAxisLine>, String> {
        let shape = self.native_face(history, document, key)?;
        analytic_revolved_face_axis(shape)
    }
}

fn unique_shape_vertices(shape: &Shape) -> Result<Vec<[f64; 3]>, String> {
    let mut explorer = Explorer::new(shape, ShapeEnum::Vertex, ShapeEnum::Shape)
        .map_err(|cause| cause.to_string())?;
    let mut points: Vec<[f64; 3]> = Vec::new();
    while explorer.more() {
        let current = explorer.current();
        let vertex = topods::vertex(current).map_err(|cause| cause.to_string())?;
        let pnt = BRepTool::pnt(&vertex).map_err(|cause| cause.to_string())?;
        let coords = [pnt.x(), pnt.y(), pnt.z()];
        if coords.iter().all(|v| v.is_finite())
            && !points
                .iter()
                .any(|existing| dist_sq(*existing, coords) <= 1e-16)
        {
            points.push(coords);
        }
        explorer.next().map_err(|cause| cause.to_string())?;
    }
    points.sort_by(|a, b| {
        a[0].total_cmp(&b[0])
            .then(a[1].total_cmp(&b[1]))
            .then(a[2].total_cmp(&b[2]))
    });
    Ok(points)
}

fn vertex_hits_on_shape(
    shape: &Shape,
    [origin, direction]: [[f64; 3]; 2],
    tolerance: f64,
) -> Result<Vec<NamedVertexHit>, String> {
    let length = direction[0].hypot(direction[1]).hypot(direction[2]);
    if !length.is_finite() || length == 0.0 || origin.iter().any(|v| !v.is_finite()) {
        return Err(format!(
            "vertex ray {origin:?} {direction:?}: expected finite origin and nonzero direction"
        ));
    }
    let unit = direction.map(|v| v / length);
    let mut hits = Vec::new();
    for position in unique_shape_vertices(shape)? {
        let offset = [
            position[0] - origin[0],
            position[1] - origin[1],
            position[2] - origin[2],
        ];
        let reach = offset[0] * unit[0] + offset[1] * unit[1] + offset[2] * unit[2];
        if !reach.is_finite() || reach < 0.0 {
            continue;
        }
        let projected = [
            origin[0] + unit[0] * reach,
            origin[1] + unit[1] * reach,
            origin[2] + unit[2] * reach,
        ];
        let distance = dist_sq(position, projected).sqrt();
        if distance.is_finite() && distance <= tolerance {
            hits.push(NamedVertexHit {
                position,
                distance,
                reach,
            });
        }
    }
    hits.sort_by(|a, b| {
        a.distance
            .total_cmp(&b.distance)
            .then(a.reach.total_cmp(&b.reach))
    });
    Ok(hits)
}

fn analytic_edge_axis(shape: &Shape) -> Result<Option<AnalyticEdgeAxis>, String> {
    let edge = topods::edge(shape).map_err(|cause| cause.to_string())?;
    let curve = BRepAdaptorCurve::with_edge(&edge).map_err(|cause| cause.to_string())?;
    let kind = curve.get_type().map_err(|cause| cause.to_string())?;
    match kind {
        CurveType::Line => {
            let line = curve.line().map_err(|cause| cause.to_string())?;
            let origin = point_coords(line.location());
            let direction = *line.direction().coordinates();
            let axis = WorkAxisLine::new(origin, direction).map_err(|cause| cause.to_string())?;
            Ok(Some(AnalyticEdgeAxis {
                kind: AnalyticEdgeAxisKind::Straight,
                line: axis,
            }))
        }
        CurveType::Circle => {
            let circle = curve.circle().map_err(|cause| cause.to_string())?;
            let origin = point_coords(circle.location());
            let direction = *circle.axis().direction().coordinates();
            let axis = WorkAxisLine::new(origin, direction).map_err(|cause| cause.to_string())?;
            Ok(Some(AnalyticEdgeAxis {
                kind: AnalyticEdgeAxisKind::CircularOrElliptical,
                line: axis,
            }))
        }
        CurveType::Ellipse => {
            let ellipse = curve.ellipse().map_err(|cause| cause.to_string())?;
            let origin = point_coords(ellipse.location());
            let direction = *ellipse.axis().direction().coordinates();
            let axis = WorkAxisLine::new(origin, direction).map_err(|cause| cause.to_string())?;
            Ok(Some(AnalyticEdgeAxis {
                kind: AnalyticEdgeAxisKind::CircularOrElliptical,
                line: axis,
            }))
        }
        _ => Ok(None),
    }
}

fn analytic_revolved_face_axis(shape: &Shape) -> Result<Option<WorkAxisLine>, String> {
    let face = topods::face(shape).map_err(|cause| cause.to_string())?;
    let adaptor = BRepAdaptorSurface::with_face(&face, false).map_err(|cause| cause.to_string())?;
    let surface = adaptor.get_type().map_err(|cause| cause.to_string())?;
    let axis = match surface {
        SurfaceType::Cylinder => adaptor.cylinder().map_err(|e| e.to_string())?.axis(),
        SurfaceType::Cone => adaptor.cone().map_err(|e| e.to_string())?.axis(),
        SurfaceType::Torus => adaptor.torus().map_err(|e| e.to_string())?.axis(),
        SurfaceType::SurfaceOfRevolution => {
            adaptor.axe_of_revolution().map_err(|e| e.to_string())?
        }
        _ => return Ok(None),
    };
    let origin = point_coords(axis.location());
    let direction = *axis.direction().coordinates();
    WorkAxisLine::new(origin, direction)
        .map(Some)
        .map_err(|cause| cause.to_string())
}

fn point_coords(point: martensite_tkmath::gp::Point3) -> [f64; 3] {
    [point.x(), point.y(), point.z()]
}

fn dist_sq(a: [f64; 3], b: [f64; 3]) -> f64 {
    (a[0] - b[0]).powi(2) + (a[1] - b[1]).powi(2) + (a[2] - b[2]).powi(2)
}

#[cfg(test)]
mod tests {
    use super::*;
    use martensite_tkmath::gp::{Axis2, Circle3, Direction3, Point3};
    use martensite_tkprim::brep_prim_api::{
        BRepPrimAPIMakeBox, BRepPrimAPIMakeCylinder, BRepPrimAPIMakeOneAxis,
    };
    use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeEdge;

    #[test]
    fn box_vertices_and_straight_edge_axes_resolve_analytically() {
        let mut maker = BRepPrimAPIMakeBox::from_sizes(2.0, 3.0, 4.0).unwrap();
        let solid: Shape = maker.solid().unwrap().into();
        let vertices = unique_shape_vertices(&solid).unwrap();
        assert_eq!(vertices.len(), 8);
        let hits = vertex_hits_on_shape(&solid, [[2.0, 3.0, 10.0], [0.0, 0.0, -1.0]], 0.1).unwrap();
        assert_eq!(hits[0].position, [2.0, 3.0, 4.0]);
        assert!(hits[0].distance < 1e-12);

        let line_edge: Shape = BRepBuilderAPIMakeEdge::from_points(
            Point3::new(1.0, 2.0, 0.0),
            Point3::new(1.0, 2.0, 5.0),
        )
        .unwrap()
        .edge()
        .unwrap()
        .into();
        let resolved = analytic_edge_axis(&line_edge).unwrap().unwrap();
        assert_eq!(resolved.kind, AnalyticEdgeAxisKind::Straight);
        assert!((resolved.line.direction()[2].abs() - 1.0).abs() < 1e-12);
    }

    #[test]
    fn circular_edge_and_cylindrical_face_resolve_exact_center_axes() {
        let axes = Axis2::new(
            Point3::new(3.0, -2.0, 1.0),
            Direction3::new(0.0, 0.0, 1.0).unwrap(),
        )
        .unwrap();
        let circle = Circle3::new(axes, 0.6).unwrap();
        let edge: Shape = BRepBuilderAPIMakeEdge::from_circ(circle)
            .unwrap()
            .edge()
            .unwrap()
            .into();
        let resolved = analytic_edge_axis(&edge).unwrap().unwrap();
        assert_eq!(resolved.kind, AnalyticEdgeAxisKind::CircularOrElliptical);
        assert_eq!(resolved.line.origin(), [3.0, -2.0, 1.0]);
        assert_eq!(resolved.line.direction(), [0.0, 0.0, 1.0]);

        let mut cyl = BRepPrimAPIMakeCylinder::with_axes(axes, 0.6, 2.0);
        let shape: Shape = cyl.solid().unwrap().into();
        let mut explorer = Explorer::new(&shape, ShapeEnum::Face, ShapeEnum::Shape).unwrap();
        let mut revolved_count = 0;
        let mut planar_count = 0;
        while explorer.more() {
            let face = explorer.current();
            match analytic_revolved_face_axis(face).unwrap() {
                Some(line) => {
                    revolved_count += 1;
                    assert!((line.origin()[0] - 3.0).abs() < 1e-12);
                    assert!((line.origin()[1] + 2.0).abs() < 1e-12);
                    assert!((line.direction()[2].abs() - 1.0).abs() < 1e-12);
                }
                None => planar_count += 1,
            }
            explorer.next().unwrap();
        }
        assert_eq!(revolved_count, 1);
        assert_eq!(planar_count, 2);
    }
}
