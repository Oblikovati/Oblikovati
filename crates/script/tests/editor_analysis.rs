// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_script::editor::{EditorPosition, ScriptEditor};
fn position(line: usize, column: usize) -> EditorPosition {
    EditorPosition { line, column }
}
fn marked_editor(source: &str) -> ScriptEditor {
    let offset = source.find('|').expect("one caret marker");
    let prefix = &source[..offset];
    let line = prefix.matches('\n').count();
    let column = prefix.rsplit('\n').next().unwrap().chars().count();
    let mut editor = ScriptEditor::new(&source.replacen('|', "", 1));
    editor.set_caret(position(line, column), false);
    editor
}
#[test]
fn brackets_match_nested_pairs_in_each_direction_and_at_end_of_line() {
    for (source, first, last) in [
        ("f|((a), b)", position(0, 1), position(0, 8)),
        ("foo(bar)|", position(0, 7), position(0, 3)),
        ("|[[]]", position(0, 0), position(0, 3)),
        ("[[]]|", position(0, 3), position(0, 0)),
        ("|{a{b}}", position(0, 0), position(0, 5)),
        ("{a{b}}|", position(0, 5), position(0, 0)),
    ] {
        assert_eq!(
            marked_editor(source).matching_bracket(),
            Some((first, last)),
            "{source}"
        );
    }
}
#[test]
fn brackets_cross_empty_lines_and_preserve_caret_first_priority() {
    let editor = marked_editor("λg(|{\n\n x=1\n})");
    assert_eq!(
        editor.matching_bracket(),
        Some((position(0, 3), position(3, 0)))
    );
    let editor = marked_editor("(|)[]");
    assert_eq!(
        editor.matching_bracket(),
        Some((position(0, 1), position(0, 0)))
    );
    let editor = marked_editor("(|x)\n");
    assert_eq!(
        editor.matching_bracket(),
        Some((position(0, 0), position(0, 2)))
    );
}
#[test]
fn missing_unbalanced_and_boundary_brackets_have_no_match() {
    for source in ["|", "|text", "x | = 1", "f|(a", "a)|)", "|(", ")|"] {
        assert_eq!(marked_editor(source).matching_bracket(), None, "{source}");
    }
    assert_eq!(
        marked_editor("()|(").matching_bracket(),
        Some((position(0, 1), position(0, 0)))
    );
}
#[test]
fn lexical_bracket_behavior_keeps_go_strings_comments_and_mixed_kind_rules() {
    assert_eq!(
        marked_editor("|(\" )\"").matching_bracket(),
        Some((position(0, 0), position(0, 3)))
    );
    assert_eq!(
        marked_editor("|(-- )").matching_bracket(),
        Some((position(0, 0), position(0, 4)))
    );
    assert_eq!(
        marked_editor("|([)]").matching_bracket(),
        Some((position(0, 0), position(0, 2)))
    );
}
#[test]
fn enclosing_call_covers_tables_nested_calls_and_unicode_columns() {
    for (source, name) in [
        (
            "oblikovati.sketch.rectangle{ width = |40 }",
            "oblikovati.sketch.rectangle",
        ),
        ("pcall(function() print(|x) end)", "print"),
        ("outer(inner()) + outer(|x)", "outer"),
        ("λ = api._group.method2{|x}", "api._group.method2"),
        ("f({x=1}, |x)", "f"),
    ] {
        assert_eq!(
            marked_editor(source).enclosing_call().unwrap().join("."),
            name,
            "{source}"
        );
    }
}
#[test]
fn enclosing_call_declines_closed_anonymous_spaced_and_previous_line_calls() {
    for source in [
        "local x = 1|0",
        "foo() |bar",
        "y = (|1 + 2)",
        "foo (|x)",
        "f(\n|x)",
        "|foo(x)",
        "f((|x))",
    ] {
        assert_eq!(marked_editor(source).enclosing_call(), None, "{source}");
    }
}
#[test]
fn hovered_chain_ends_at_its_segment_and_accepts_just_after_the_word() {
    let editor = ScriptEditor::new("x = oblikovati.documents.create{}");
    for (column, expected) in [
        (27, "oblikovati.documents.create"),
        (18, "oblikovati.documents"),
        (23, "oblikovati.documents"),
        (30, "oblikovati.documents.create"),
    ] {
        assert_eq!(
            editor
                .method_chain_at(position(0, column))
                .unwrap()
                .join("."),
            expected
        );
    }
    verify_unicode_hover();
}
#[test]
fn hovered_chain_refuses_out_of_range_hits_and_non_identifiers() {
    let editor = ScriptEditor::new("a = 1\nλ");
    for hit in [
        position(0, 2),
        position(0, 6),
        position(2, 0),
        position(usize::MAX, usize::MAX),
        position(1, 0),
    ] {
        assert_eq!(editor.method_chain_at(hit), None, "{hit:?}");
    }
    assert_eq!(ScriptEditor::new("").method_chain_at(position(0, 0)), None);
    assert_eq!(
        ScriptEditor::new("a..b")
            .method_chain_at(position(0, 3))
            .unwrap(),
        ["b"]
    );
}
#[test]
fn analysis_is_read_only_and_tracks_edits_undo_redo_and_reload() {
    let mut editor = marked_editor("api.f(|x)");
    let before = editor.selection();
    assert_eq!(editor.enclosing_call().unwrap(), ["api", "f"]);
    assert!(editor.matching_bracket().is_some());
    assert_eq!(editor.selection(), before);
    assert!(!editor.can_undo());
    editor.insert(")");
    assert_eq!(editor.enclosing_call(), None);
    editor.undo();
    assert_eq!(editor.enclosing_call().unwrap(), ["api", "f"]);
    editor.redo();
    assert_eq!(editor.enclosing_call(), None);
    editor.set_text("none");
    assert_eq!(editor.matching_bracket(), None);
}

fn verify_unicode_hover() {
    let editor = ScriptEditor::new("λ api._item2");
    assert_eq!(
        editor.method_chain_at(position(0, 10)).unwrap(),
        ["api", "_item2"]
    );
}
