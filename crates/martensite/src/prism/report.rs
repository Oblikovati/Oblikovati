// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use profile::BuiltProfile;
use rustycad_tkbrep::{topexp, topods::Shape, toptools::IndexedMapOfShape};
use rustycad_tkg3d::top_abs::ShapeEnum;
use rustycad_tkgeombase::gprop::GProps;
use rustycad_tkprim::brep_prim_api::BRepPrimAPIMakeSweep;
use rustycad_tktopalgo::brep_gprop::BRepGProp;
pub(super) fn construction_lineage(
    maker: &mut BRepPrimAPIMakePrism,
    profile: &BuiltProfile,
    shape: &Shape,
) -> Result<Vec<(PrismFaceRole, Shape)>, String> {
    let mut lineage = cap_lineage(maker)?;
    for (key, edge) in &profile.edges {
        lineage.push((
            PrismFaceRole::ProfileEdge(*key),
            generated_face(maker, *key, edge)?,
        ));
    }
    require_face_coverage(shape, &lineage)?;
    Ok(lineage)
}
pub(super) fn checked_report(
    shape: &Shape,
    lineage: &[(PrismFaceRole, Shape)],
) -> Result<PrismProbeReport, String> {
    require_face_coverage(shape, lineage)?;
    let faces = lineage
        .iter()
        .map(|(role, shape)| face_report(*role, shape))
        .collect::<Result<Vec<_>, _>>()?;
    let properties = crate::analytic_shape_properties(shape, "prism result")?;
    Ok(PrismProbeReport {
        properties,
        faces,
        topology: topology_summary(shape)?,
    })
}
fn cap_lineage(maker: &mut BRepPrimAPIMakePrism) -> Result<Vec<(PrismFaceRole, Shape)>, String> {
    Ok(vec![
        (
            PrismFaceRole::Start,
            maker.first_shape().map_err(|error| error.to_string())?,
        ),
        (
            PrismFaceRole::End,
            maker.last_shape().map_err(|error| error.to_string())?,
        ),
    ])
}
fn generated_face(
    maker: &mut BRepPrimAPIMakePrism,
    key: u64,
    edge: &Shape,
) -> Result<Shape, String> {
    let generated = maker
        .generated(edge)
        .map_err(|error| format!("prism edge {key}: expected generated face history: {error}"))?;
    if generated.len() != 1 {
        return Err(format!(
            "prism edge {key} generated {} shapes: expected exactly one side face",
            generated.len()
        ));
    }
    let shape = generated.first().expect("one generated shape").clone();
    rustycad_tkbrep::topods::face(&shape)
        .map_err(|error| format!("prism edge {key}: expected generated face: {error}"))?;
    Ok(shape)
}
fn require_face_coverage(shape: &Shape, lineage: &[(PrismFaceRole, Shape)]) -> Result<(), String> {
    let mut faces = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(shape, ShapeEnum::Face, &mut faces)
        .map_err(|error| error.to_string())?;
    if faces.extent() != lineage.len() {
        return Err(format!(
            "prism faces={}, lineage={}: expected complete face coverage",
            faces.extent(),
            lineage.len()
        ));
    }
    for index in 1..=faces.extent() {
        let face = faces.find_key(index).map_err(|error| error.to_string())?;
        require_single_role(face, index, lineage)?;
    }
    Ok(())
}
fn require_single_role(
    face: &Shape,
    index: usize,
    lineage: &[(PrismFaceRole, Shape)],
) -> Result<(), String> {
    let parents = lineage
        .iter()
        .filter(|(_, generated)| generated.is_same(face))
        .count();
    if parents == 1 {
        return Ok(());
    }
    Err(format!(
        "prism face {index} history count={parents}: expected exactly one construction role"
    ))
}
pub(super) fn face_report(role: PrismFaceRole, shape: &Shape) -> Result<PrismFaceProbe, String> {
    let mut properties = GProps::new();
    BRepGProp::surface_properties(shape, &mut properties, false, false)
        .map_err(|error| format!("prism face {role:?}: expected analytic area: {error}"))?;
    let centroid = properties.centre_of_mass();
    let centroid = [centroid.x(), centroid.y(), centroid.z()];
    let area = properties.mass();
    require_finite_face(role, area, centroid)?;
    let mut wires = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(shape, ShapeEnum::Wire, &mut wires)
        .map_err(|error| error.to_string())?;
    Ok(PrismFaceProbe {
        role,
        area,
        centroid,
        loops: wires.extent(),
    })
}

fn require_finite_face(role: PrismFaceRole, area: f64, centroid: [f64; 3]) -> Result<(), String> {
    if !area.is_finite() || area <= 0. || centroid.iter().any(|value| !value.is_finite()) {
        return Err(format!(
            "prism face {role:?}, area={area}, centroid={centroid:?}: expected finite positive analytic area and centroid"
        ));
    }
    Ok(())
}

#[cfg(test)]
#[path = "../../tests/prism/report.rs"]
mod tests;

fn topology_summary(shape: &Shape) -> Result<PrismTopologySummary, String> {
    Ok(PrismTopologySummary {
        solid: shape.shape_type().map_err(|cause| cause.to_string())? == ShapeEnum::Solid,
        faces: entity_count(shape, ShapeEnum::Face)?,
        edges: entity_count(shape, ShapeEnum::Edge)?,
        vertices: entity_count(shape, ShapeEnum::Vertex)?,
        shells: entity_count(shape, ShapeEnum::Shell)?,
        wires: entity_count(shape, ShapeEnum::Wire)?,
    })
}
fn entity_count(shape: &Shape, kind: ShapeEnum) -> Result<usize, String> {
    let mut entities = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(shape, kind, &mut entities).map_err(|cause| {
        format!("prism topology {kind:?}: expected unique native entities: {cause}")
    })?;
    Ok(entities.extent())
}
