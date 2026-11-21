// SPDX-License-Identifier: GPL-2.0-only
//! Exercise linked text through the typed host, history, and the package reopen path.
use super::*;
use oblikovati_api::types::{
    SKETCH_TEXT_FIELD_FILE_NAME, SKETCH_TEXT_FIELD_MODEL_PARAMETER, SKETCH_TEXT_FIELD_PLACEHOLDER,
    SketchTextField,
};
use oblikovati_api::wire::ParameterRenameArgs;
use oblikovati_model::sketch::resolve_document_sketch_text;

#[test]
fn typed_linked_text_tracks_value_history_rename_and_reopen() {
    let (mut files, document, mut source) = entities::configured_scaled(0.125);
    let entity = create_linked_text(&mut files, &mut source);
    let baseline = read_text(&files, entity);
    assert_eq!(
        linked_display(&files, document, entity),
        "10.000\nsketch.obk"
    );
    change_linked_parameter(&mut files, &mut source);
    assert_eq!(
        linked_display(&files, document, entity),
        "12.000\nsketch.obk"
    );
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(read_text(&files, entity), baseline);
    assert_eq!(
        linked_display(&files, document, entity),
        "10.000\nsketch.obk"
    );
    files.commands(document).unwrap().redo().unwrap();
    rename_linked_parameter(&mut files, &mut source);
    let renamed = edit_height(&mut files, &mut source, entity);
    assert_eq!(
        renamed.style.linked_fields.as_ref().unwrap()[0].parameter,
        "diameter"
    );
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(read_text(&files, entity), renamed);
    assert_eq!(
        linked_display(&files, document, entity),
        "12.000\nsketch.obk"
    );
}

#[test]
fn typed_linked_text_clears_sources_when_edit_replaces_the_literal_content() {
    let (mut files, document, mut source) = entities::configured_scaled(0.125);
    let entity = create_linked_text(&mut files, &mut source);
    let edit = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .edit_text(&EditTextArgs {
            entity_id: entity,
            text: Some(Box::new("plain".into())),
            ..Default::default()
        })
        .unwrap();
    assert!(edit.style.linked_fields.is_none());
    assert_eq!(linked_display(&files, document, entity), "plain");
}

fn create_linked_text(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) -> u64 {
    let client = Client::new(FileHost::new(files, source, "test"));
    let mut dimension = numeric("d0", "10 mm");
    dimension.kind = "model".into();
    client.parameters().add(&dimension).unwrap();
    client.sketch().create(&Default::default()).unwrap();
    client
        .sketch()
        .add_text(&AddTextArgs {
            anchor: Some(vec![1., 2.]),
            text: format!("{SKETCH_TEXT_FIELD_PLACEHOLDER}\n{SKETCH_TEXT_FIELD_PLACEHOLDER}"),
            height: "3 mm".into(),
            linked_fields: Some(vec![
                SketchTextField {
                    at: 0,
                    kind: SKETCH_TEXT_FIELD_MODEL_PARAMETER,
                    parameter: "d0".into(),
                    ..Default::default()
                },
                SketchTextField {
                    at: 2,
                    kind: SKETCH_TEXT_FIELD_FILE_NAME,
                    ..Default::default()
                },
            ]),
            ..Default::default()
        })
        .unwrap()
        .entity_id
}

fn change_linked_parameter(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    let client = Client::new(FileHost::new(files, source, "test"));
    client.parameters().set(&numeric("d0", "12 mm")).unwrap();
}

fn rename_linked_parameter(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    let client = Client::new(FileHost::new(files, source, "test"));
    client
        .parameters()
        .rename(&ParameterRenameArgs {
            name: "d0".into(),
            new_name: "diameter".into(),
        })
        .unwrap();
}

fn edit_height(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    entity: u64,
) -> SketchTextResult {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .edit_text(&EditTextArgs {
            entity_id: entity,
            height: "4 mm".into(),
            ..Default::default()
        })
        .unwrap()
}

fn linked_display(files: &DiscoveryWorkspace, document: u64, entity: u64) -> String {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let text = snapshot.sketches.sketches()[0]
        .geometry()
        .text(SketchEntityId(entity))
        .unwrap();
    let file_name = files
        .path(document)
        .unwrap()
        .file_name()
        .unwrap()
        .to_str()
        .unwrap();
    resolve_document_sketch_text(
        &text.style,
        &text.parameter_bindings,
        &snapshot.parameters,
        file_name,
    )
    .unwrap()
    .into_owned()
    .content
}
