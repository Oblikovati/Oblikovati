// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use rustycad_tkbrep::topods::{Edge, Face, Wire};
use rustycad_tktopalgo::brep_builder_api::{BRepBuilderAPIMakeFace, BRepBuilderAPIMakePolygon};
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
                Entity::Edges(hole.edge_keys.clone()),
                format!(
                    "prism inner loop {:?}: expected accepted wire",
                    hole.edge_keys
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
    let mut maker = BRepBuilderAPIMakePolygon::new();
    let [x, y] = profile.vertices[0];
    maker
        .add_point(placement::model_point(plane, [x, y], profile.edge_keys[0])?)
        .map_err(|error| {
            Phase::Profile.native(Entity::Edge(profile.edge_keys[0]), String::new(), error)
        })?;
    let mut edges = Vec::new();
    for (index, [x, y]) in profile.vertices.iter().copied().enumerate().skip(1) {
        add_segment_point(&mut maker, [x, y], profile.edge_keys[index - 1], plane)?;
        append_edge(&maker, profile.edge_keys[index - 1], &mut edges)?;
    }
    close_profile_with_history(&mut maker, profile, &mut edges)?;
    let wire = completed_wire(&maker, profile)?;
    Ok((wire, edges))
}
fn close_profile_with_history(
    maker: &mut BRepBuilderAPIMakePolygon,
    profile: &PrismProbeLoop,
    edges: &mut Vec<(u64, Edge)>,
) -> Result<(), EvaluationFailure> {
    close_loop(maker, profile)?;
    append_edge(
        maker,
        *profile.edge_keys.last().expect("validated keys"),
        edges,
    )
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
    profile: &PrismProbeLoop,
) -> Result<(), EvaluationFailure> {
    maker.close().map_err(|error| {
        Phase::Profile.native(
            Entity::Edges(profile.edge_keys.clone()),
            format!("prism loop {:?}: expected closed wire", profile.edge_keys),
            error,
        )
    })
}
fn completed_wire(
    maker: &BRepBuilderAPIMakePolygon,
    profile: &PrismProbeLoop,
) -> Result<Wire, EvaluationFailure> {
    maker.wire().map_err(|error| {
        Phase::Profile.native(
            Entity::Edges(profile.edge_keys.clone()),
            format!(
                "prism loop {:?}: expected completed wire",
                profile.edge_keys
            ),
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
