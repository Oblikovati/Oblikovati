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

#[test]
fn projects_fixture_sketches_with_caller_owned_ids_and_preserves_references() {
    use oblikovati_model::document_sketch::DocumentSketchId;
    use oblikovati_persistence::inventor_spatial::project_inventor_point_and_line_sketch;
    let source = read_inventor_spatial_lines_and_points(FIXTURE).unwrap();
    let units = oblikovati_model::units::UnitsOfMeasure::default();
    let first =
        project_inventor_point_and_line_sketch(DocumentSketchId(101), &source[0], &units).unwrap();
    let second =
        project_inventor_point_and_line_sketch(DocumentSketchId(102), &source[1], &units).unwrap();
    let scaled = units.clone().with_working_scale(0.1).unwrap();
    let millimetre_working =
        project_inventor_point_and_line_sketch(DocumentSketchId(103), &source[0], &scaled).unwrap();
    assert_eq!(millimetre_working.points()[0].position, [0., 0., 20.]);
    assert_eq!(first.id(), DocumentSketchId(101));
    assert_eq!(second.name(), Some("3D Sketch6"));
    assert_eq!(first.points()[0].position, [0., 0., 2.]);
    assert_eq!(
        first.lines()[0].endpoints,
        [first.points()[0].id, first.points()[1].id]
    );
    assert_eq!(
        second.lines()[0].endpoints,
        [second.points()[1].id, second.points()[2].id]
    );
    assert!(
        project_inventor_point_and_line_sketch(DocumentSketchId(0), &source[0], &units).is_err()
    );
}

#[test]
fn refuses_spatial_coordinate_overflow_in_working_units() {
    use oblikovati_model::{document_sketch::DocumentSketchId, units::UnitsOfMeasure};
    use oblikovati_persistence::inventor_spatial::{
        InventorSpatialPoint, InventorSpatialSketch, project_inventor_point_and_line_sketch,
    };
    let source = InventorSpatialSketch {
        node: 1,
        name: None,
        points: vec![InventorSpatialPoint {
            node: 2,
            position_cm: [f64::MAX, 0., 0.],
        }],
        lines: vec![],
    };
    let units = UnitsOfMeasure::default().with_working_scale(0.1).unwrap();
    let error =
        project_inventor_point_and_line_sketch(DocumentSketchId(1), &source, &units).unwrap_err();
    assert!(error.contains("finite coordinates"), "{error}");
}
