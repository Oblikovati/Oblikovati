// SPDX-License-Identifier: GPL-2.0-only
//! Probe the pinned kernel's fillet face and edge history (#4045).
use martensite_tkbrep::{
    topexp,
    topods::{self, Shape},
    toptools::IndexedMapOfShape,
};
use martensite_tkfillet::{brep_fillet_api::BRepFilletAPIMakeFillet, chfi3d::FilletShape};
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkgeombase::gprop::GProps;
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeBox;
use martensite_tktopalgo::{
    brep_builder_api::BRepBuilderAPIMakeShape, brep_check::BRepCheckAnalyzer, brep_gprop::BRepGProp,
};

fn shapes(body: &Shape, kind: ShapeEnum) -> Vec<Shape> {
    let mut map = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(body, kind, &mut map).unwrap();
    (1..=map.extent())
        .map(|index| map.find_key(index).unwrap().clone())
        .collect()
}

#[test]
fn fillet_face_history_covers_a_valid_box_result() {
    let original = BRepPrimAPIMakeBox::from_sizes(10., 10., 10.)
        .unwrap()
        .shape()
        .unwrap()
        .clone();
    let edges = shapes(&original, ShapeEnum::Edge);
    let faces = shapes(&original, ShapeEnum::Face);
    let mut fillet = BRepFilletAPIMakeFillet::new(&original, FilletShape::Rational).unwrap();
    fillet
        .add_with_radius(2., &topods::edge(&edges[0]).unwrap())
        .unwrap();
    let result = fillet.shape().unwrap().clone();
    verify_validity_and_volume(&result);
    let new_faces = shapes(&result, ShapeEnum::Face);
    verify_face_oracle(&new_faces);
    verify_face_history(&mut fillet, &edges[0], &faces, &new_faces);
    report_edge_history(&mut fillet, &edges, &shapes(&result, ShapeEnum::Edge));
}

fn verify_validity_and_volume(result: &Shape) {
    assert!(
        BRepCheckAnalyzer::new(result, true, false, true)
            .unwrap()
            .is_valid()
            .unwrap()
    );
    let mut properties = GProps::new();
    BRepGProp::volume_properties(result, &mut properties, true, false, false).unwrap();
    let expected = 1000. - (4. - std::f64::consts::PI) * 10.;
    assert!((properties.mass() - expected).abs() < 1e-6);
}

fn verify_face_history(
    fillet: &mut BRepFilletAPIMakeFillet,
    edge: &Shape,
    faces: &[Shape],
    output: &[Shape],
) {
    let mut named: Vec<_> = fillet.generated(edge).unwrap().iter().cloned().collect();
    assert!(
        !named.is_empty(),
        "selected edge: expected generated face history"
    );
    for face in faces {
        named.extend(fillet.modified(face).unwrap().iter().cloned());
        if output.iter().any(|item| item.is_same(face)) {
            named.push(face.clone());
        }
    }
    for face in output {
        assert!(
            named.iter().any(|item| item.is_same(face)),
            "result face has no source history"
        );
    }
}

fn verify_face_oracle(faces: &[Shape]) {
    assert_eq!(
        faces.len(),
        7,
        "expected six trimmed planes and one blend face"
    );
    let (mut planes, curved) = classify_face_areas(faces);
    planes.sort_by(f64::total_cmp);
    let cap = 100. - (4. - std::f64::consts::PI);
    let expected = [80., 80., cap, cap, 100., 100.];
    assert_eq!(planes.len(), expected.len());
    for (actual, target) in planes.iter().zip(expected) {
        assert!(
            (actual - target).abs() < 1e-6,
            "plane face area {actual}: expected {target}"
        );
    }
    assert_eq!(curved.len(), 1);
    assert!((curved[0] - 10. * std::f64::consts::PI).abs() < 1e-6);
}

fn classify_face_areas(faces: &[Shape]) -> (Vec<f64>, Vec<f64>) {
    use martensite_tkmath::geom_abs::SurfaceType;
    let mut planes = Vec::new();
    let mut curved = Vec::new();
    for face in faces {
        assert_eq!(shapes(face, ShapeEnum::Wire).len(), 1, "face loop count");
        match surface_type(face) {
            SurfaceType::Plane => planes.push(face_area(face)),
            SurfaceType::Cylinder | SurfaceType::BSplineSurface => curved.push(face_area(face)),
            other => panic!(
                "fillet face surface {other:?}: expected plane or cylindrical representation"
            ),
        }
    }
    (planes, curved)
}

fn face_area(face: &Shape) -> f64 {
    let mut properties = GProps::new();
    BRepGProp::surface_properties(face, &mut properties, false, false).unwrap();
    properties.mass()
}

fn surface_type(face: &Shape) -> martensite_tkmath::geom_abs::SurfaceType {
    use martensite_tkbrep::brep_adaptor::BRepAdaptorSurface;
    use martensite_tkg3d::adaptor3d::Adaptor3dSurface;
    let face = topods::face(face).unwrap();
    BRepAdaptorSurface::with_face(&face, false)
        .unwrap()
        .get_type()
        .unwrap()
}

fn report_edge_history(fillet: &mut BRepFilletAPIMakeFillet, input: &[Shape], output: &[Shape]) {
    let mut attributed = Vec::new();
    for edge in input {
        attributed.extend(fillet.modified(edge).unwrap().iter().cloned());
        if output.iter().any(|item| item.is_same(edge)) {
            attributed.push(edge.clone());
        }
    }
    let covered = output
        .iter()
        .filter(|edge| attributed.iter().any(|source| source.is_same(edge)))
        .count();
    eprintln!(
        "fillet edge history: inputs={} outputs={} directly_attributed={covered}",
        input.len(),
        output.len()
    );
    assert!(
        covered >= 11,
        "expected at least 11 directly attributed edges, got {covered}"
    );
}
