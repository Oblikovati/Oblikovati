// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_persistence::inventor_spatial::read_inventor_spatial_lines_and_points;

const FIXTURE: &[u8] = include_bytes!("fixtures/viewport-selection-solid.ipt");

#[test]
fn reads_spatial_points_lines_names_and_endpoint_identity_from_inventor_fixture() {
    let sketches = read_inventor_spatial_lines_and_points(FIXTURE).unwrap();
    assert_eq!(sketches.len(), 2);
    assert_eq!(sketches[0].name.as_deref(), Some("3D Sketch1"));
    assert_eq!(sketches[1].name.as_deref(), Some("3D Sketch6"));
    assert_eq!(
        sketches
            .iter()
            .map(|sketch| sketch.points.len())
            .collect::<Vec<_>>(),
        [2, 3]
    );
    assert_eq!(
        sketches
            .iter()
            .map(|sketch| sketch.lines.len())
            .collect::<Vec<_>>(),
        [1, 1]
    );
    assert_eq!(sketches[0].points[0].position_cm, [0., 0., 2.]);
    assert_eq!(sketches[0].points[1].position_cm, [1.5, 0., 2.]);
    assert_eq!(sketches[1].points[0].position_cm, [-1., -0.5, 2.5]);
    for sketch in &sketches {
        assert_eq!(
            sketch.lines[0].endpoints,
            [
                sketch.points[sketch.points.len() - 2].node,
                sketch.points[sketch.points.len() - 1].node
            ]
        );
    }
    assert_eq!(
        sketches,
        read_inventor_spatial_lines_and_points(FIXTURE).unwrap()
    );
}

#[test]
fn rejects_non_inventor_bytes_without_returning_partial_geometry() {
    let error =
        read_inventor_spatial_lines_and_points(b"not an Inventor compound file").unwrap_err();
    assert!(error.contains("Inventor document"), "{error}");
}
