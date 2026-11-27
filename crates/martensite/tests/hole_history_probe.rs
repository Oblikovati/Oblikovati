// SPDX-License-Identifier: GPL-2.0-only
//! Check the pinned kernel's cylindrical cut and history for Hole (#4093).
use martensite_tkbo::brep_algo_api::BRepAlgoAPICut;
use martensite_tkbrep::{
    brep_adaptor::BRepAdaptorSurface,
    topexp,
    topods::{self, Shape},
    toptools::IndexedMapOfShape,
};
use martensite_tkg3d::{adaptor3d::Adaptor3dSurface, top_abs::ShapeEnum};
use martensite_tkgeombase::gprop::GProps;
use martensite_tkmath::{
    geom_abs::SurfaceType,
    gp::{Axis2, Direction3, Point3},
};
use martensite_tkprim::brep_prim_api::{BRepPrimAPIMakeBox, BRepPrimAPIMakeCylinder};
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

fn cylindrical_cut(start: f64, depth: f64) -> (Shape, Shape, BRepAlgoAPICut) {
    let block = BRepPrimAPIMakeBox::from_sizes(10., 10., 10.)
        .unwrap()
        .shape()
        .unwrap()
        .clone();
    let axis = Axis2::new(
        Point3::new(5., 5., start),
        Direction3::new(0., 0., 1.).unwrap(),
    )
    .unwrap();
    let drill = BRepPrimAPIMakeCylinder::with_axes(axis, 1., depth)
        .shape()
        .unwrap()
        .clone();
    let cut = BRepAlgoAPICut::from_shapes(&block, &drill).unwrap();
    (block, drill, cut)
}

#[test]
fn through_hole_has_valid_analytic_volume_and_faces() {
    let (_, _, cut) = cylindrical_cut(-1., 12.);
    assert!(cut.is_done() && !cut.has_errors());
    let result = cut.shape();
    assert!(
        BRepCheckAnalyzer::new(result, true, false, true)
            .unwrap()
            .is_valid()
            .unwrap()
    );
    let mut volume = GProps::new();
    BRepGProp::volume_properties(result, &mut volume, true, false, false).unwrap();
    let expected = 1000. - 10. * std::f64::consts::PI;
    assert!(
        (volume.mass() - expected).abs() < 1e-6,
        "volume {}: expected {expected}",
        volume.mass()
    );
    let faces = shapes(result, ShapeEnum::Face);
    assert_eq!(faces.len(), 7, "expected six block faces and one bore face");
    assert_surface_types(&faces, 6, 1);
    assert_face_areas(
        &faces,
        &[
            20. * std::f64::consts::PI,
            100. - std::f64::consts::PI,
            100. - std::f64::consts::PI,
            100.,
            100.,
            100.,
            100.,
        ],
    );
}

fn assert_surface_types(faces: &[Shape], planes: usize, cylinders: usize) {
    let kinds = faces
        .iter()
        .map(|shape| {
            let face = topods::face(shape).unwrap();
            BRepAdaptorSurface::with_face(&face, false)
                .unwrap()
                .get_type()
                .unwrap()
        })
        .collect::<Vec<_>>();
    assert_eq!(
        kinds
            .iter()
            .filter(|kind| **kind == SurfaceType::Plane)
            .count(),
        planes
    );
    assert_eq!(
        kinds
            .iter()
            .filter(|kind| **kind == SurfaceType::Cylinder)
            .count(),
        cylinders
    );
}

fn assert_face_areas(faces: &[Shape], expected: &[f64]) {
    let mut areas = faces
        .iter()
        .map(|face| {
            let mut props = GProps::new();
            BRepGProp::surface_properties(face, &mut props, false, false).unwrap();
            props.mass()
        })
        .collect::<Vec<_>>();
    areas.sort_by(f64::total_cmp);
    assert_eq!(areas.len(), expected.len());
    for (actual, expected) in areas.iter().zip(expected) {
        assert!(
            (actual - expected).abs() < 1e-6,
            "face area {actual}: expected {expected}"
        );
    }
}

#[test]
fn through_hole_face_history_identifies_every_output_face() {
    let (block, drill, cut) = cylindrical_cut(-1., 12.);
    let output = shapes(cut.shape(), ShapeEnum::Face);
    let input = shapes(&block, ShapeEnum::Face)
        .into_iter()
        .chain(shapes(&drill, ShapeEnum::Face));
    let mut histories = Vec::new();
    for face in input {
        let mut attributed = cut.modified(&face).unwrap();
        attributed.extend(cut.generated(&face).unwrap());
        if output.iter().any(|item| item.is_same(&face)) {
            attributed.push(face);
        }
        histories.push(attributed);
    }
    for face in output {
        let parents = histories
            .iter()
            .filter(|history| history.iter().any(|item| item.is_same(&face)))
            .count();
        assert_eq!(parents, 1, "output face: expected one history parent");
    }
}

#[test]
fn blind_hole_at_the_placement_face_has_valid_analytic_geometry() {
    let (_, _, cut) = cylindrical_cut(5., 5.);
    assert!(cut.is_done() && !cut.has_errors());
    let result = cut.shape();
    assert!(
        BRepCheckAnalyzer::new(result, true, false, true)
            .unwrap()
            .is_valid()
            .unwrap()
    );
    let mut volume = GProps::new();
    BRepGProp::volume_properties(result, &mut volume, true, false, false).unwrap();
    let expected = 1000. - 5. * std::f64::consts::PI;
    assert!(
        (volume.mass() - expected).abs() < 1e-6,
        "volume {}: expected {expected}",
        volume.mass()
    );
    assert_eq!(
        shapes(result, ShapeEnum::Face).len(),
        8,
        "expected six block faces, bore and flat bottom"
    );
    let pi = std::f64::consts::PI;
    let faces = shapes(result, ShapeEnum::Face);
    assert_surface_types(&faces, 7, 1);
    assert_face_areas(
        &faces,
        &[pi, 10. * pi, 100. - pi, 100., 100., 100., 100., 100.],
    );
}

#[test]
fn through_hole_with_both_caps_on_the_body_is_valid() {
    let (_, _, cut) = cylindrical_cut(0., 10.);
    assert!(cut.is_done() && !cut.has_errors());
    let result = cut.shape();
    assert!(
        BRepCheckAnalyzer::new(result, true, false, true)
            .unwrap()
            .is_valid()
            .unwrap()
    );
    let mut volume = GProps::new();
    BRepGProp::volume_properties(result, &mut volume, true, false, false).unwrap();
    let expected = 1000. - 10. * std::f64::consts::PI;
    assert!(
        (volume.mass() - expected).abs() < 1e-6,
        "volume {}: expected {expected}",
        volume.mass()
    );
    assert_eq!(shapes(result, ShapeEnum::Face).len(), 7);
}
