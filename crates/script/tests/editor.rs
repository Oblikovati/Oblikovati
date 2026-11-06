// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_script::editor::{EditorPosition, ScriptEditor};
fn position(line: usize, column: usize) -> EditorPosition {
    EditorPosition { line, column }
}
#[test]
fn source_load_normalizes_crlf_and_counts_unicode_columns() {
    let mut editor = ScriptEditor::new("αβ\r\n世界\r\n");
    assert_eq!(editor.text(), "αβ\n世界\n");
    assert_eq!(editor.line_count(), 3);
    assert_eq!(editor.line(1).unwrap(), "世界");
    assert!(editor.line(3).unwrap_err().contains('3'));
    editor.set_caret(position(0, 1), false);
    editor.insert("é");
    assert_eq!(editor.text(), "αéβ\n世界\n");
    assert_eq!(editor.selection().caret, position(0, 2));
    editor.undo();
    assert_eq!(editor.text(), "αβ\n世界\n");
    editor.redo();
    assert_eq!(editor.text(), "αéβ\n世界\n");
}
#[test]
fn reversed_selection_paste_is_one_reversible_multiline_replacement() {
    let mut editor = ScriptEditor::new("alpha\nbeta\ngamma");
    editor.set_caret(position(2, 2), false);
    editor.set_caret(position(0, 2), true);
    assert_eq!(editor.selected_text(), "pha\nbeta\nga");
    editor.insert("X\nY");
    assert_eq!(editor.text(), "alX\nYmma");
    assert_eq!(editor.selection().caret, position(1, 1));
    assert!(editor.selection().is_empty());
    editor.undo();
    assert_eq!(editor.text(), "alpha\nbeta\ngamma");
    editor.redo();
    assert_eq!(editor.text(), "alX\nYmma");
}
#[test]
fn typed_words_coalesce_but_whitespace_and_paste_form_new_steps() {
    let mut editor = ScriptEditor::new("");
    for character in ["f", "o", "o", " ", "b", "a", "r"] {
        editor.insert(character);
    }
    editor.undo();
    assert_eq!(editor.text(), "foo ");
    editor.undo();
    assert_eq!(editor.text(), "foo");
    editor.undo();
    assert_eq!(editor.text(), "");
    assert!(!editor.can_undo());
    verify_redo_and_new_edit(&mut editor);
}
fn verify_redo_and_new_edit(editor: &mut ScriptEditor) {
    editor.redo();
    assert_eq!(editor.text(), "foo");
    editor.insert("pasted");
    editor.undo();
    assert_eq!(editor.text(), "foo");
    assert!(editor.can_redo());
    editor.insert("x");
    assert!(!editor.can_redo());
}

#[test]
fn backward_and_forward_unicode_deletions_coalesce_and_line_joins_restore() {
    let mut editor = ScriptEditor::new("αβ\nγδ");
    editor.set_caret(position(1, 0), false);
    editor.backspace();
    assert_eq!(editor.text(), "αβγδ");
    editor.undo();
    assert_eq!(editor.text(), "αβ\nγδ");
    editor.set_caret(position(0, 2), false);
    editor.delete();
    assert_eq!(editor.text(), "αβγδ");
    editor.undo();
    assert_eq!(editor.text(), "αβ\nγδ");
    verify_coalesced_deletions(&mut editor);
}
fn verify_coalesced_deletions(editor: &mut ScriptEditor) {
    editor.set_caret(position(1, 2), false);
    editor.backspace();
    editor.backspace();
    editor.undo();
    assert_eq!(editor.text(), "αβ\nγδ");
    editor.set_caret(position(0, 0), false);
    editor.delete();
    editor.delete();
    editor.undo();
    assert_eq!(editor.text(), "αβ\nγδ");
}

#[test]
fn auto_indent_selection_delete_and_empty_boundary_edits_match_go() {
    let mut editor = ScriptEditor::new(" \tfoo");
    editor.backspace();
    editor.insert("");
    editor.undo();
    editor.redo();
    assert!(!editor.can_undo());
    editor.set_caret(position(0, 5), false);
    editor.newline();
    assert_eq!(editor.text(), " \tfoo\n \t");
    verify_clear_and_reload(&mut editor);
}
fn verify_clear_and_reload(editor: &mut ScriptEditor) {
    editor.select_all();
    editor.delete();
    assert_eq!(editor.text(), "");
    editor.delete();
    assert_eq!(editor.text(), "");
    editor.undo();
    assert_eq!(editor.text(), " \tfoo\n \t");
    editor.select_all();
    editor.backspace();
    assert_eq!(editor.text(), "");
    editor.set_text("fresh");
    assert!(!editor.can_undo() && !editor.can_redo());
    assert_eq!(editor.selection().caret, position(0, 0));
}

#[test]
fn arrows_collapse_selection_wrap_lines_and_vertical_goal_survives_short_lines() {
    let mut editor = ScriptEditor::new("abcdef\nx\nabcdef");
    editor.set_caret(position(0, 5), false);
    editor.move_down(false);
    assert_eq!(editor.selection().caret, position(1, 1));
    editor.move_down(false);
    assert_eq!(editor.selection().caret, position(2, 5));
    editor.move_down(false);
    assert_eq!(editor.selection().caret, position(2, 6));
    editor.move_up(false);
    editor.move_up(false);
    assert_eq!(editor.selection().caret, position(0, 5));
    editor.move_up(false);
    assert_eq!(editor.selection().caret, position(0, 0));
    editor.move_left(false);
    assert_eq!(editor.selection().caret, position(0, 0));
    verify_arrow_selection(&mut editor);
}
fn verify_arrow_selection(editor: &mut ScriptEditor) {
    editor.set_caret(position(0, 6), false);
    editor.move_right(true);
    assert_eq!(editor.selected_text(), "\n");
    editor.move_left(false);
    assert_eq!(editor.selection().caret, position(0, 6));
    editor.move_right(true);
    editor.move_right(false);
    assert_eq!(editor.selection().caret, position(1, 0));
    editor.move_left(true);
    assert_eq!(editor.selected_text(), "\n");
    editor.set_caret(position(usize::MAX, usize::MAX), false);
    assert_eq!(editor.selection().caret, position(2, 6));
    editor.move_right(false);
    assert_eq!(editor.selection().caret, position(2, 6));
    editor.set_caret(position(0, usize::MAX), false);
    assert_eq!(editor.selection().caret, position(0, 6));
}

#[test]
fn smart_home_end_and_document_bounds_extend_without_changing_source() {
    let mut editor = ScriptEditor::new("  alpha\n\tbeta");
    editor.move_end(false);
    assert_eq!(editor.selection().caret, position(0, 7));
    editor.move_home(false);
    assert_eq!(editor.selection().caret, position(0, 2));
    editor.move_home(false);
    assert_eq!(editor.selection().caret, position(0, 0));
    editor.move_document_end(true);
    assert_eq!(editor.selected_text(), editor.text());
    editor.move_document_start(false);
    assert!(editor.selection().is_empty());
    assert!(!editor.can_undo());
}
#[test]
fn edits_at_separate_positions_and_replacements_do_not_merge_unrelated_changes() {
    let mut editor = ScriptEditor::new("abcd");
    editor.insert("X");
    editor.set_caret(position(0, 4), false);
    editor.insert("Y");
    editor.undo();
    assert_eq!(editor.text(), "Xabcd");
    editor.undo();
    assert_eq!(editor.text(), "abcd");
    editor.set_caret(position(0, 1), false);
    editor.set_caret(position(0, 3), true);
    editor.insert("Z");
    editor.insert("Q");
    editor.undo();
    assert_eq!(editor.text(), "aZd");
    editor.undo();
    assert_eq!(editor.text(), "abcd");
}
