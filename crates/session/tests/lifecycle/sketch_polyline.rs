// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::SketchPolylineCreation,
    sketch::{Sketch2D, SketchPlane},
};
fn polyline_input() -> SketchPolylineCreation {
    SketchPolylineCreation {
        points: vec![[0.0; 2], [3.0, 0.0], [3.0, 2.0]],
        closed: true,
        construction: false,
    }
}
fn polyline_session() -> (DocumentSession<RecordingObserver>, DocumentSketchId) {
    let mut session = session(None);
    let id = session
        .run("Sketch", |snapshot, ids| {
            let id = ids.allocate_sketch().unwrap();
            snapshot
                .sketches
                .add(id, "Polyline", SketchPlane::xy(), Sketch2D::default())?;
            Ok::<_, oblikovati_model::sketch::SketchError>(id)
        })
        .unwrap();
    (session, id)
}
#[test]
fn rejected_polyline_reply_drops_every_point_and_line_but_keeps_reservations() {
    let (mut session, sketch) = polyline_session();
    let before = session.snapshot().sketches.clone();
    let result = session.create_sketch_polyline_projected(sketch, &polyline_input(), |created| {
        assert_eq!(created.unwrap().lines.len(), 3);
        Err::<(), _>("reply rejected")
    });
    assert!(result.is_err());
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(session.history().sketch_reservations().entity, 6);
    let created = session
        .create_sketch_polyline(sketch, &polyline_input())
        .unwrap();
    assert_eq!(created.lines[0].0, 10);
    session.undo().unwrap();
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(session.history().sketch_reservations().entity, 12);
}
