// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use martensite_tkbrep::topods::{Edge, Face, Wire};
use martensite_tkmath::gp::{Axis2, Circle3, Direction3};
use martensite_tktopalgo::brep_builder_api::{
    BRepBuilderAPIMakeEdge, BRepBuilderAPIMakeFace, BRepBuilderAPIMakePolygon,
    BRepBuilderAPIMakeWire,
};
pub(super) struct BuiltProfile {
    pub face: Face,
    pub edges: Vec<(u64, Edge)>,
}
pub(super) fn build_profile(
    definition: &PrismProbeDefinition,
    plane: SketchPlane,
) -> Result<BuiltProfile, EvaluationFailure> {
    let (outer, mut edges) = build_loop(&definition.outer, plane)?;
    let mut maker = BRepBuilderAPIMakeFace::from_wire(&outer, true).map_err(|error| {
        Phase::Profile.native(
            Entity::Body,
            "prism outer face: expected planar construction".into(),
            error,
        )
    })?;
    append_profile_holes(&mut maker, &definition.holes, plane, &mut edges)?;
    let face = completed_profile_face(&maker)?;
    crate::checked_shape::require_geometry(&face, "prism profile")?;
    Ok(BuiltProfile { face, edges })
}
fn append_profile_holes(
    maker: &mut BRepBuilderAPIMakeFace,
    holes: &[PrismProbeLoop],
    plane: SketchPlane,
    edges: &mut Vec<(u64, Edge)>,
) -> Result<(), EvaluationFailure> {
    for hole in holes {
        let (wire, inner_edges) = build_loop(hole, plane)?;
        maker.add(&wire).map_err(|cause| {
            Phase::Profile.native(
                Entity::Edges(hole.edge_keys()),
                format!(
                    "prism inner loop {:?}: expected accepted wire",
                    hole.edge_keys()
                ),
                cause,
            )
        })?;
        edges.extend(inner_edges);
    }
    Ok(())
}
fn completed_profile_face(maker: &BRepBuilderAPIMakeFace) -> Result<Face, EvaluationFailure> {
    maker.face().map_err(|cause| {
        Phase::Profile.native(
            Entity::Body,
            "prism profile: expected constructed face".into(),
            cause,
        )
    })
}
fn build_loop(
    profile: &PrismProbeLoop,
    plane: SketchPlane,
) -> Result<(Wire, Vec<(u64, Edge)>), EvaluationFailure> {
    match profile {
        PrismProbeLoop::Polygon {
            edge_keys,
            vertices,
        } => build_polygon_loop(edge_keys, vertices, plane),
        PrismProbeLoop::Circle {
            edge_key,
            center,
            radius,
        } => build_circle_loop(*edge_key, *center, *radius, plane),
    }
}
fn build_polygon_loop(
    edge_keys: &[u64],
    vertices: &[[f64; 2]],
    plane: SketchPlane,
) -> Result<(Wire, Vec<(u64, Edge)>), EvaluationFailure> {
    let mut maker = BRepBuilderAPIMakePolygon::new();
    let [x, y] = vertices[0];
    maker
        .add_point(placement::model_point(plane, [x, y], edge_keys[0])?)
        .map_err(|error| Phase::Profile.native(Entity::Edge(edge_keys[0]), String::new(), error))?;
    let mut edges = Vec::new();
    for (index, [x, y]) in vertices.iter().copied().enumerate().skip(1) {
        add_segment_point(&mut maker, [x, y], edge_keys[index - 1], plane)?;
        append_edge(&maker, edge_keys[index - 1], &mut edges)?;
    }
    close_profile_with_history(&mut maker, edge_keys, &mut edges)?;
    let wire = completed_polygon_wire(&maker, edge_keys)?;
    Ok((wire, edges))
}
fn build_circle_loop(
    edge_key: u64,
    center: [f64; 2],
    radius: f64,
    plane: SketchPlane,
) -> Result<(Wire, Vec<(u64, Edge)>), EvaluationFailure> {
    let origin = placement::model_point(plane, center, edge_key)?;
    let normal = plane.normal();
    let direction = Direction3::new(normal[0], normal[1], normal[2]).map_err(|cause| {
        Phase::Profile.native(
            Entity::Edge(edge_key),
            "prism circle: expected usable sketch-plane normal".into(),
            cause,
        )
    })?;
    let circle = Circle3::new(
        Axis2::new(origin, direction).map_err(|cause| {
            Phase::Profile.native(
                Entity::Edge(edge_key),
                "prism circle: expected usable local frame".into(),
                cause,
            )
        })?,
        radius,
    )
    .map_err(|cause| {
        Phase::Profile.native(
            Entity::Edge(edge_key),
            format!("prism circle radius {radius:?}: expected positive radius"),
            cause,
        )
    })?;
    let edge = BRepBuilderAPIMakeEdge::from_circ(circle)
        .map_err(|cause| {
            Phase::Profile.native(
                Entity::Edge(edge_key),
                "prism circle: expected exact native edge".into(),
                cause,
            )
        })?
        .edge()
        .map_err(|cause| {
            Phase::Profile.native(
                Entity::Edge(edge_key),
                "prism circle: expected completed native edge".into(),
                cause,
            )
        })?;
    let wire = BRepBuilderAPIMakeWire::from_edge(&edge)
        .map_err(|cause| {
            Phase::Profile.native(
                Entity::Edge(edge_key),
                "prism circle: expected closed native wire".into(),
                cause,
            )
        })?
        .wire()
        .map_err(|cause| {
            Phase::Profile.native(
                Entity::Edge(edge_key),
                "prism circle: expected completed native wire".into(),
                cause,
            )
        })?;
    Ok((wire, vec![(edge_key, edge)]))
}
fn close_profile_with_history(
    maker: &mut BRepBuilderAPIMakePolygon,
    edge_keys: &[u64],
    edges: &mut Vec<(u64, Edge)>,
) -> Result<(), EvaluationFailure> {
    close_loop(maker, edge_keys)?;
    append_edge(maker, *edge_keys.last().expect("validated keys"), edges)
}
fn add_segment_point(
    maker: &mut BRepBuilderAPIMakePolygon,
    point: [f64; 2],
    key: u64,
    plane: SketchPlane,
) -> Result<(), EvaluationFailure> {
    maker
        .add_point(placement::model_point(plane, point, key)?)
        .map_err(|error| {
            Phase::Profile.native(
                Entity::Edge(key),
                format!("prism edge {key}, point {point:?}: expected connected point"),
                error,
            )
        })
}
fn close_loop(
    maker: &mut BRepBuilderAPIMakePolygon,
    edge_keys: &[u64],
) -> Result<(), EvaluationFailure> {
    maker.close().map_err(|error| {
        Phase::Profile.native(
            Entity::Edges(edge_keys.to_vec()),
            format!("prism loop {edge_keys:?}: expected closed wire"),
            error,
        )
    })
}
fn completed_polygon_wire(
    maker: &BRepBuilderAPIMakePolygon,
    edge_keys: &[u64],
) -> Result<Wire, EvaluationFailure> {
    maker.wire().map_err(|error| {
        Phase::Profile.native(
            Entity::Edges(edge_keys.to_vec()),
            format!("prism loop {:?}: expected completed wire", edge_keys),
            error,
        )
    })
}
fn append_edge(
    maker: &BRepBuilderAPIMakePolygon,
    key: u64,
    edges: &mut Vec<(u64, Edge)>,
) -> Result<(), EvaluationFailure> {
    if !maker.added() || maker.edge().is_null() {
        return Err(Phase::Profile.refused(Kind::InvalidGeometry, Entity::Edge(key), format!(
            "prism edge {key}: expected a distinct constructed segment; kernel collapsed the input"
        )));
    }
    edges.push((key, maker.edge().clone()));
    Ok(())
}
