// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_gui::GuiParameterMutation::{Add, Set};

fn size_request(expression: &str) -> ParameterSetArgs {
    ParameterSetArgs {
        name: "size".into(),
        expression: expression.into(),
        ..Default::default()
    }
}
fn editor_expression(owner: &TestOwner) -> String {
    owner
        .gui_command_state()
        .unwrap()
        .parameters
        .iter()
        .find(|parameter| parameter.name == "size")
        .unwrap()
        .expression
        .clone()
}
#[test]
fn parameter_editor_uses_typed_owner_commands_and_history() {
    let (mut owner, _awake) = owner();
    assert!(
        owner
            .edit_parameter_from_gui(Add, &size_request("10 mm"))
            .is_err()
    );
    owner.create_part_document().unwrap();
    assert!(
        owner
            .edit_parameter_from_gui(Add, &size_request("10 mm"))
            .unwrap()
    );
    assert_eq!(editor_expression(&owner), "10 mm");
    assert!(
        owner
            .edit_parameter_from_gui(Set, &size_request("20 mm"))
            .unwrap()
    );
    assert_eq!(editor_expression(&owner), "20 mm");
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(editor_expression(&owner), "10 mm");
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(editor_expression(&owner), "20 mm");
}
#[test]
fn duplicate_parameter_creation_keeps_the_existing_value_and_history() {
    let (mut owner, _awake) = owner();
    owner.create_part_document().unwrap();
    owner
        .edit_parameter_from_gui(Add, &size_request("10 mm"))
        .unwrap();
    let before = owner.gui_command_state().unwrap();
    assert!(
        owner
            .edit_parameter_from_gui(Add, &size_request("20 mm"))
            .is_err()
    );
    assert_eq!(owner.gui_command_state().unwrap(), before);
}
