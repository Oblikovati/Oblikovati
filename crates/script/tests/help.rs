// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::wire::{METHOD_DOCUMENTS_CREATE, METHOD_DOCUMENTS_LIST};
use oblikovati_script::{
    editor::{EditorPosition, ScriptEditor},
    help::{MethodHelp, MethodHelpCatalog, MethodHelpSnapshot, MethodParameter},
};
fn fixture() -> MethodHelpSnapshot {
    MethodHelpSnapshot {
        api_version: "test".into(),
        api_revision: "test".into(),
        methods: vec![
            create_help(),
            MethodHelp {
                wire: METHOD_DOCUMENTS_LIST.into(),
                summary: "List documents.".into(),
                parameters: Vec::new(),
            },
        ],
    }
}
fn create_help() -> MethodHelp {
    MethodHelp {
        wire: METHOD_DOCUMENTS_CREATE.into(),
        summary: "Create document.".into(),
        parameters: vec![MethodParameter {
            name: "type".into(),
            type_name: "string".into(),
        }],
    }
}

fn marked(source: &str) -> ScriptEditor {
    let offset = source.find('|').unwrap();
    let mut editor = ScriptEditor::new(&source.replacen('|', "", 1));
    editor.set_caret(
        EditorPosition {
            line: 0,
            column: source[..offset].chars().count(),
        },
        false,
    );
    editor
}
#[test]
fn lookup_and_table_signatures_preserve_order_and_empty_arguments() {
    let catalog = MethodHelpCatalog::new(fixture()).unwrap();
    assert_eq!(
        catalog.lookup(METHOD_DOCUMENTS_CREATE).unwrap().signature(),
        "documents.create{ type }"
    );
    assert_eq!(
        catalog.lookup(METHOD_DOCUMENTS_LIST).unwrap().signature(),
        "documents.list{}"
    );
    assert!(catalog.lookup("documents.missing").is_none());
    verify_multi_argument_signature();
}
#[test]
fn duplicate_wire_entries_are_refused_and_empty_injected_catalog_is_valid() {
    let mut snapshot = fixture();
    snapshot.methods.push(snapshot.methods[0].clone());
    let error = MethodHelpCatalog::new(snapshot).err().unwrap();
    assert!(error.contains(METHOD_DOCUMENTS_CREATE) && error.contains("unique"));
    let mut empty = fixture();
    empty.methods.clear();
    assert!(
        MethodHelpCatalog::new(empty)
            .unwrap()
            .lookup(METHOD_DOCUMENTS_CREATE)
            .is_none()
    );
}
#[test]
fn published_catalog_is_filtered_to_host_methods_and_has_current_contract_fields() {
    let catalog = MethodHelpCatalog::published(&[METHOD_DOCUMENTS_CREATE.into()]).unwrap();
    let help = catalog.lookup(METHOD_DOCUMENTS_CREATE).unwrap();
    assert!(!help.summary.is_empty());
    assert_eq!(help.signature(), "documents.create{ type, name, subType }");
    assert!(catalog.lookup(METHOD_DOCUMENTS_LIST).is_none());
    assert!(
        MethodHelpCatalog::published(&[])
            .unwrap()
            .lookup(METHOD_DOCUMENTS_CREATE)
            .is_none()
    );
}
#[test]
fn signature_help_uses_only_known_host_chains_and_nearest_call() {
    let catalog = MethodHelpCatalog::new(fixture()).unwrap();
    assert_signature(
        &catalog,
        "oblikovati.documents.create{|type='part'}",
        METHOD_DOCUMENTS_CREATE,
    );
    for source in [
        "documents.create{|x}",
        "oblikovati.unknown{|x}",
        "print(|x)",
        "oblikovati.documents.create(print(|x))",
        "oblikovati.documents.create{} |",
    ] {
        assert!(
            catalog.signature_help(&marked(source)).is_none(),
            "{source}"
        );
    }
}
#[test]
fn hover_help_requires_full_method_and_rejects_invalid_positions() {
    let catalog = MethodHelpCatalog::new(fixture()).unwrap();
    let editor = ScriptEditor::new("λ = oblikovati.documents.create{}");
    let hit = EditorPosition {
        line: 0,
        column: 27,
    };
    assert_hover(&catalog, &editor, hit, METHOD_DOCUMENTS_CREATE);
    for hit in [
        EditorPosition {
            line: 0,
            column: 18,
        },
        EditorPosition { line: 0, column: 0 },
        EditorPosition { line: 5, column: 0 },
    ] {
        assert!(catalog.hover_help(&editor, hit).is_none());
    }
}
#[test]
fn complete_embedded_catalog_round_trips_with_unique_named_fields() {
    let snapshot: MethodHelpSnapshot =
        serde_json::from_str(include_str!("../resources/method-help.json")).unwrap();
    assert_eq!(snapshot.api_version, "0.158.0");
    assert_eq!(
        snapshot.api_revision,
        "32ff38398ab6d6a7df0ca838654d5b8bc8d8b9cd"
    );
    assert_eq!(snapshot.methods.len(), 628);
    verify_snapshot_names(&snapshot);
    MethodHelpCatalog::new(snapshot).unwrap();
}

fn verify_multi_argument_signature() {
    let help = MethodHelp {
        wire: "test.method".into(),
        summary: String::new(),
        parameters: vec![
            MethodParameter {
                name: "first".into(),
                type_name: "string".into(),
            },
            MethodParameter {
                name: "last".into(),
                type_name: "bool".into(),
            },
        ],
    };
    assert_eq!(help.signature(), "test.method{ first, last }");
}

fn assert_signature(catalog: &MethodHelpCatalog, source: &str, wire: &str) {
    assert_eq!(catalog.signature_help(&marked(source)).unwrap().wire, wire);
}

fn assert_hover(
    catalog: &MethodHelpCatalog,
    editor: &ScriptEditor,
    hit: EditorPosition,
    wire: &str,
) {
    assert_eq!(catalog.hover_help(editor, hit).unwrap().wire, wire);
}

fn verify_snapshot_names(snapshot: &MethodHelpSnapshot) {
    assert!(
        snapshot
            .methods
            .windows(2)
            .all(|pair| pair[0].wire < pair[1].wire)
    );
    for help in &snapshot.methods {
        assert!(!help.summary.is_empty(), "{}", help.wire);
        let names: std::collections::BTreeSet<&str> = help
            .parameters
            .iter()
            .map(|parameter| parameter.name.as_str())
            .collect();
        assert_eq!(names.len(), help.parameters.len(), "{}", help.wire);
    }
}
