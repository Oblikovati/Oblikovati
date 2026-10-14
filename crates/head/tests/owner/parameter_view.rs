// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn add_view_parameters(owner: &mut TestOwner) {
    let client = Client::new(owner.document_host());
    for (name, expression, value_type) in [
        ("width", "4 cm", "numeric"),
        ("material", "é🙂", "text"),
        ("vented", "false", "boolean"),
    ] {
        client
            .parameters()
            .add(&ParameterSetArgs {
                name: name.into(),
                expression: expression.into(),
                value_type: value_type.into(),
                ..Default::default()
            })
            .unwrap();
    }
}
#[test]
fn parameter_view_uses_canonical_details_and_model_identity_without_worker_admission() {
    let (mut owner, awake) = owner();
    assert_eq!(owner.gui_parameter_snapshot().unwrap(), None);
    let document = create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let generation = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let view = owner.gui_parameter_snapshot().unwrap().unwrap();
    view.validate().unwrap();
    assert_eq!(view.document, document);
    assert_eq!(
        view.rows
            .iter()
            .map(|row| row.detail.units.as_str())
            .collect::<Vec<_>>(),
        ["mm", "text", "boolean"]
    );
    assert_eq!(view.rows[1].detail.parameter_info.value, "é🙂");
    let session = owner.documents().workspace().session(document).unwrap();
    assert_eq!(
        view.rows
            .iter()
            .map(|row| row.parameter)
            .collect::<Vec<_>>(),
        session.snapshot().parameters.ids()
    );
    assert_eq!(session.generation(), generation);
    assert!(awake.try_recv().is_err());
}
#[test]
fn renaming_does_not_change_the_parameter_editor_identity_or_the_prior_snapshot() {
    let (mut owner, awake) = owner();
    create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let before = owner.gui_parameter_snapshot().unwrap().unwrap();
    Client::new(owner.document_host())
        .parameters()
        .rename(&ParameterRenameArgs {
            name: "width".into(),
            new_name: "span".into(),
        })
        .unwrap();
    let renamed = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert_eq!(renamed.rows[0].parameter, before.rows[0].parameter);
    assert_eq!(renamed.rows[0].detail.parameter_info.name, "span");
    assert_eq!(before.rows[0].detail.parameter_info.name, "width");
}
#[test]
fn different_active_documents_keep_separate_parameter_view_namespaces() {
    let (mut owner, awake) = owner();
    create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let first = owner.gui_parameter_snapshot().unwrap().unwrap();
    create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let second = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert_ne!(first.document, second.document);
    assert_eq!(first.rows[0].parameter, second.rows[0].parameter);
    assert_eq!(first.rows[0].detail, second.rows[0].detail);
}
#[path = "parameter_edits.rs"]
mod edits;
