// SPDX-License-Identifier: GPL-2.0-only
//! Read point and line records from Inventor spatial sketches without changing the source.

use std::collections::BTreeMap;

#[cfg(not(target_arch = "wasm32"))]
use corliss::{
    Document, DocumentKind,
    grammar::{DecodeStatus, DecodedNode, FieldValue},
    model::{Model, Part, view::floats},
};

#[cfg(not(target_arch = "wasm32"))]
const POINT_3D: u32 = 0xCE52_DF3E;
#[cfg(not(target_arch = "wasm32"))]
const LINE_3D: u32 = 0x8EF0_6C89;

/// A point and its segment-local identity in an Inventor document.
#[derive(Clone, Debug, PartialEq)]
pub struct InventorSpatialPoint {
    pub node: usize,
    pub position_cm: [f64; 3],
}

/// A line whose endpoints reference points in the same spatial sketch.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct InventorSpatialLine {
    pub node: usize,
    pub endpoints: [usize; 2],
}

/// The recognized point and line records of one spatial sketch.
#[derive(Clone, Debug, PartialEq)]
pub struct InventorSpatialSketch {
    pub node: usize,
    pub name: Option<String>,
    pub points: Vec<InventorSpatialPoint>,
    pub lines: Vec<InventorSpatialLine>,
}

/// Read recognized spatial points and lines without interpreting other sketch entity types.
/// Example: `let sketches = read_inventor_spatial_lines_and_points(&bytes)?;`.
#[cfg(not(target_arch = "wasm32"))]
pub fn read_inventor_spatial_lines_and_points(
    bytes: &[u8],
) -> Result<Vec<InventorSpatialSketch>, String> {
    let document =
        Document::from_bytes(bytes).map_err(|error| format!("Inventor document: {error}"))?;
    if document.kind() != Some(DocumentKind::Part) {
        return Err(format!(
            "Inventor document kind {:?}: expected a part",
            document.kind()
        ));
    }
    let model = Model::load(&document).map_err(|error| format!("Inventor part model: {error}"))?;
    let part = Part::new(&model).map_err(|error| format!("Inventor part: {error}"))?;
    collect_spatial_geometry(&part)
}

/// Read recognized spatial points and lines without interpreting other sketch entity types.
/// Example: `let sketches = read_inventor_spatial_lines_and_points(&bytes)?;`.
#[cfg(target_arch = "wasm32")]
pub fn read_inventor_spatial_lines_and_points(
    _bytes: &[u8],
) -> Result<Vec<InventorSpatialSketch>, String> {
    Err(
        "Inventor document import: unsupported on wasm32 until upstream corliss adopts web-time"
            .into(),
    )
}

#[cfg(not(target_arch = "wasm32"))]
fn collect_spatial_geometry(part: &Part<'_>) -> Result<Vec<InventorSpatialSketch>, String> {
    let mut sketches = BTreeMap::<usize, InventorSpatialSketch>::new();
    for (node, geometry) in part.view().nodes() {
        if matches!(geometry.type_id.0, POINT_3D | LINE_3D) {
            append_sketch_node(part, &mut sketches, node, geometry)?;
        }
    }
    for sketch in sketches.values() {
        validate_endpoints(sketch)?;
    }
    Ok(sketches.into_values().collect())
}

#[cfg(not(target_arch = "wasm32"))]
fn append_sketch_node(
    part: &Part<'_>,
    sketches: &mut BTreeMap<usize, InventorSpatialSketch>,
    node: usize,
    geometry: &DecodedNode,
) -> Result<(), String> {
    require_complete_node(geometry, node)?;
    let Some(owner) = spatial_owner(geometry, node)? else {
        return Ok(());
    };
    require_sketch_owner(part, node, owner)?;
    let sketch = sketches
        .entry(owner)
        .or_insert_with(|| InventorSpatialSketch {
            node: owner,
            name: part.name(owner),
            points: Vec::new(),
            lines: Vec::new(),
        });
    append_geometry(sketch, geometry, node)
}

#[cfg(not(target_arch = "wasm32"))]
fn spatial_owner(geometry: &DecodedNode, node: usize) -> Result<Option<usize>, String> {
    match geometry.get("sketch") {
        Some(FieldValue::Ref(owner, _)) => Ok(owner.index()),
        value => Err(format!(
            "Inventor spatial node {node} sketch reference {value:?}: expected a node reference"
        )),
    }
}

#[cfg(not(target_arch = "wasm32"))]
fn require_complete_node(geometry: &DecodedNode, node: usize) -> Result<(), String> {
    if geometry.status != DecodeStatus::Full {
        return Err(format!(
            "Inventor spatial node {node}: expected complete decoding, got {:?}",
            geometry.status
        ));
    }
    Ok(())
}

#[cfg(not(target_arch = "wasm32"))]
fn require_sketch_owner(part: &Part<'_>, node: usize, owner: usize) -> Result<(), String> {
    if part.view().node(owner).is_none() {
        return Err(format!(
            "Inventor spatial node {node} owner {owner}: expected an existing sketch node"
        ));
    }
    Ok(())
}

#[cfg(not(target_arch = "wasm32"))]
fn append_geometry(
    sketch: &mut InventorSpatialSketch,
    geometry: &DecodedNode,
    node: usize,
) -> Result<(), String> {
    if geometry.type_id.0 == POINT_3D {
        sketch.points.push(read_spatial_point(geometry, node)?);
    } else {
        sketch.lines.push(read_spatial_line(geometry, node)?);
    }
    Ok(())
}

#[cfg(not(target_arch = "wasm32"))]
fn read_spatial_point(geometry: &DecodedNode, node: usize) -> Result<InventorSpatialPoint, String> {
    let values = floats(geometry, "pos");
    let position_cm: [f64; 3] = values.try_into().map_err(|values: Vec<f64>| {
        format!("Inventor spatial point {node} position {values:?}: expected three coordinates")
    })?;
    if !position_cm.iter().all(|value| value.is_finite()) {
        return Err(format!(
            "Inventor spatial point {node} position {position_cm:?}: expected finite coordinates"
        ));
    }
    Ok(InventorSpatialPoint { node, position_cm })
}

#[cfg(not(target_arch = "wasm32"))]
fn read_spatial_line(geometry: &DecodedNode, node: usize) -> Result<InventorSpatialLine, String> {
    let Some(FieldValue::List(points)) = geometry.get("points") else {
        return Err(format!(
            "Inventor spatial line {node}: expected a point reference list"
        ));
    };
    if points.elements.len() != 2 {
        return Err(format!(
            "Inventor spatial line {node} has {} endpoints: expected two",
            points.elements.len()
        ));
    }
    let first = endpoint_reference(&points.elements[0], node)?;
    let second = endpoint_reference(&points.elements[1], node)?;
    Ok(InventorSpatialLine {
        node,
        endpoints: [first, second],
    })
}

#[cfg(not(target_arch = "wasm32"))]
fn endpoint_reference(fields: &[corliss::grammar::Field], node: usize) -> Result<usize, String> {
    match fields {
        [field] => match field.value {
            FieldValue::Ref(reference, _) => reference.index(),
            _ => None,
        },
        _ => None,
    }
    .ok_or_else(|| format!("Inventor spatial line {node} endpoint {fields:?}: expected one non-null point reference"))
}

fn validate_endpoints(sketch: &InventorSpatialSketch) -> Result<(), String> {
    for line in &sketch.lines {
        for endpoint in line.endpoints {
            if !sketch.points.iter().any(|point| point.node == endpoint) {
                return Err(format!(
                    "Inventor spatial line {} endpoint {endpoint}: expected a point in sketch {}",
                    line.node, sketch.node
                ));
            }
        }
    }
    Ok(())
}

/// Project recognized Inventor points and lines with document-reserved entity identities.
/// Example: `project_inventor_point_and_line_sketch(id, &source, &units, &mut ids)?`.
/// This does not establish that the source sketch has no other entity types.
pub fn project_inventor_point_and_line_sketch(
    id: oblikovati_model::document_sketch::DocumentSketchId,
    source: &InventorSpatialSketch,
    units: &oblikovati_model::units::UnitsOfMeasure,
    ids: &mut oblikovati_model::identity::RecipeIdAllocator,
) -> Result<oblikovati_model::spatial_sketch::SpatialSketch, String> {
    use oblikovati_model::spatial_sketch::SpatialSketch;
    validate_endpoints(source)?;
    let mut mapping = BTreeMap::new();
    let mut points = Vec::new();
    for point in &source.points {
        let id = reserve_node(&mut mapping, point.node, ids)?;
        points.push(oblikovati_model::spatial_sketch::SpatialSketchPoint {
            id,
            position: point
                .position_cm
                .map(|coordinate| coordinate / units.working_scale()),
        });
    }
    let mut lines = Vec::new();
    for line in &source.lines {
        let endpoints = [
            mapped_point(&mapping, line.endpoints[0])?,
            mapped_point(&mapping, line.endpoints[1])?,
        ];
        let id = reserve_node(&mut mapping, line.node, ids)?;
        lines.push(oblikovati_model::spatial_sketch::SpatialSketchLine { id, endpoints });
    }
    SpatialSketch::new(id, source.name.clone(), points, lines)
}

fn reserve_node(
    mapping: &mut BTreeMap<usize, oblikovati_model::sketch::SketchEntityId>,
    node: usize,
    ids: &mut oblikovati_model::identity::RecipeIdAllocator,
) -> Result<oblikovati_model::sketch::SketchEntityId, String> {
    if mapping.contains_key(&node) {
        return Err(format!(
            "Inventor geometry node {node}: expected a unique node identity"
        ));
    }
    let id = oblikovati_model::sketch::SketchEntityId(ids.allocate().map_err(|error| {
        format!("Inventor geometry node {node}: expected an available document entity ID: {error}")
    })?);
    mapping.insert(node, id);
    Ok(id)
}

fn mapped_point(
    mapping: &BTreeMap<usize, oblikovati_model::sketch::SketchEntityId>,
    node: usize,
) -> Result<oblikovati_model::sketch::SketchEntityId, String> {
    mapping.get(&node).copied().ok_or_else(|| {
        format!("Inventor endpoint node {node}: expected a point in the same spatial sketch")
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn missing_sketch_reference_is_refused() {
        let node = DecodedNode {
            type_id: corliss::TypeId(POINT_3D),
            status: DecodeStatus::Full,
            fields: Vec::new(),
            error: None,
        };
        let error = spatial_owner(&node, 24).unwrap_err();
        assert!(error.contains("node 24"), "{error}");
        assert!(error.contains("expected a node reference"), "{error}");
    }

    #[test]
    fn partial_spatial_geometry_is_refused() {
        let node = DecodedNode {
            type_id: corliss::TypeId(POINT_3D),
            status: DecodeStatus::Partial,
            fields: Vec::new(),
            error: Some("truncated".into()),
        };
        let error = require_complete_node(&node, 24).unwrap_err();
        assert!(error.contains("node 24"), "{error}");
        assert!(error.contains("Partial"), "{error}");
    }

    #[test]
    fn malformed_point_does_not_produce_geometry() {
        let node = DecodedNode {
            type_id: corliss::TypeId(POINT_3D),
            status: DecodeStatus::Full,
            fields: Vec::new(),
            error: None,
        };
        let mut sketch = InventorSpatialSketch {
            node: 12,
            name: None,
            points: Vec::new(),
            lines: Vec::new(),
        };
        let error = append_geometry(&mut sketch, &node, 24).unwrap_err();
        assert!(error.contains("point 24"), "{error}");
        assert!(sketch.points.is_empty());
    }

    #[test]
    fn missing_line_endpoint_is_refused() {
        let error = endpoint_reference(&[], 21).unwrap_err();
        assert!(error.contains("line 21"), "{error}");
        assert!(
            error.contains("expected one non-null point reference"),
            "{error}"
        );
    }

    #[test]
    fn line_endpoints_must_exist_in_the_same_sketch() {
        let sketch = InventorSpatialSketch {
            node: 12,
            name: Some("Sketch".into()),
            points: vec![InventorSpatialPoint {
                node: 20,
                position_cm: [0.; 3],
            }],
            lines: vec![InventorSpatialLine {
                node: 21,
                endpoints: [20, 99],
            }],
        };
        let error = validate_endpoints(&sketch).unwrap_err();
        assert!(error.contains("endpoint 99"), "{error}");
        assert!(error.contains("sketch 12"), "{error}");
    }
}
