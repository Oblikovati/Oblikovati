// SPDX-License-Identifier: GPL-2.0-only
//! Exact copied-prism blend history probes (#4045, #4046).
use super::*;
use martensite_tkbrep::{
    topexp,
    topods::{self, Shape},
    toptools::IndexedMapOfShape,
};
use martensite_tkfillet::{
    brep_fillet_api::{BRepFilletAPIMakeChamfer, BRepFilletAPIMakeFillet},
    chfi3d::FilletShape,
};
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
    let (copied, copied_edge) = copy_prism_and_edge(&source.shape, 0);
    let mut fillet = BRepFilletAPIMakeFillet::new(&copied, FilletShape::Rational).unwrap();
    fillet
        .add_with_radius(2., &topods::edge(&copied_edge).unwrap())
        .unwrap();
    let result = fillet.shape().unwrap().clone();
    verify_copy_volumes(&source.shape, &result);
    verify_copied_face_history(&mut fillet, &copied, &copied_edge, &result);
}

#[test]
fn copied_prism_chamfer_preserves_input_and_attributes_planar_faces() {
    let source = box_prism();
    assert_eq!(entities(&source.shape, ShapeEnum::Edge).len(), 12);
    for edge_index in 0..12 {
        verify_chamfer_edge(&source.shape, edge_index);
    }
}

fn verify_chamfer_edge(source: &Shape, edge_index: usize) {
    let (copied, copied_edge) = copy_prism_and_edge(source, edge_index);
    let mut chamfer = BRepFilletAPIMakeChamfer::new(&copied).unwrap();
    chamfer
        .add_with_dist(2., &topods::edge(&copied_edge).unwrap())
        .unwrap();
    let result = chamfer.shape().unwrap().clone();
    verify_chamfer_geometry(source, &result);
    verify_copied_face_history(&mut chamfer, &copied, &copied_edge, &result);
}

fn verify_chamfer_geometry(source: &Shape, result: &Shape) {
    crate::checked_shape::require_geometry(result, "chamfer probe result").unwrap();
    let actual = crate::checked_shape::analytic_properties(result, "chamfer probe result").unwrap();
    let original =
        crate::checked_shape::analytic_properties(source, "chamfer probe input").unwrap();
    assert!((actual.volume - 980.).abs() < 1e-6);
    assert!((original.volume - 1000.).abs() < 1e-6);
    verify_chamfer_faces(result);
}

fn verify_chamfer_faces(result: &Shape) {
    let mut areas = entities(result, ShapeEnum::Face)
        .iter()
        .map(chamfer_face_area)
        .collect::<Vec<_>>();
    areas.sort_by(f64::total_cmp);
    let expected = [20. * 2_f64.sqrt(), 80., 80., 98., 98., 100., 100.];
    assert_eq!(areas.len(), expected.len());
    for (actual, target) in areas.iter().zip(expected) {
        assert!(
            (actual - target).abs() < 1e-6,
            "face area {actual}: expected {target}"
        );
    }
}

fn chamfer_face_area(face: &Shape) -> f64 {
    use martensite_tkbrep::brep_adaptor::BRepAdaptorSurface;
    use martensite_tkg3d::adaptor3d::Adaptor3dSurface;
    use martensite_tkgeombase::gprop::GProps;
    use martensite_tkmath::geom_abs::SurfaceType;
    use martensite_tktopalgo::brep_gprop::BRepGProp;
    let surface = BRepAdaptorSurface::with_face(&topods::face(face).unwrap(), false).unwrap();
    assert_eq!(surface.get_type().unwrap(), SurfaceType::Plane);
    assert_eq!(entities(face, ShapeEnum::Wire).len(), 1);
    let mut properties = GProps::new();
    BRepGProp::surface_properties(face, &mut properties, false, false).unwrap();
    properties.mass()
}

fn copy_prism_and_edge(source: &Shape, edge_index: usize) -> (Shape, Shape) {
    let input_edges = entities(source, ShapeEnum::Edge);
    let mut copier = crate::checked_shape::copy_shape(source, "blend probe").unwrap();
    let mapped = copier.modified(&input_edges[edge_index]).unwrap();
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
    fillet: &mut impl BRepBuilderAPIMakeShape,
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
