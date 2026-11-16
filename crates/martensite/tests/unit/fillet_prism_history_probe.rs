// SPDX-License-Identifier: GPL-2.0-only
//! Exact copied-prism fillet history probe (#4045).
use super::*;
use martensite_tkbrep::{
    topexp,
    topods::{self, Shape},
    toptools::IndexedMapOfShape,
};
use martensite_tkfillet::{brep_fillet_api::BRepFilletAPIMakeFillet, chfi3d::FilletShape};
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;

fn entities(body: &Shape, kind: ShapeEnum) -> Vec<Shape> {
    let mut map = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(body, kind, &mut map).unwrap();
    (1..=map.extent())
        .map(|index| map.find_key(index).unwrap().clone())
        .collect()
}

fn box_prism() -> PrismProbeSnapshot {
    retain_planar_prism(&PrismProbeDefinition {
        outer: PrismProbeLoop::polygon(
            vec![1, 2, 3, 4],
            vec![[0., 0.], [10., 0.], [10., 10.], [0., 10.]],
        ),
        holes: Vec::new(),
        displacement: [0., 0., 10.],
    })
    .unwrap()
}

#[test]
fn copied_prism_edge_fillet_retains_source_and_attributes_result_faces() {
    let source = box_prism();
    let (copied, copied_edge) = copy_prism_and_edge(&source.shape);
    let mut fillet = BRepFilletAPIMakeFillet::new(&copied, FilletShape::Rational).unwrap();
    fillet
        .add_with_radius(2., &topods::edge(&copied_edge).unwrap())
        .unwrap();
    let result = fillet.shape().unwrap().clone();
    verify_copy_volumes(&source.shape, &result);
    verify_copied_face_history(&mut fillet, &copied, &copied_edge, &result);
}

fn copy_prism_and_edge(source: &Shape) -> (Shape, Shape) {
    let input_edges = entities(source, ShapeEnum::Edge);
    let mut copier = crate::checked_shape::copy_shape(source, "fillet probe").unwrap();
    let mapped = copier.modified(&input_edges[0]).unwrap();
    assert_eq!(mapped.len(), 1, "copied edge: expected one source mapping");
    let edge = mapped.first().unwrap().clone();
    (copier.shape().unwrap().clone(), edge)
}

fn verify_copy_volumes(source: &Shape, result: &Shape) {
    crate::checked_shape::require_geometry(result, "fillet probe result").unwrap();
    let actual = crate::checked_shape::analytic_properties(result, "fillet probe result").unwrap();
    let original = crate::checked_shape::analytic_properties(source, "fillet probe input").unwrap();
    let expected = 1000. - (4. - std::f64::consts::PI) * 10.;
    assert!((actual.volume - expected).abs() < 1e-6);
    assert!((original.volume - 1000.).abs() < 1e-6);
}

fn verify_copied_face_history(
    fillet: &mut BRepFilletAPIMakeFillet,
    copied: &Shape,
    edge: &Shape,
    result: &Shape,
) {
    let output = entities(result, ShapeEnum::Face);
    let mut attributed: Vec<_> = fillet.generated(edge).unwrap().iter().cloned().collect();
    assert!(
        !attributed.is_empty(),
        "copied edge: expected generated face history"
    );
    for face in entities(copied, ShapeEnum::Face) {
        attributed.extend(fillet.modified(&face).unwrap().iter().cloned());
        if output.iter().any(|item| item.is_same(&face)) {
            attributed.push(face);
        }
    }
    for face in &output {
        assert!(
            attributed.iter().any(|item| item.is_same(face)),
            "result face: expected generated, modified, or retained source"
        );
    }
}
