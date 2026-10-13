// SPDX-License-Identifier: GPL-2.0-only
//! End-to-end diagnostic region extrusion; this does not publish a feature.
use super::*;
use rustycad_tkbrep::brep_adaptor::BRepAdaptorSurface;
use rustycad_tkg3d::adaptor3d::Adaptor3dSurface;
use rustycad_tkmath::gp::Vector3;
use rustycad_tkprim::brep_prim_api::BRepPrimAPIMakePrism;
use rustycad_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;
use std::collections::BTreeSet;

struct PrismOracleRow {
    region: [f64; 3],
    volume: f64,
    face: [f64; 7],
}

pub(super) fn verify_curved_prisms(case: &str, regions: &BuilderFace) {
    let oracle = prism_oracle_rows(case);
    let mut used = BTreeSet::new();
    for region in bounded_region_faces(regions).unwrap() {
        verify_region_prism(&region, &oracle, &mut used);
    }
    assert_eq!(
        used.len(),
        oracle.len(),
        "case {case}: expected every oracle face"
    );
}

fn verify_region_prism(region: &Shape, oracle: &[PrismOracleRow], used: &mut BTreeSet<usize>) {
    let region_key = region_oracle_key(region);
    let mut maker =
        BRepPrimAPIMakePrism::new(region, Vector3::new(0., 0., 2.), true, true).unwrap();
    let solid = checked_prism_solid(&mut maker);
    let volume = prism_volume(&solid);
    let faces = indexed_region_shapes(&solid, ShapeEnum::Face).unwrap();
    verify_prism_generated_history(region, &faces, &mut maker);
    for index in 1..=faces.extent() {
        match_prism_face(
            faces.find_key(index).unwrap(),
            region_key,
            volume,
            oracle,
            used,
        );
    }
}

fn region_oracle_key(region: &Shape) -> [f64; 3] {
    let properties = region_surface_properties(region).unwrap();
    let center = properties.centre_of_mass();
    [properties.mass(), center.x(), center.y()]
}

fn checked_prism_solid(maker: &mut BRepPrimAPIMakePrism) -> Shape {
    let solid = maker.shape().unwrap().clone();
    assert_eq!(solid.shape_type().unwrap(), ShapeEnum::Solid);
    let validity = BRepCheckAnalyzer::new(&solid, true, false, true).unwrap();
    assert!(validity.is_valid().unwrap());
    solid
}

fn prism_volume(solid: &Shape) -> f64 {
    let mut volume = GProps::new();
    BRepGProp::volume_properties(solid, &mut volume, true, false, false).unwrap();
    volume.mass()
}

fn match_prism_face(
    shape: &Shape,
    region: [f64; 3],
    volume: f64,
    oracle: &[PrismOracleRow],
    used: &mut BTreeSet<usize>,
) {
    let actual = prism_face_metrics(shape);
    let index = unique_prism_face(region, actual, oracle);
    assert!(
        used.insert(index),
        "region {region:?}: duplicate oracle face {index}"
    );
    assert_eq!(actual[4..], oracle[index].face[4..]);
    assert!((volume - oracle[index].volume).abs() < 1e-8);
}

fn unique_prism_face(region: [f64; 3], actual: [f64; 7], oracle: &[PrismOracleRow]) -> usize {
    let candidates: Vec<usize> = oracle
        .iter()
        .enumerate()
        .filter_map(|(index, row)| {
            (close_values(&region, &row.region) && close_values(&actual[..4], &row.face[..4]))
                .then_some(index)
        })
        .collect();
    assert_eq!(
        candidates.len(),
        1,
        "region {region:?}, face {actual:?}: expected one oracle face"
    );
    candidates[0]
}

fn close_values(actual: &[f64], expected: &[f64]) -> bool {
    actual.len() == expected.len()
        && actual
            .iter()
            .zip(expected)
            .all(|(left, right)| (left - right).abs() < 1e-8)
}

fn prism_face_metrics(shape: &Shape) -> [f64; 7] {
    let properties = region_surface_properties(shape).unwrap();
    let center = properties.centre_of_mass();
    let adaptor = BRepAdaptorSurface::with_face(&face(shape).unwrap(), true).unwrap();
    [
        properties.mass(),
        center.x(),
        center.y(),
        center.z(),
        indexed_region_shapes(shape, ShapeEnum::Wire)
            .unwrap()
            .extent() as f64,
        indexed_region_shapes(shape, ShapeEnum::Edge)
            .unwrap()
            .extent() as f64,
        adaptor.get_type().unwrap() as i32 as f64,
    ]
}

fn prism_oracle_rows(case: &str) -> Vec<PrismOracleRow> {
    include_str!("occt-8.0.0-curved-prisms.csv")
        .lines()
        .filter_map(|line| line.strip_prefix(&format!("{case},")))
        .map(|line| {
            let values: Vec<f64> = line
                .split(',')
                .map(|value| value.parse().unwrap())
                .collect();
            assert_eq!(values.len(), 11);
            PrismOracleRow {
                region: values[..3].try_into().unwrap(),
                volume: values[3],
                face: values[4..].try_into().unwrap(),
            }
        })
        .collect()
}

fn verify_prism_generated_history(
    region: &Shape,
    faces: &IndexedMapOfShape,
    maker: &mut BRepPrimAPIMakePrism,
) {
    use rustycad_tkprim::brep_prim_api::BRepPrimAPIMakeSweep;
    let edges = indexed_region_shapes(region, ShapeEnum::Edge).unwrap();
    let mut roles = vec![maker.first_shape().unwrap(), maker.last_shape().unwrap()];
    for index in 1..=edges.extent() {
        let generated = maker.generated(edges.find_key(index).unwrap()).unwrap();
        assert_eq!(
            generated.len(),
            1,
            "profile edge {index}: expected one side face"
        );
        roles.push(generated.first().unwrap().clone());
    }
    assert_eq!(roles.len(), faces.extent());
    for index in 1..=faces.extent() {
        let shape = faces.find_key(index).unwrap();
        assert_eq!(roles.iter().filter(|role| role.is_same(shape)).count(), 1);
    }
}
