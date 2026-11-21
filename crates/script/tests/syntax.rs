// SPDX-License-Identifier: GPL-2.0-only
use LuaTokenKind::{
    Builtin, Comment, Identifier, Keyword, Number, Operator, String as StringToken,
};
use oblikovati_script::{
    editor::{CompletionEngine, CompletionKind},
    syntax::{LuaLineState, LuaToken, LuaTokenKind, tokenize_line},
};
fn spans(line: &str, state: LuaLineState) -> (Vec<(LuaTokenKind, String)>, LuaLineState) {
    let characters: Vec<char> = line.chars().collect();
    let (tokens, carry) = tokenize_line(line, state);
    let result = tokens
        .iter()
        .map(|token| {
            (
                token.kind,
                characters[token.start..token.end].iter().collect(),
            )
        })
        .collect();
    (result, carry)
}
fn verify(line: &str, expected: &[(LuaTokenKind, &str)]) {
    let expected: Vec<_> = expected
        .iter()
        .map(|(kind, text)| (*kind, (*text).to_owned()))
        .collect();
    assert_eq!(
        spans(line, LuaLineState::Plain),
        (expected, LuaLineState::Plain),
        "{line}"
    );
}
#[test]
fn keywords_builtins_and_identifiers_match_go_token_shapes() {
    verify(
        "local x = print(foo)",
        &[
            (Keyword, "local"),
            (Identifier, "x"),
            (Operator, "="),
            (Builtin, "print"),
            (Operator, "("),
            (Identifier, "foo"),
            (Operator, ")"),
        ],
    );
}
#[test]
fn unavailable_globals_and_lua54_words_are_plain_identifiers() {
    verify(
        "_item2 GOTO goto require rawlen",
        &[
            (Identifier, "_item2"),
            (Identifier, "GOTO"),
            (Identifier, "goto"),
            (Identifier, "require"),
            (Identifier, "rawlen"),
        ],
    );
}
#[test]
fn every_completion_keyword_and_builtin_has_the_same_syntax_kind() {
    let candidates = CompletionEngine::new(&[]).suggest("", 0).0;
    for candidate in candidates {
        let kind = match candidate.kind {
            CompletionKind::Keyword => Keyword,
            CompletionKind::Builtin => Builtin,
            _ => panic!("unexpected bare candidate {:?}", candidate.kind),
        };
        verify(&candidate.text, &[(kind, &candidate.text)]);
    }
}
#[test]
fn quoted_strings_skip_escaped_quotes_and_stop_at_line_end() {
    verify(
        r#"s = "a\"b" .. 'c'"#,
        &[
            (Identifier, "s"),
            (Operator, "="),
            (StringToken, r#""a\"b""#),
            (Operator, "."),
            (Operator, "."),
            (StringToken, "'c'"),
        ],
    );
    verify("'unfinished\\", &[(StringToken, "'unfinished\\")]);
    verify("'a\\\\' x", &[(StringToken, "'a\\\\'"), (Identifier, "x")]);
    assert_eq!(
        spans("'unfinished", LuaLineState::Plain).1,
        LuaLineState::Plain
    );
}
#[test]
fn numeric_highlighting_keeps_go_liberal_decimal_and_hex_spans() {
    for text in [
        "0xFF",
        "0X1.afp+3",
        "3.14e-2",
        "1E+4",
        ".5",
        "0x",
        "1e",
        "1..2",
        "0x1.P-2",
    ] {
        verify(text, &[(Number, text)]);
    }
}
#[test]
fn numbers_stop_at_operators_and_nonhex_characters() {
    verify(
        "1+2 - 3",
        &[
            (Number, "1"),
            (Operator, "+"),
            (Number, "2"),
            (Operator, "-"),
            (Number, "3"),
        ],
    );
}
#[test]
fn incomplete_hex_spans_leave_nonhex_characters_for_identifiers() {
    verify(
        ". + 0xG",
        &[
            (Operator, "."),
            (Operator, "+"),
            (Number, "0x"),
            (Identifier, "G"),
        ],
    );
}
#[test]
fn line_comments_consume_remaining_text_and_invalid_long_openers() {
    verify(
        "x = 1 -- trailing note",
        &[
            (Identifier, "x"),
            (Operator, "="),
            (Number, "1"),
            (Comment, "-- trailing note"),
        ],
    );
    for text in ["--", "--[=oops", "--[", "-- '", "--[[closed]]"] {
        verify(text, &[(Comment, text)]);
    }
}
#[test]
fn long_comments_carry_level_across_empty_lines_and_resume_normal_tokens() {
    let (tokens, state) = spans("--[==[ open", LuaLineState::Plain);
    assert_eq!(tokens, [(Comment, "--[==[ open".into())]);
    assert_eq!(state, LuaLineState::LongComment { level: 2 });
    assert_eq!(spans("", state), (Vec::new(), state));
    assert_eq!(spans("wrong ]=] closer", state).1, state);
    let (tokens, state) = spans("done ]==] local x", state);
    assert_eq!(
        tokens,
        [
            (Comment, "done ]==]".into()),
            (Keyword, "local".into()),
            (Identifier, "x".into())
        ]
    );
    assert_eq!(state, LuaLineState::Plain);
}
#[test]
fn long_strings_match_exact_levels_and_open_another_construct_after_closing() {
    verify(
        "s = [[raw \"text\"]]",
        &[
            (Identifier, "s"),
            (Operator, "="),
            (StringToken, "[[raw \"text\"]]"),
        ],
    );
}
#[test]
fn closing_one_long_construct_can_open_another_on_the_same_line() {
    let (_, state) = spans("[=[body", LuaLineState::Plain);
    assert_eq!(state, LuaLineState::LongString { level: 1 });
    let (tokens, state) = spans("]=] --[[comment", state);
    assert_eq!(
        tokens,
        [(StringToken, "]=]".into()), (Comment, "--[[comment".into())]
    );
    assert_eq!(state, LuaLineState::LongComment { level: 0 });
    let (tokens, state) = spans("]] + 2", state);
    assert_eq!(
        tokens,
        [
            (Comment, "]]".into()),
            (Operator, "+".into()),
            (Number, "2".into())
        ]
    );
    assert_eq!(state, LuaLineState::Plain);
}
#[test]
fn invalid_bracket_openers_and_single_rune_operators_stay_separate() {
    verify(
        "[=x] [] ...",
        &[
            (Operator, "["),
            (Operator, "="),
            (Identifier, "x"),
            (Operator, "]"),
            (Operator, "["),
            (Operator, "]"),
            (Operator, "."),
            (Operator, "."),
            (Operator, "."),
        ],
    );
    verify("[", &[(Operator, "[")]);
    verify("-", &[(Operator, "-")]);
}
#[test]
fn unicode_columns_and_go_ascii_whitespace_rules_are_preserved() {
    let (tokens, state) = tokenize_line("λ = 'é'\t x", LuaLineState::Plain);
    assert_eq!(state, LuaLineState::Plain);
    for (index, kind, start, end) in [
        (0, Operator, 0, 1),
        (2, StringToken, 4, 7),
        (3, Identifier, 9, 10),
    ] {
        assert_eq!(tokens[index], LuaToken { kind, start, end });
    }
    verify(" \t", &[]);
    verify("\u{a0}", &[(Operator, "\u{a0}")]);
}
#[test]
fn arbitrary_carried_levels_and_empty_input_are_bounded() {
    for state in [
        LuaLineState::Plain,
        LuaLineState::LongString { level: usize::MAX },
        LuaLineState::LongComment { level: usize::MAX },
    ] {
        assert_eq!(tokenize_line("", state), (Vec::new(), state));
        let (_, result) = tokenize_line("]]", state);
        assert_eq!(result, state);
    }
}
#[test]
fn generated_source_and_long_bracket_levels_have_ordered_in_bounds_tokens() {
    for level in 0..20 {
        let equal = "=".repeat(level);
        let source = format!("local s = [{equal}[λ{equal} ]{equal}] + .2e-3 -- tail");
        verify_token_bounds(&source, LuaLineState::Plain);
    }
    let alphabet = [
        '[', ']', '=', '\'', '"', '\\', '-', '0', 'x', '.', 'e', '+', ' ', 'λ',
    ];
    for first in alphabet {
        for second in alphabet {
            verify_token_bounds(&format!("{first}{second}[] '\\' 0xF"), LuaLineState::Plain);
        }
    }
}
fn verify_token_bounds(source: &str, incoming: LuaLineState) {
    let (tokens, _) = tokenize_line(source, incoming);
    let length = source.chars().count();
    let mut previous = 0;
    for token in tokens {
        assert!(
            previous <= token.start && token.start < token.end && token.end <= length,
            "{source:?}: {token:?}"
        );
        previous = token.end;
    }
}

#[test]
fn visible_lines_inherit_context_and_edits_refresh_the_following_lines() {
    use oblikovati_script::editor::{EditorPosition, ScriptEditor};
    let mut editor = ScriptEditor::new("--[[\nlocal x = 1\n]]\nprint(x)");
    let before = editor.selection();
    assert_eq!(editor.highlight_lines(1, 2).unwrap()[0][0].kind, Comment);
    assert_eq!(editor.highlight_lines(3, 4).unwrap()[0][0].kind, Builtin);
    assert_eq!(editor.selection(), before);
    assert!(!editor.can_undo());
    editor.set_caret(EditorPosition { line: 0, column: 4 }, true);
    editor.delete();
    assert_eq!(editor.highlight_lines(1, 2).unwrap()[0][0].kind, Keyword);
    editor.undo();
    assert_eq!(editor.highlight_lines(1, 2).unwrap()[0][0].kind, Comment);
    editor.redo();
    assert_eq!(editor.highlight_lines(1, 2).unwrap()[0][0].kind, Keyword);
}
#[test]
fn invalid_view_ranges_are_refused_and_empty_ranges_return_no_lines() {
    use oblikovati_script::editor::ScriptEditor;
    let editor = ScriptEditor::new("");
    assert_eq!(editor.highlight_lines(0, 1).unwrap(), [Vec::new()]);
    assert!(editor.highlight_lines(1, 1).unwrap().is_empty());
    for (first, last) in [(2, 1), (0, 2), (usize::MAX, usize::MAX)] {
        let error = editor.highlight_lines(first, last).unwrap_err();
        assert!(error.contains(&format!("{first}..{last}")) && error.contains("0..=1"));
    }
}
