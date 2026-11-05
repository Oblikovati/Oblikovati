// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_script::editor::{EditorPosition, ScriptEditor};
fn position(line: usize, column: usize) -> EditorPosition {
    EditorPosition { line, column }
}
#[test]
fn word_right_uses_identifier_punctuation_blank_and_line_boundaries() {
    let mut editor = ScriptEditor::new("foo.bar baz\nλa_2");
    for expected in [
        position(0, 3),
        position(0, 4),
        position(0, 8),
        position(1, 0),
        position(1, 1),
        position(1, 4),
        position(1, 4),
    ] {
        editor.move_word_right(false);
        assert_eq!(editor.selection().caret, expected);
    }
    editor.set_caret(position(0, 7), false);
    editor.move_word_right(false);
    assert_eq!(editor.selection().caret, position(0, 8));
    assert!(!editor.can_undo());
}
#[test]
fn word_left_skips_blanks_keeps_unicode_character_columns_and_extends_anchor() {
    let mut editor = ScriptEditor::new("foo bar\nλa_2");
    editor.move_document_end(false);
    for expected in [
        position(1, 1),
        position(1, 0),
        position(0, 7),
        position(0, 4),
        position(0, 0),
        position(0, 0),
    ] {
        editor.move_word_left(false);
        assert_eq!(editor.selection().caret, expected);
    }
    editor.set_caret(position(0, 4), false);
    editor.move_word_right(true);
    assert_eq!(editor.selected_text(), "bar\n");
    verify_word_left_blanks(&mut editor);
}
fn verify_word_left_blanks(editor: &mut ScriptEditor) {
    editor.set_text("   a");
    editor.set_caret(position(0, 3), false);
    editor.move_word_left(false);
    assert_eq!(editor.selection().caret, position(0, 0));
}

#[test]
fn double_click_selects_ascii_identifiers_and_collapses_off_word_hits() {
    let mut editor = ScriptEditor::new("λfoo_2 .bar");
    editor.select_word(position(0, 3));
    assert_eq!(editor.selected_text(), "foo_2");
    editor.select_word(position(0, 0));
    assert!(editor.selection().is_empty());
    editor.select_word(position(0, 6));
    assert!(editor.selection().is_empty());
    editor.select_word(position(0, 7));
    assert!(editor.selection().is_empty());
    editor.select_word(position(usize::MAX, usize::MAX));
    assert!(editor.selection().is_empty());
}
#[test]
fn comments_use_shared_indent_skip_blank_lines_and_toggle_as_single_history_steps() {
    let source = "    foo\n      bar\n\n \t";
    let mut editor = ScriptEditor::new(source);
    editor.select_all();
    editor.toggle_line_comment();
    assert_eq!(editor.text(), "    -- foo\n    --   bar\n\n \t");
    editor.toggle_line_comment();
    assert_eq!(editor.text(), source);
    editor.undo();
    assert!(editor.text().contains("-- foo"));
    editor.undo();
    assert_eq!(editor.text(), source);
    editor.redo();
    assert!(editor.text().contains("-- foo"));
}
#[test]
fn mixed_comments_are_nested_and_bare_prefixes_uncomment_without_losing_indent() {
    let mut editor = ScriptEditor::new("-- a\nb");
    editor.select_all();
    editor.toggle_line_comment();
    assert_eq!(editor.text(), "-- -- a\n-- b");
    editor.set_text(" \t--foo\n -- bar");
    editor.select_all();
    editor.toggle_line_comment();
    assert_eq!(editor.text(), " \tfoo\n bar");
    editor.set_text("single");
    editor.toggle_line_comment();
    assert_eq!(editor.text(), "-- single");
}
#[test]
fn block_indent_outdent_keeps_selection_and_excludes_final_column_zero_line() {
    let mut editor = ScriptEditor::new("one\ntwo\nthree");
    editor.set_caret(position(2, 0), false);
    editor.set_caret(position(0, 1), true);
    editor.indent_selection();
    assert_eq!(editor.text(), "    one\n    two\nthree");
    assert_eq!(editor.selected_text(), "    one\n    two");
    editor.outdent_selection();
    assert_eq!(editor.text(), "one\ntwo\nthree");
    editor.undo();
    assert_eq!(editor.text(), "    one\n    two\nthree");
    editor.undo();
    assert_eq!(editor.text(), "one\ntwo\nthree");
}
#[test]
fn partial_space_tab_and_blank_block_operations_keep_go_noop_rules() {
    let mut editor = ScriptEditor::new("  two\n\tone\n     five");
    editor.select_all();
    editor.outdent_selection();
    assert_eq!(editor.text(), "two\none\n five");
    editor.set_text("");
    editor.outdent_selection();
    editor.toggle_line_comment();
    assert_eq!(editor.text(), "");
    assert!(!editor.can_undo());
    editor.set_text("\n\t");
    editor.select_all();
    editor.toggle_line_comment();
    assert_eq!(editor.text(), "\n\t");
    editor.undo();
    assert_eq!(editor.text(), "\n\t");
}
#[test]
fn comment_columns_count_characters_and_whitespace_only_lines_remain_unchanged() {
    let mut editor = ScriptEditor::new("\tλ\n  μ\n\u{a0}");
    editor.select_all();
    editor.toggle_line_comment();
    assert_eq!(editor.text(), "\t-- λ\n --  μ\n\u{a0}");
    editor.undo();
    assert_eq!(editor.text(), "\tλ\n  μ\n\u{a0}");
}
