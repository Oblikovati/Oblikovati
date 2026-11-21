// SPDX-License-Identifier: GPL-2.0-only
use super::{sketch_fixture::owned_sketch_session, *};
use oblikovati_api::types::{SketchTextRun, TEXT_ALIGN_BASELINE, TEXT_ALIGN_LEFT};
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

#[test]
fn text_format_changes_restore_the_same_entity_through_history() {
    let mut session = owned_sketch_session(None, "Label", Sketch2D::default());
    let plain = style("A界🙂");
    let id = session
        .create_sketch_text(SKETCH, [1., 2.], plain.clone())
        .unwrap();
    let mut bold = plain.clone();
    bold.formatting_runs = Some(vec![bold_middle_character()]);
    session.edit_sketch_text(SKETCH, id, bold.clone()).unwrap();
    assert_eq!(geometry(&session).text(id).unwrap().style, bold);
    session.undo().unwrap();
    assert_eq!(geometry(&session).text(id).unwrap().style, plain);
    session.redo().unwrap();
    assert_eq!(geometry(&session).text(id).unwrap().style, bold);
}

#[test]
fn shortening_text_cannot_leave_formatting_outside_the_new_content() {
    let mut session = owned_sketch_session(None, "Label", Sketch2D::default());
    let mut formatted = style("A界🙂");
    formatted.formatting_runs = Some(vec![bold_middle_character()]);
    let id = session
        .create_sketch_text(SKETCH, [1., 2.], formatted.clone())
        .unwrap();
    let mut short = formatted.clone();
    short.content = "A".into();
    let error = session.edit_sketch_text(SKETCH, id, short).unwrap_err();
    assert!(error.to_string().contains("Unicode scalars"));
    assert_eq!(geometry(&session).text(id).unwrap().style, formatted);
    session.undo().unwrap();
    assert!(geometry(&session).texts().is_empty());
}

fn bold_middle_character() -> SketchTextRun {
    SketchTextRun {
        start: 1,
        end: 2,
        bold: true,
        ..Default::default()
    }
}

fn style(content: &str) -> SketchTextStyle {
    SketchTextStyle {
        content: content.into(),
        height: 0.305,
        h_align: TEXT_ALIGN_LEFT,
        v_align: TEXT_ALIGN_BASELINE,
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
