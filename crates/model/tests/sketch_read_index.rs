// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::{OwnedSketchGeometry, Sketch2D, SketchEntityId as Id};
fn fixture() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, position) in [(100, [0., -0.]), (200, [3., 4.]), (300, [-4., 3.])] {
        sketch.add_defining_point(Id(id), position).unwrap();
    }
    sketch.add_line(Id(90), [Id(100), Id(200)]).unwrap();
    sketch
        .add_arc(Id(10), Id(100), [Id(200), Id(300)], true)
        .unwrap();
    sketch.add_point(Id(50), [8., 9.]).unwrap();
    sketch
}
#[test]
fn batch_reads_preserve_authored_order_and_hidden_point_bits() {
    let sketch = fixture();
    let before = sketch.recipe();
    let index = sketch.read_index();
    assert_eq!(index.entity_ids(), &[Id(90), Id(10), Id(50)]);
    assert!(matches!(
        index.entity(Id(90)).unwrap(),
        OwnedSketchGeometry::Line(_)
    ));
    assert_eq!(
        index.point(Id(100)).unwrap().position[1].to_bits(),
        (-0f64).to_bits()
    );
    assert_eq!(index.arc_radius(Id(10)).unwrap().to_bits(), 5f64.to_bits());
    assert_eq!(sketch.arc_radius(Id(10)).unwrap().to_bits(), 5f64.to_bits());
    assert_eq!(
        serde_json::to_vec(&before).unwrap(),
        serde_json::to_vec(&sketch.recipe()).unwrap()
    );
}
#[test]
fn indexed_roles_match_independent_axis_and_construction_metadata() {
    for construction in [false, true] {
        for centerline in [false, true] {
            let mut sketch = fixture();
            let line = Id(90);
            sketch
                .set_curve_construction(&[line], construction)
                .unwrap();
            sketch.set_line_centerline(line, centerline).unwrap();
            let index = sketch.read_index();
            assert_eq!(
                index.is_construction(line).unwrap(),
                construction || centerline
            );
            assert_eq!(index.is_centerline(line).unwrap(), centerline);
            assert!(!index.is_construction(Id(50)).unwrap());
            assert!(!index.is_centerline(Id(10)).unwrap());
        }
    }
}
#[test]
fn failed_reads_name_missing_or_wrong_geometry() {
    let sketch = fixture();
    let index = sketch.read_index();
    let missing = Id(999);
    for cause in [
        index.entity(missing).unwrap_err(),
        index.point(missing).unwrap_err(),
        index.arc_radius(missing).unwrap_err(),
        index.is_construction(missing).unwrap_err(),
        index.is_centerline(missing).unwrap_err(),
    ] {
        assert!(cause.to_string().contains("999"));
    }
    let point_failure = index.point(Id(90)).unwrap_err().to_string();
    assert!(point_failure.contains("point geometry"));
    let arc_failure = index.arc_radius(Id(90)).unwrap_err().to_string();
    assert!(arc_failure.contains("circular arc"));
}
#[test]
fn degenerate_radius_is_refused_by_both_read_paths() {
    let mut sketch = fixture();
    sketch
        .add_arc(Id(20), Id(100), [Id(100), Id(300)], false)
        .unwrap();
    let expected = sketch.arc_radius(Id(20)).unwrap_err().to_string();
    assert_eq!(
        sketch
            .read_index()
            .arc_radius(Id(20))
            .unwrap_err()
            .to_string(),
        expected
    );
    assert!(expected.contains("positive finite"));
}

#[test]
fn circular_center_markers_retain_hidden_identity_without_duplicate_glyphs() {
    let mut sketch = fixture();
    sketch.add_circle(Id(40), Id(100), 2.).unwrap();
    sketch.add_circle(Id(41), Id(50), 2.).unwrap();
    let centers = sketch.read_index().implicit_circular_centers().unwrap();
    assert_eq!(centers.len(), 1);
    assert_eq!(centers[0].id, Id(100));
    assert_eq!(centers[0].position[1].to_bits(), (-0f64).to_bits());
}
