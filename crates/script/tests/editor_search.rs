// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_script::editor::{EditorPosition, EditorSearch, ScriptEditor};
fn position(line: usize, column: usize) -> EditorPosition {
    EditorPosition { line, column }
}
#[test]
fn literal_find_is_case_sensitive_nonoverlapping_line_ordered_and_unicode_addressed() {
    let editor = ScriptEditor::new("λfoo bar foo\nfoofoo\nFoo");
    let matches = editor.find("foo");
    verify_ordered_spans(&matches);
    assert!(editor.find("").is_empty());
    assert!(editor.find("foo\nfoo").is_empty());
    assert_eq!(ScriptEditor::new("aaa").find("aa").len(), 1);
    assert_eq!(ScriptEditor::new("éé").find("é")[1].start(), position(0, 1));
}
#[test]
fn selecting_matches_and_replacing_one_reuses_the_existing_edit_history() {
    let mut editor = ScriptEditor::new("alpha beta beta");
    let matches = editor.find("beta");
    editor.select_match(&matches[1]).unwrap();
    assert_eq!(editor.selected_text(), "beta");
    editor.insert("γ");
    assert_eq!(editor.text(), "alpha beta γ");
    editor.undo();
    assert_eq!(editor.text(), "alpha beta beta");
    editor.redo();
    assert_eq!(editor.text(), "alpha beta γ");
}
#[test]
fn old_and_foreign_matches_decline_without_changing_the_selection() {
    let mut editor = ScriptEditor::new("match");
    let old = editor.find("match").remove(0);
    let other_editor = ScriptEditor::new("match");
    let foreign = other_editor.find("match").remove(0);
    let original = editor.selection();
    assert!(editor.select_match(&foreign).is_err());
    assert_eq!(editor.selection(), original);
    editor.insert("x");
    let changed = editor.selection();
    assert!(editor.select_match(&old).is_err());
    assert_eq!(editor.selection(), changed);
    editor.undo();
    assert!(editor.select_match(&old).is_err());
    editor.set_text("match");
    assert!(editor.select_match(&old).is_err());
}
#[test]
fn replace_all_is_one_edit_and_preserves_go_whole_source_multiline_behavior() {
    let mut editor = ScriptEditor::new("x = x + x\na\nb");
    assert_eq!(editor.replace_all("x", "variable"), 3);
    assert_eq!(editor.text(), "variable = variable + variable\na\nb");
    editor.undo();
    assert_eq!(editor.text(), "x = x + x\na\nb");
    editor.redo();
    assert_eq!(editor.text(), "variable = variable + variable\na\nb");
    assert!(editor.find("a\nb").is_empty());
    assert_eq!(editor.replace_all("a\nb", "joined"), 1);
    assert!(editor.text().ends_with("joined"));
    editor.undo();
    assert!(editor.text().ends_with("a\nb"));
}
#[test]
fn empty_and_missing_queries_leave_history_and_generation_untouched() {
    let mut editor = ScriptEditor::new("abc");
    let matched = editor.find("abc").remove(0);
    assert_eq!(editor.replace_all("", "x"), 0);
    assert_eq!(editor.replace_all("missing", "x"), 0);
    assert!(!editor.can_undo());
    editor.select_match(&matched).unwrap();
    assert_eq!(editor.replace_all("abc", "abc"), 1);
    assert!(editor.can_undo());
    assert!(editor.select_match(&matched).is_err());
}
#[test]
fn next_previous_wrap_and_query_changes_select_the_first_current_hit() {
    let mut editor = ScriptEditor::new("a a a");
    let mut search = EditorSearch::default();
    assert_eq!(search.ordinal(&editor), (0, 0));
    assert!(search.highlights(&editor).is_empty());
    search.set_query(&mut editor, "a");
    assert_eq!(search.ordinal(&editor), (1, 3));
    assert!(search.previous(&mut editor));
    assert_eq!(search.ordinal(&editor), (3, 3));
    assert!(search.next(&mut editor));
    assert_eq!(search.ordinal(&editor), (1, 3));
    search.next(&mut editor);
    assert_eq!(search.ordinal(&editor), (2, 3));
    search.set_query(&mut editor, "missing");
    assert!(!search.next(&mut editor));
    assert!(!search.previous(&mut editor));
    assert_eq!(search.ordinal(&editor), (0, 0));
}
#[test]
fn source_edits_invalidate_highlights_and_next_refreshes_without_stale_selection() {
    let mut editor = ScriptEditor::new("a a");
    let mut search = EditorSearch::default();
    search.set_query(&mut editor, "a");
    let old = search.highlights(&editor);
    editor.insert("removed");
    assert!(search.highlights(&editor).is_empty());
    assert_eq!(search.ordinal(&editor), (0, 0));
    assert!(search.next(&mut editor));
    assert_eq!(search.ordinal(&editor), (1, 1));
    assert_eq!(editor.selection().caret, position(0, 9));
    assert!(editor.select_match(&old[0]).is_err());
}
#[test]
fn find_bar_replace_all_refreshes_including_replacements_containing_the_query() {
    let mut editor = ScriptEditor::new("a a");
    let mut search = EditorSearch::default();
    search.set_query(&mut editor, "a");
    assert_eq!(search.replace_all(&mut editor, "aa"), 2);
    assert_eq!(search.ordinal(&editor), (1, 4));
    assert_eq!(search.highlights(&editor).len(), 4);
    assert_eq!(search.replace_all(&mut editor, "b"), 4);
    assert_eq!(search.ordinal(&editor), (0, 0));
    assert_eq!(editor.text(), "bb bb");
    editor.undo();
    assert!(search.next(&mut editor));
    assert_eq!(search.ordinal(&editor), (1, 4));
}

fn verify_ordered_spans(matches: &[oblikovati_script::editor::EditorMatch]) {
    let spans: Vec<_> = matches
        .iter()
        .map(|matched| (matched.start(), matched.end()))
        .collect();
    assert_eq!(
        spans,
        [
            (position(0, 1), position(0, 4)),
            (position(0, 9), position(0, 12)),
            (position(1, 0), position(1, 3)),
            (position(1, 3), position(1, 6))
        ]
    );
}
