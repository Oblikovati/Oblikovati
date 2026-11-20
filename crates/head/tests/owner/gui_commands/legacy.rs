// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn gui_model_browser_state_projects_the_typed_model_tree() {
    let (mut owner, awake) = owner();
    create(&mut owner, &awake);
    let expected = Client::new(owner.document_host()).model().tree().unwrap();
    let state = owner.gui_command_state().unwrap();
    let actual = state.browser.unwrap();
    assert_eq!(actual.name, expected.document);
    assert_eq!(
        actual.sketches.len(),
        usize::try_from(expected.sketches).unwrap()
    );
    assert_eq!(
        actual.bodies.len(),
        usize::try_from(expected.bodies).unwrap()
    );
    let features = expected.features.unwrap_or_default();
    assert_eq!(actual.features.len(), features.len());
    for (actual, expected) in actual.features.iter().zip(features) {
        assert_eq!(actual.name, expected.name);
    }
    let names: Vec<_> = state
        .parameters
        .iter()
        .map(|parameter| parameter.name.clone())
        .collect();
    assert_eq!(names, expected.parameters.unwrap_or_default());
}

#[test]
fn gui_sketch_preferences_publish_and_apply_on_the_owner() {
    let (mut owner, awake) = owner();
    create(&mut owner, &awake);
    let mut options = owner.gui_command_state().unwrap().sketch_options.unwrap();
    assert!(options.grid_visible && options.heads_up_display);
    options.grid_visible = false;
    options.heads_up_display = false;
    options.grid_spacing_cm = 2.5;
    options.grid_major_every = 8;
    options.snap_to_grid = false;
    assert!(owner.set_sketch_options_from_gui(options.clone()).unwrap());
    assert_eq!(
        owner.gui_command_state().unwrap().sketch_options,
        Some(options)
    );
}

#[test]
fn gui_document_tab_close_uses_the_typed_document_route() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: "/memory/closable.opd".into(),
        })
        .unwrap();
    assert!(
        owner
            .close_document_from_gui(&CloseDocumentArgs {
                id: document,
                force: false
            })
            .unwrap()
    );
    assert!(owner.documents().workspace().documents().is_empty());
}

#[test]
fn gui_history_jump_uses_the_typed_transaction_route() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    dirty_property(&mut owner, document);
    let before = owner.gui_command_state().unwrap();
    assert_eq!(before.histories[0].document, document);
    assert_eq!(before.histories[0].position, 1);
    assert_eq!(before.histories[0].entries.as_ref().unwrap().len(), 1);
    assert!(owner.jump_history_from_gui(document, 0).unwrap());
    let after = owner.gui_command_state().unwrap();
    assert_eq!(after.histories[0].position, 0);
    assert!(after.undo_label.is_none());
    assert!(after.redo_label.is_some());
}

#[test]
fn gui_units_actions_use_the_typed_document_route() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .document_units
            .unwrap()
            .length_unit,
        "mm"
    );
    assert!(
        owner
            .set_document_units_from_gui(
                document,
                &SetDocumentUnitsArgs {
                    length_unit: Some(Box::new("cm".into())),
                    length_display_precision: Some(Box::new(5)),
                    ..Default::default()
                }
            )
            .unwrap()
    );
    let after = owner.gui_command_state().unwrap().document_units.unwrap();
    assert_eq!(after.length_unit, "cm");
    assert_eq!(after.length_display_precision, 5);
}

#[test]
fn gui_constraint_settings_apply_to_the_active_document_only() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let initial = owner
        .gui_command_state()
        .unwrap()
        .document_sketch_settings
        .unwrap();
    assert!(
        !owner
            .set_document_sketch_settings_from_gui(document, &initial, &initial)
            .unwrap()
    );
    let mut changed = initial.clone();
    changed.infer_constraints = false;
    assert!(
        owner
            .set_document_sketch_settings_from_gui(document + 1, &initial, &changed)
            .is_err()
    );
    assert_eq!(
        owner.gui_command_state().unwrap().document_sketch_settings,
        Some(initial.clone())
    );
    assert!(
        owner
            .set_document_sketch_settings_from_gui(document, &initial, &changed)
            .unwrap()
    );
    assert!(
        owner
            .set_document_sketch_settings_from_gui(document, &initial, &initial)
            .is_err()
    );
    assert_eq!(
        owner.gui_command_state().unwrap().document_sketch_settings,
        Some(changed)
    );
}
