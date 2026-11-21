// SPDX-License-Identifier: GPL-2.0-only
//! Verify Unicode offsets and replacement for authored fields in Format Text.
use super::*;

#[test]
fn insertion_uses_scalar_positions_and_keeps_field_order() {
    let mut draft = GuiSketchTextDraft {
        content: "界x".into(),
        ..Default::default()
    };
    assert_eq!(insert_field(&mut draft, 3..4, parameter("d0")), Some(6..6));
    assert_eq!(insert_field(&mut draft, 0..0, parameter("d1")), Some(3..3));
    let fields = draft.formatting.linked_fields.as_deref().unwrap();
    assert_eq!((fields[0].at, fields[0].parameter.as_str()), (0, "d1"));
    assert_eq!((fields[1].at, fields[1].parameter.as_str()), (2, "d0"));
}

#[test]
fn replacing_a_selected_field_keeps_the_other_field_identity() {
    let mut draft = GuiSketchTextDraft::default();
    insert_field(&mut draft, 0..0, parameter("d0")).unwrap();
    insert_field(&mut draft, 3..3, parameter("d1")).unwrap();
    insert_field(&mut draft, 0..3, parameter("d2")).unwrap();
    let fields = draft.formatting.linked_fields.as_deref().unwrap();
    assert_eq!((fields[0].at, fields[0].parameter.as_str()), (0, "d2"));
    assert_eq!((fields[1].at, fields[1].parameter.as_str()), (1, "d1"));
}

#[test]
fn literal_edits_move_fields_and_removing_a_placeholder_removes_the_link() {
    let mut draft = GuiSketchTextDraft::default();
    insert_field(&mut draft, 0..0, parameter("d0")).unwrap();
    let before = draft.content.clone();
    draft.content.insert(0, '界');
    retain_unchanged(&mut draft, &before);
    assert_eq!(draft.formatting.linked_fields.as_ref().unwrap()[0].at, 1);
    let before = draft.content.clone();
    draft.content.truncate("界".len());
    retain_unchanged(&mut draft, &before);
    assert!(draft.formatting.linked_fields.is_none());
}

#[test]
fn backspace_removes_the_field_before_the_actual_caret() {
    let mut draft = GuiSketchTextDraft::default();
    insert_field(&mut draft, 0..0, parameter("left")).unwrap();
    insert_field(&mut draft, 3..3, parameter("right")).unwrap();
    let before = draft.content.clone();
    draft.content.replace_range(..3, "");
    retain_after_input(&mut draft, &before, 3..3, 0);
    let [field] = draft.formatting.linked_fields.as_deref().unwrap() else {
        panic!("expected one linked field after Backspace");
    };
    assert_eq!((field.at, field.parameter.as_str()), (0, "right"));
}

#[test]
fn field_choices_keep_empty_finish_separate_from_file_name() {
    let sources = GuiSketchTextFieldSources::default();
    let mut controls = LinkedFieldControls {
        kind: SourceKind::Finish,
        field_name: Some(FILENAME.into()),
        ..Default::default()
    };
    assert!(
        controls
            .resolve_selection(controls.kind.names(&sources))
            .is_none()
    );
    controls.kind = SourceKind::Standard;
    let field = controls
        .resolve_selection(controls.kind.names(&sources))
        .unwrap();
    assert_eq!(field.kind, SKETCH_TEXT_FIELD_FILE_NAME);
    assert!(field.precision.is_none());
}

fn parameter(name: &str) -> SketchTextField {
    SketchTextField {
        kind: SKETCH_TEXT_FIELD_MODEL_PARAMETER,
        parameter: name.into(),
        ..Default::default()
    }
}
