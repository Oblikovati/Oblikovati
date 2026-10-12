// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use rustycad_tkbrep::{brep::BRepTool, topexp::Explorer, topods::Face};
use rustycad_tkg3d::top_abs::ShapeEnum;
use rustycad_tkmath::{
    gp::{Transform3, Vector3},
    toploc::Location,
};
fn faces(shape: &Shape) -> Vec<Face> {
    let mut explorer = Explorer::new(shape, ShapeEnum::Face, ShapeEnum::Shape).unwrap();
    let mut faces = Vec::new();
    while explorer.more() {
        faces.push(rustycad_tkbrep::topods::face(explorer.current()).unwrap());
        explorer.next().unwrap();
    }
    faces
}
#[test]
fn snapshot_detaches_each_face_surface_and_isolates_source_geometry_changes() {
    let source = build_probe([2., 3., 4.]).unwrap();
    let frozen = ReleaseProbeSnapshot::capture(&source, [2., 3., 4.]).unwrap();
    let baseline = frozen.properties().unwrap();
    let original_faces = faces(&source);
    let copied_faces = faces(&frozen.shape);
    assert_eq!((original_faces.len(), copied_faces.len()), (6, 6));
    for (original, copied) in original_faces.iter().zip(&copied_faces) {
        assert!(!original.is_partner(copied));
        let original_surface = BRepTool::surface(original).unwrap();
        let copied_surface = BRepTool::surface(copied).unwrap();
        assert_ne!(original_surface, copied_surface);
        let before = copied_surface.value(0., 0.).unwrap();
        original_surface
            .translate(Vector3::new(100., 200., 300.))
            .unwrap();
        assert_eq!(copied_surface.value(0., 0.).unwrap(), before);
    }
    assert_eq!(frozen.properties().unwrap(), baseline);
}
#[test]
fn detached_snapshots_preserve_location_orientation_and_analytic_values() {
    let source = build_probe([2., 3., 4.]).unwrap();
    let located = source
        .moved(
            &Location::new(Transform3::translation([10., 20., 30.])),
            false,
        )
        .unwrap();
    let frozen = ReleaseProbeSnapshot::capture(&located, [2., 3., 4.]).unwrap();
    let detached = frozen.detached().unwrap();
    assert!(!frozen.shape.is_partner(&detached.shape));
    assert_eq!(frozen.shape.location(), detached.shape.location());
    assert_eq!(frozen.shape.orientation(), detached.shape.orientation());
    let properties = detached.properties().unwrap();
    assert!((properties.volume - 24.).abs() < 24. * 1e-12);
    for (actual, expected) in properties.centroid.into_iter().zip([11., 21.5, 32.]) {
        assert!((actual - expected).abs() < expected * 1e-12);
    }
    assert_eq!(frozen.properties().unwrap(), detached.properties().unwrap());
}
#[test]
fn null_sources_and_nonfinite_analytic_results_are_named_declines() {
    assert!(
        ReleaseProbeSnapshot::capture(&Shape::default(), [1.; 3])
            .err()
            .unwrap()
            .contains("non-null")
    );
    verify_nonfinite_properties_declined();
}
fn verify_nonfinite_properties_declined() {
    for properties in [
        ReleaseProbeProperties {
            volume: f64::INFINITY,
            centroid: [0.; 3],
        },
        ReleaseProbeProperties {
            volume: 1.,
            centroid: [f64::NAN, 0., 0.],
        },
    ] {
        assert!(
            crate::require_finite_properties(&properties, [1.; 3])
                .unwrap_err()
                .contains("expected finite analytic")
        );
    }
}

fn unique_shapes(shape: &Shape, kind: ShapeEnum) -> Vec<Shape> {
    let mut indexed = rustycad_tkbrep::toptools::IndexedMapOfShape::default();
    rustycad_tkbrep::topexp::map_shapes_of_type(shape, kind, &mut indexed).unwrap();
    (1..=indexed.extent())
        .map(|index| indexed.find_key(index).unwrap().clone())
        .collect()
}
#[test]
fn edge_geometry_and_face_attributes_remain_isolated_after_capture() {
    let source = build_probe([2., 3., 4.]).unwrap();
    let frozen = ReleaseProbeSnapshot::capture(&source, [2., 3., 4.]).unwrap();
    let originals = unique_shapes(&source, ShapeEnum::Edge);
    let copies = unique_shapes(&frozen.shape, ShapeEnum::Edge);
    assert_eq!((originals.len(), copies.len()), (12, 12));
    for (original, copied) in originals.iter().zip(&copies) {
        let original = rustycad_tkbrep::topods::edge(original).unwrap();
        let copied = rustycad_tkbrep::topods::edge(copied).unwrap();
        let original = BRepTool::curve_with_location(&original).unwrap().curve;
        let copied = BRepTool::curve_with_location(&copied).unwrap().curve;
        assert_ne!(original, copied);
        let before = copied.value(0.).unwrap();
        original.translate(Vector3::new(5., 6., 7.)).unwrap();
        assert_eq!(copied.value(0.).unwrap(), before);
    }
    verify_face_tolerance_isolation(&source, &frozen.shape);
}
fn verify_face_tolerance_isolation(source: &Shape, frozen: &Shape) {
    let original = faces(source)[0].clone();
    let copied = faces(frozen)[0].clone();
    let before = BRepTool::face_tolerance(&copied).unwrap();
    rustycad_tkbrep::brep::BRepBuilder::default()
        .update_face_tolerance(&original, 0.25)
        .unwrap();
    assert_eq!(BRepTool::face_tolerance(&copied).unwrap(), before);
    assert_eq!(BRepTool::face_tolerance(&original).unwrap(), 0.25);
}
#[test]
fn detached_copies_match_occt_properties_and_unique_topology_for_scaled_reversed_inputs() {
    let corpus = include_str!("snapshot/occt-8.0.0.csv");
    assert_eq!(corpus.lines().count(), 6);
    for line in corpus.lines() {
        let expected: Vec<f64> = line
            .split(',')
            .map(|column| column.parse().unwrap())
            .collect();
        verify_occt_row(&expected);
    }
}
fn verify_occt_row(expected: &[f64]) {
    let scale = expected[0];
    let extents = [2., 3., 4.].map(|extent| extent * scale);
    let source = build_probe(extents).unwrap();
    let source = located_orientation(source, scale, expected[1] == 1.);
    let frozen = ReleaseProbeSnapshot::capture(&source, extents).unwrap();
    verify_occt_properties(frozen.properties().unwrap(), &expected[2..6]);
    for (kind, count) in [ShapeEnum::Face, ShapeEnum::Edge, ShapeEnum::Vertex]
        .into_iter()
        .zip(&expected[6..])
    {
        assert_eq!(unique_shapes(&frozen.shape, kind).len(), *count as usize);
    }
}

fn verify_occt_properties(properties: ReleaseProbeProperties, expected: &[f64]) {
    let actual = [
        properties.volume,
        properties.centroid[0],
        properties.centroid[1],
        properties.centroid[2],
    ];
    for (actual, reference) in actual.into_iter().zip(expected) {
        assert!(
            (actual - reference).abs() <= reference.abs() * 1e-12,
            "got {actual}, expected {reference}"
        );
    }
}
#[test]
fn kernel_parameter_limits_cannot_be_retained_as_a_successful_snapshot() {
    let extents = [1e103; 3];
    let error = retain_box_release(extents)
        .err()
        .expect("expected a named refusal for the kernel parameter domain");
    assert!(
        error.contains("expected a constructed solid") && error.contains(&format!("{extents:?}")),
        "{error}"
    );
}

fn located_orientation(source: Shape, scale: f64, reversed: bool) -> Shape {
    let pose = Location::new(Transform3::translation(
        [10., 20., 30.].map(|coordinate| coordinate * scale),
    ));
    let source = source.moved(&pose, false).unwrap();
    if reversed {
        return source.reversed();
    }
    source
}
#[test]
fn vertex_point_mutation_does_not_cross_the_snapshot_boundary() {
    let source = build_probe([2., 3., 4.]).unwrap();
    let frozen = ReleaseProbeSnapshot::capture(&source, [2., 3., 4.]).unwrap();
    let originals = unique_shapes(&source, ShapeEnum::Vertex);
    let copies = unique_shapes(&frozen.shape, ShapeEnum::Vertex);
    assert_eq!((originals.len(), copies.len()), (8, 8));
    let original = rustycad_tkbrep::topods::vertex(&originals[0]).unwrap();
    let copied = rustycad_tkbrep::topods::vertex(&copies[0]).unwrap();
    assert!(!original.is_partner(&copied));
    let before = BRepTool::pnt(&copied).unwrap();
    let tolerance = BRepTool::vertex_tolerance(&original).unwrap();
    let changed = rustycad_tkmath::gp::Point3::new(100., 200., 300.);
    rustycad_tkbrep::brep::BRepBuilder::default()
        .update_vertex_point(&original, changed, tolerance)
        .unwrap();
    assert_eq!(BRepTool::pnt(&copied).unwrap(), before);
    assert_ne!(BRepTool::pnt(&original).unwrap(), before);
}
