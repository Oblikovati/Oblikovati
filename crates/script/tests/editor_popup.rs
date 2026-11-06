// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_script::{
    editor::{
        CompletionEngine, CompletionKey, CompletionKind, CompletionPopup, EditorPosition,
        ScriptEditor,
    },
    help::MethodHelpCatalog,
};
fn source_editor(source: &str) -> ScriptEditor {
    let mut editor = ScriptEditor::new(source);
    editor.move_document_end(false);
    editor
}
fn popup() -> CompletionPopup {
    CompletionPopup::new(CompletionEngine::new(
        &["documents.create", "documents.list", "sketch.circle"].map(String::from),
    ))
}
#[test]
fn empty_prefix_requires_explicit_trigger_and_missing_candidates_stay_hidden() {
    let editor = source_editor("oblikovati.");
    let mut popup = popup();
    assert!(!popup.visible(&editor));
    popup.refresh(&editor, false);
    assert!(popup.view(&editor).is_none());
    popup.refresh(&editor, true);
    assert_eq!(popup.view(&editor).unwrap().candidates.len(), 2);
    let editor = source_editor("oblikovati.unknown.");
    popup.refresh(&editor, true);
    assert!(popup.view(&editor).is_none());
}
#[test]
fn typed_prefix_opens_popup_and_selection_clamps_at_both_ends() {
    let mut editor = source_editor("oblikovati.");
    let mut popup = popup();
    popup.refresh(&editor, true);
    assert!(popup.handle_key(&mut editor, CompletionKey::Up).unwrap());
    assert_eq!(popup.view(&editor).unwrap().selected, 0);
    for _ in 0..4 {
        popup.handle_key(&mut editor, CompletionKey::Down).unwrap();
    }
    assert_eq!(popup.view(&editor).unwrap().selected, 1);
    editor.insert("sk");
    popup.refresh(&editor, false);
    assert!(popup.visible(&editor));
    assert_eq!(popup.view(&editor).unwrap().selected, 0);
}
#[test]
fn refresh_keeps_a_valid_index_and_replacing_engine_discards_old_capabilities() {
    let mut editor = source_editor("oblikovati.");
    let mut popup = popup();
    popup.refresh(&editor, true);
    popup.handle_key(&mut editor, CompletionKey::Down).unwrap();
    popup.refresh(&editor, true);
    assert_eq!(popup.view(&editor).unwrap().selected, 1);
    popup.replace_engine(CompletionEngine::new(&["new.method".into()]));
    assert!(popup.view(&editor).is_none());
    popup.refresh(&editor, true);
    assert_eq!(popup.view(&editor).unwrap().candidates[0].text, "new");
    assert_eq!(popup.view(&editor).unwrap().selected, 0);
}
#[test]
fn enter_and_tab_replace_only_the_prefix_with_one_undoable_edit() {
    for key in [CompletionKey::Enter, CompletionKey::Tab] {
        let mut editor = source_editor("oblikovati.documents.cre");
        let mut popup = popup();
        popup.refresh(&editor, false);
        assert!(popup.handle_key(&mut editor, key).unwrap());
        assert_eq!(editor.text(), "oblikovati.documents.create");
        assert!(!popup.visible(&editor));
        editor.undo();
        assert_eq!(editor.text(), "oblikovati.documents.cre");
        editor.redo();
        assert_eq!(editor.text(), "oblikovati.documents.create");
    }
}
#[test]
fn escape_and_unrelated_keys_preserve_source_and_normal_editor_routing() {
    let mut editor = source_editor("pr");
    let mut popup = popup();
    popup.refresh(&editor, false);
    assert!(!popup.handle_key(&mut editor, CompletionKey::Other).unwrap());
    assert!(popup.visible(&editor));
    assert!(
        popup
            .handle_key(&mut editor, CompletionKey::Escape)
            .unwrap()
    );
    assert_eq!(editor.text(), "pr");
    assert!(!editor.can_undo());
    assert!(!popup.handle_key(&mut editor, CompletionKey::Enter).unwrap());
}
#[test]
fn stale_foreign_or_moved_source_never_draws_or_accepts_old_suggestions() {
    let mut editor = source_editor("pr");
    let mut popup = popup();
    popup.refresh(&editor, false);
    let foreign = source_editor("pr");
    assert!(!popup.visible(&foreign));
    editor.insert("x");
    assert!(!popup.handle_key(&mut editor, CompletionKey::Tab).unwrap());
    assert!(
        popup
            .accept(&mut editor)
            .unwrap_err()
            .contains("visible current")
    );
    editor.undo();
    assert!(!popup.visible(&editor));
    popup.refresh(&editor, false);
    editor.move_left(false);
    assert!(popup.view(&editor).is_none());
}
#[test]
fn visible_rows_follow_selection_and_width_counts_unicode_characters() {
    let mut editor = source_editor("");
    let mut popup = popup();
    popup.refresh(&editor, true);
    assert_eq!(popup.view(&editor).unwrap().shown, 8);
    for _ in 0..12 {
        popup.handle_key(&mut editor, CompletionKey::Down).unwrap();
    }
    let view = popup.view(&editor).unwrap();
    assert_eq!((view.first, view.selected, view.shown), (5, 12, 8));
    assert_eq!(view.anchor, EditorPosition::default());
    let editor = source_editor("oblikovati.");
    popup.replace_engine(CompletionEngine::new(&["éé.method".into()]));
    popup.refresh(&editor, true);
    assert_eq!(popup.view(&editor).unwrap().width_columns, 6);
}
#[test]
fn visible_popup_suppresses_signature_and_hover_help_until_dismissed() {
    let catalog =
        MethodHelpCatalog::published(&[oblikovati_api::wire::METHOD_DOCUMENTS_CREATE.into()])
            .unwrap();
    let editor = source_editor("oblikovati.documents.create{");
    let mut popup = popup();
    popup.refresh(&editor, true);
    let hit = EditorPosition {
        line: 0,
        column: 23,
    };
    assert!(popup.signature_help(&editor, &catalog).is_none());
    assert!(popup.hover_help(&editor, hit, &catalog).is_none());
    popup.dismiss();
    assert!(popup.signature_help(&editor, &catalog).is_some());
    assert!(popup.hover_help(&editor, hit, &catalog).is_some());
}
#[test]
fn candidate_kind_glyphs_match_the_legacy_popup() {
    assert_eq!(
        [
            CompletionKind::Keyword,
            CompletionKind::Builtin,
            CompletionKind::Module,
            CompletionKind::Method
        ]
        .map(CompletionKind::tag),
        ["k", "b", "m", "f"]
    );
}
