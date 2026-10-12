// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_script::editor::{CompletionEngine, CompletionKind, EditorPosition, ScriptEditor};
fn engine() -> CompletionEngine {
    CompletionEngine::new(
        &[
            oblikovati_api::wire::METHOD_DOCUMENTS_ACTIVATE,
            oblikovati_api::wire::METHOD_DOCUMENTS_CLOSE,
            "sketch.rectangle",
            "sketch.circle",
            "sketch.line.add",
        ]
        .map(String::from),
    )
}
fn texts(engine: &CompletionEngine, source: &str) -> Vec<String> {
    engine
        .suggest(source, source.chars().count())
        .0
        .into_iter()
        .map(|item| item.text)
        .collect()
}
fn marked(source: &str) -> ScriptEditor {
    let offset = source.find('|').unwrap();
    let prefix = &source[..offset];
    let line = prefix.matches('\n').count();
    let column = prefix.rsplit('\n').next().unwrap().chars().count();
    let mut editor = ScriptEditor::new(&source.replacen('|', "", 1));
    editor.set_caret(EditorPosition { line, column }, false);
    editor
}
#[test]
fn live_namespace_prefixes_details_and_nested_groups_match_go_order() {
    let engine = engine();
    assert_eq!(texts(&engine, "oblikovati."), ["documents", "sketch"]);
    let (items, start) = engine.suggest("oblikovati.sketch.rec", 20);
    assert_eq!(start, 18);
    assert_eq!(items[0].text, "rectangle");
    assert_eq!(items[0].detail, "oblikovati.sketch.rectangle");
    assert_eq!(items[0].kind, CompletionKind::Method);
    let (items, _) = engine.suggest("oblikovati.sketch.li", 19);
    assert_eq!(items[0].kind, CompletionKind::Module);
    assert_eq!(texts(&engine, "oblikovati.sketch.line."), ["add"]);
    assert_eq!(
        engine.suggest("oblikovati.", 11).0[0].detail,
        "oblikovati.documents"
    );
}
#[test]
fn bare_words_keywords_builtins_and_ascii_case_are_ranked_by_text() {
    let engine = engine();
    let (items, start) = engine.suggest("  pr", 4);
    assert_eq!(start, 2);
    assert_eq!(items[0].text, "print");
    assert_eq!(items[0].kind, CompletionKind::Builtin);
    assert_eq!(engine.suggest("lo", 2).0[0].kind, CompletionKind::Keyword);
    assert_eq!(texts(&engine, "PR"), ["print"]);
    assert_eq!(texts(&engine, "oblikovati.sketch.REC"), ["rectangle"]);
    assert!(texts(&engine, "Oblikovati.sketch.rec").is_empty());
    assert!(texts(&engine, "oblikovati.Sketch.rec").is_empty());
    assert_eq!(texts(&engine, ""), texts(&engine, " "));
    assert!(texts(&engine, "").windows(2).all(|pair| pair[0] < pair[1]));
}
#[test]
fn unknown_paths_concat_and_unicode_boundaries_do_not_join_unrelated_chains() {
    let engine = engine();
    assert!(texts(&engine, "oblikovati.nope.x").is_empty());
    assert!(texts(&engine, "other.thing").is_empty());
    assert_eq!(texts(&engine, "a..pr"), ["print"]);
    assert_eq!(texts(&engine, ".lo"), ["local"]);
    assert_eq!(engine.suggest("λ pr", usize::MAX).1, 2);
    assert_eq!(engine.suggest("λ pr", usize::MAX).0[0].text, "print");
    assert_eq!(engine.suggest("anything", 0).1, 0);
    assert!(texts(&engine, "oblikovati.sketch.λ").len() > 1);
}
#[test]
fn duplicate_methods_and_callable_namespaces_have_one_stable_candidate() {
    let engine =
        CompletionEngine::new(&["group", "group.method", "group.method"].map(String::from));
    let (items, _) = engine.suggest("oblikovati.", 11);
    assert_eq!(items.len(), 1);
    assert_eq!(items[0].kind, CompletionKind::Module);
    assert_eq!(texts(&engine, "oblikovati.group."), ["method"]);
    assert!(texts(&CompletionEngine::new(&[]), "oblikovati.").is_empty());
}
#[test]
fn accepting_a_current_popup_changes_only_prefix_and_records_one_undo_step() {
    let mut editor = marked("λ = 2\noblikovati.sketch.rec|{x=1}");
    let popup = editor.complete(&engine());
    assert_eq!(
        popup.replace_start(),
        EditorPosition {
            line: 1,
            column: 18
        }
    );
    assert_eq!(popup.candidates()[0].text, "rectangle");
    editor.accept_completion(&popup, 0).unwrap();
    assert_eq!(editor.text(), "λ = 2\noblikovati.sketch.rectangle{x=1}");
    editor.undo();
    assert_eq!(editor.text(), "λ = 2\noblikovati.sketch.rec{x=1}");
    editor.redo();
    assert_eq!(editor.text(), "λ = 2\noblikovati.sketch.rectangle{x=1}");
}
#[test]
fn invalid_index_and_moved_caret_decline_without_changing_text_or_history() {
    let mut editor = marked("pr|");
    let popup = editor.complete(&engine());
    let error = editor.accept_completion(&popup, 1).unwrap_err();
    assert!(error.contains("index 1") && error.contains("below 1"));
    assert_eq!(editor.text(), "pr");
    assert!(!editor.can_undo());
    editor.move_left(false);
    assert!(editor.accept_completion(&popup, 0).is_err());
    assert_eq!(editor.text(), "pr");
    let empty = marked("oblikovati.nope.|").complete(&engine());
    assert!(empty.candidates().is_empty());
}
#[test]
fn foreign_and_stale_popups_decline_after_edit_undo_and_reload() {
    let mut editor = marked("pr|");
    let other = marked("pr|");
    let foreign = other.complete(&engine());
    assert!(editor.accept_completion(&foreign, 0).is_err());
    let popup = editor.complete(&engine());
    editor.insert("x");
    assert!(editor.accept_completion(&popup, 0).is_err());
    editor.undo();
    assert!(editor.accept_completion(&popup, 0).is_err());
    editor.set_text("pr");
    editor.move_end(false);
    assert!(editor.accept_completion(&popup, 0).is_err());
    assert_eq!(editor.text(), "pr");
}
#[test]
fn completion_acceptance_preserves_go_selection_prefix_behavior() {
    let mut editor = marked("prefix pr|");
    let popup = editor.complete(&engine());
    editor.set_caret(EditorPosition::default(), false);
    editor.move_end(true);
    editor.accept_completion(&popup, 0).unwrap();
    assert_eq!(editor.text(), "prefix print");
    assert!(editor.selection().is_empty());
}
#[cfg(target_os = "linux")]
mod runtime {
    use super::*;
    use oblikovati_api::{Error, client::Caller};
    use oblikovati_script::{IsolatedLuaEngine, ScriptLimits, ScriptProgram};
    use std::path::Path;
    struct NoCompletionHost;
    impl Caller for NoCompletionHost {
        fn call(&self, method: &str, _: &[u8]) -> Result<Vec<u8>, Error> {
            Err(Error(format!(
                "unexpected completion host call {method}: expected no calls"
            )))
        }
    }
    #[test]
    fn completion_builtins_exactly_match_isolated_lua51_worker_globals() {
        let items = engine().suggest("", 0).0;
        let names: Vec<String> = items
            .iter()
            .filter(|item| item.kind == CompletionKind::Builtin)
            .map(|item| item.text.clone())
            .collect();
        let output = worker_globals();
        assert_eq!(String::from_utf8(output).unwrap().trim(), names.join(","));
        assert!(!items.iter().any(|item| item.text == "goto"));
        assert_eq!(
            items
                .iter()
                .filter(|item| item.kind == CompletionKind::Keyword)
                .count(),
            21
        );
    }
    fn worker_globals() -> Vec<u8> {
        let worker = IsolatedLuaEngine::new(
            Path::new(env!("CARGO_BIN_EXE_oblikovati-script-worker")),
            Vec::new(),
        );
        let program = ScriptProgram { source: "local names={} for name in pairs(_G) do names[#names+1]=name end table.sort(names) print(table.concat(names, ','))".into(),
            methods: Vec::new(), limits: ScriptLimits::default(), parse_only: false };
        let mut output = Vec::new();
        worker
            .start(program)
            .unwrap()
            .wait(&NoCompletionHost, &mut output)
            .unwrap();
        output
    }
}
