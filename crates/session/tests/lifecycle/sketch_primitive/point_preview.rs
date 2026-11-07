// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::SketchPointPlacement;
#[test]
fn point_preview_matches_commit_without_consuming_ids_or_live_history() {
    for center in [false, true] {
        let mut session = session(None);
        let sketch = empty_sketch(&mut session, false);
        let before = session.snapshot().sketches.clone();
        let marks = session.history().sketch_reservations();
        let generation = session.generation();
        let events = session.observer().observations.len();
        let placement = SketchPointPlacement {
            position: [3., -4.],
            center,
            coincident_with: None,
        };
        let preview = session.preview_sketch_point(sketch, &placement).unwrap();
        assert_eq!(session.snapshot().sketches, before);
        assert_eq!(session.history().sketch_reservations(), marks);
        assert_eq!(session.generation(), generation);
        assert_eq!(session.observer().observations.len(), events);
        let point = session.create_sketch_point(sketch, &placement).unwrap();
        assert_eq!(preview.members, [point]);
        assert_eq!(&preview.geometry, geometry(&session, sketch));
        assert_eq!(preview.geometry.is_center_point(point), center);
    }
}
#[test]
fn invalid_point_preview_names_the_invalid_coordinate_and_preserves_geometry() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    let before = session.snapshot().sketches.clone();
    let placement = SketchPointPlacement {
        position: [f64::NAN, 2.],
        center: true,
        coincident_with: None,
    };
    let error = session
        .preview_sketch_point(sketch, &placement)
        .unwrap_err()
        .to_string();
    assert!(error.contains("NaN"), "{error}");
    assert_eq!(session.snapshot().sketches, before);
}
