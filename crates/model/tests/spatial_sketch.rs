// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::SketchEntityId,
    spatial_sketch::{SpatialSketch, SpatialSketchLine, SpatialSketchPoint},
};

fn point(id: u64, position: [f64; 3]) -> SpatialSketchPoint {
    SpatialSketchPoint {
        id: SketchEntityId(id),
        position,
    }
}
fn line(id: u64, first: u64, second: u64) -> SpatialSketchLine {
    SpatialSketchLine {
        id: SketchEntityId(id),
        endpoints: [SketchEntityId(first), SketchEntityId(second)],
    }
}

#[test]
fn keeps_source_order_names_and_shared_endpoint_identity() {
    let points = vec![
        point(7, [0., 1., 2.]),
        point(2, [3., 4., 5.]),
        point(9, [1., 0., 0.]),
    ];
    let lines = vec![line(11, 7, 2), line(12, 2, 9)];
    let sketch = SpatialSketch::new(
        DocumentSketchId(4),
        Some("3D Sketch1".into()),
        points.clone(),
        lines.clone(),
    )
    .unwrap();
    assert_eq!(sketch.id(), DocumentSketchId(4));
    assert_eq!(sketch.name(), Some("3D Sketch1"));
    assert_eq!(sketch.points(), points);
    assert_eq!(sketch.lines(), lines);
    assert_eq!(
        sketch.lines()[0].endpoints[1],
        sketch.lines()[1].endpoints[0]
    );
}

#[test]
fn rejects_missing_duplicate_and_nonfinite_geometry() {
    let invalid = [
        SpatialSketch::new(DocumentSketchId(0), None, vec![], vec![]),
        SpatialSketch::new(DocumentSketchId(1), None, vec![point(0, [0.; 3])], vec![]),
        SpatialSketch::new(
            DocumentSketchId(1),
            None,
            vec![point(1, [f64::NAN, 0., 0.])],
            vec![],
        ),
        SpatialSketch::new(
            DocumentSketchId(1),
            None,
            vec![point(1, [0.; 3]), point(1, [1.; 3])],
            vec![],
        ),
        SpatialSketch::new(
            DocumentSketchId(1),
            None,
            vec![point(1, [0.; 3])],
            vec![line(1, 1, 1)],
        ),
        SpatialSketch::new(
            DocumentSketchId(1),
            None,
            vec![point(1, [0.; 3])],
            vec![line(2, 1, 3)],
        ),
    ];
    for result in invalid {
        assert!(result.is_err());
    }
}
