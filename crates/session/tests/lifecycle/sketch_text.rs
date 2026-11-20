// SPDX-License-Identifier: GPL-2.0-only
use super::{sketch_fixture::owned_sketch_session, *};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchEntityId},
};

const SKETCH: DocumentSketchId = DocumentSketchId(1);

#[test]
fn text_create_and_edit_are_separate_undo_steps_with_a_stable_anchor() {
    let mut session = owned_sketch_session(None, "Label", Sketch2D::default());
    let id = session
        .create_sketch_text(SKETCH, [1., 2.], style("A"))
        .unwrap();
    let anchor = geometry(&session).text(id).unwrap().anchor;
    session.edit_sketch_text(SKETCH, id, style("B")).unwrap();
    assert_text(&session, id, anchor, "B");
    session.undo().unwrap();
    assert_text(&session, id, anchor, "A");
    session.undo().unwrap();
    assert!(geometry(&session).entity_ids().is_empty());
    assert!(geometry(&session).point(anchor).is_err());
    session.redo().unwrap();
    assert_text(&session, id, anchor, "A");
    session.redo().unwrap();
    assert_text(&session, id, anchor, "B");
}

fn style(content: &str) -> SketchTextStyle {
    SketchTextStyle {
        content: content.into(),
        height: 0.305,
        ..Default::default()
    }
}

fn geometry(session: &DocumentSession<RecordingObserver>) -> &Sketch2D {
    session.snapshot().sketches.get(SKETCH).unwrap().geometry()
}

fn assert_text(
    session: &DocumentSession<RecordingObserver>,
    id: SketchEntityId,
    anchor: SketchEntityId,
    content: &str,
) {
    let geometry = geometry(session);
    let text = geometry.text(id).unwrap();
    assert_eq!(text.style.content, content);
    assert_eq!(text.anchor, anchor);
    assert_eq!(geometry.point(anchor).unwrap().position, [1., 2.]);
}
