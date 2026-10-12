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
) -> Result<BuiltProfile, String> {
    let (outer, mut edges) = build_loop(&definition.outer, plane)?;
    let mut maker = BRepBuilderAPIMakeFace::from_wire(&outer, true)
        .map_err(|error| format!("prism outer face: expected planar construction: {error}"))?;
    for hole in &definition.holes {
        let (wire, inner_edges) = build_loop(hole, plane)?;
        maker.add(&wire).map_err(|error| {
            format!(
                "prism inner loop {:?}: expected accepted wire: {error}",
                hole.edge_keys
            )
        })?;
        edges.extend(inner_edges);
    }
    let face = maker
        .face()
        .map_err(|error| format!("prism profile: expected constructed face: {error}"))?;
    crate::require_geometric_shape(&face, "prism profile")?;
    Ok(BuiltProfile { face, edges })
}
fn build_loop(
    profile: &PrismProbeLoop,
    plane: SketchPlane,
) -> Result<(Wire, Vec<(u64, Edge)>), String> {
    let mut maker = BRepBuilderAPIMakePolygon::new();
    let [x, y] = profile.vertices[0];
    maker
        .add_point(placement::model_point(plane, [x, y])?)
        .map_err(|error| error.to_string())?;
    let mut edges = Vec::new();
    for (index, [x, y]) in profile.vertices.iter().copied().enumerate().skip(1) {
        add_segment_point(&mut maker, [x, y], profile.edge_keys[index - 1], plane)?;
        append_edge(&maker, profile.edge_keys[index - 1], &mut edges)?;
    }
    close_loop(&mut maker, profile)?;
    append_edge(
        &maker,
        *profile.edge_keys.last().expect("validated keys"),
        &mut edges,
    )?;
    let wire = completed_wire(&maker, profile)?;
    Ok((wire, edges))
}
fn add_segment_point(
    maker: &mut BRepBuilderAPIMakePolygon,
    point: [f64; 2],
    key: u64,
    plane: SketchPlane,
) -> Result<(), String> {
    maker
        .add_point(placement::model_point(plane, point)?)
        .map_err(|error| {
            format!("prism edge {key}, point {point:?}: expected connected point: {error}")
        })
}
fn close_loop(
    maker: &mut BRepBuilderAPIMakePolygon,
    profile: &PrismProbeLoop,
) -> Result<(), String> {
    maker.close().map_err(|error| {
        format!(
            "prism loop {:?}: expected closed wire: {error}",
            profile.edge_keys
        )
    })
}
fn completed_wire(
    maker: &BRepBuilderAPIMakePolygon,
    profile: &PrismProbeLoop,
) -> Result<Wire, String> {
    maker.wire().map_err(|error| {
        format!(
            "prism loop {:?}: expected completed wire: {error}",
            profile.edge_keys
        )
    })
}
fn append_edge(
    maker: &BRepBuilderAPIMakePolygon,
    key: u64,
    edges: &mut Vec<(u64, Edge)>,
) -> Result<(), String> {
    if !maker.added() || maker.edge().is_null() {
        return Err(format!(
            "prism edge {key}: expected a distinct constructed segment; kernel collapsed the input"
        ));
    }
    edges.push((key, maker.edge().clone()));
    Ok(())
}
