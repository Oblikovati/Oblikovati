// SPDX-License-Identifier: GPL-2.0-only
//! Check Boolean history against the retained sketch-prism path (#4093).
use super::*;
use martensite_tkbo::brep_algo_api::BRepAlgoAPICut;
use martensite_tkbrep::{topexp, topods::Shape, toptools::IndexedMapOfShape};
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkmath::gp::{Axis2, Direction3, Point3};
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeCylinder;
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;

fn prism_faces(body: &Shape) -> Vec<Shape> {
    let mut map = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(body, ShapeEnum::Face, &mut map).unwrap();
    (1..=map.extent())
        .map(|index| map.find_key(index).unwrap().clone())
        .collect()
}

#[test]
fn copied_sketch_prism_cut_preserves_original_face_lineage() {
    let source = retain_planar_prism(&PrismProbeDefinition {
        outer: PrismProbeLoop::polygon(
            vec![1, 2, 3, 4],
            vec![[0., 0.], [10., 0.], [10., 10.], [0., 10.]],
        ),
        holes: Vec::new(),
        displacement: [0., 0., 10.],
    })
    .unwrap();
    let mut copier = crate::checked_shape::copy_shape(&source.shape, "hole probe").unwrap();
    let target = copier.shape().unwrap().clone();
    let axis = Axis2::new(
        Point3::new(5., 5., -1.),
        Direction3::new(0., 0., 1.).unwrap(),
    )
    .unwrap();
    let tool = BRepPrimAPIMakeCylinder::with_axes(axis, 1., 12.)
        .shape()
        .unwrap()
        .clone();
    let cut = BRepAlgoAPICut::from_shapes(&target, &tool).unwrap();
    crate::checked_shape::require_geometry(cut.shape(), "hole prism probe").unwrap();
    let result = prism_faces(cut.shape());
    assert_eq!(result.len(), 7);
    for (_, original_face) in &source.faces {
        let copies = copier.modified(original_face).unwrap();
        assert_eq!(copies.len(), 1, "input face: expected one detached copy");
        let copied = copies.first().unwrap();
        let mut descendants = cut.modified(copied).unwrap();
        if result.iter().any(|face| face.is_same(copied)) {
            descendants.push(copied.clone());
        }
        assert_eq!(
            descendants.len(),
            1,
            "input face: expected one surviving descendant"
        );
        assert!(result.iter().any(|face| face.is_same(&descendants[0])));
    }
    let original =
        crate::checked_shape::analytic_properties(&source.shape, "source prism").unwrap();
    let drilled = crate::checked_shape::analytic_properties(cut.shape(), "drilled prism").unwrap();
    assert!((original.volume - 1000.).abs() < 1e-6);
    assert!((drilled.volume - (1000. - 10. * std::f64::consts::PI)).abs() < 1e-6);
}
