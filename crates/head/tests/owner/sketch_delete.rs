// SPDX-License-Identifier: GPL-2.0-only
use super::geometric_tools::{geometric_owner, geometry_for_relation};
use super::slot_input::{PlanarInputWindow, deliver};
use super::*;
use crate::placement_test_inputs::*;
use oblikovati_gui::{GuiSketchConstraint, GuiSketchTool};

#[test]
fn native_delete_uses_the_common_cascade_and_one_history_step() {
    let (mut owner, document, sketch) =
        geometric_owner(&[[0., 0.], [100., 20.], [140., 120.]], true);
    owner
        .constrain_sketch_from_gui(GuiSketchConstraint::Horizontal)
        .unwrap();
    deliver(&mut owner, &[button([50, 10], true)]);
    let before = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    deliver(&mut owner, &[button([50, 10], true)]);
    assert!(owner.gui_command_state().unwrap().sketch_delete_available);
    assert!(owner.delete_sketch_selection_from_gui().unwrap());
    let geometry = geometry_for_relation(&owner, document, sketch);
    assert_eq!(geometry.lines().len(), 1);
    assert!(geometry.constraints().is_empty());
    assert!(!owner.gui_command_state().unwrap().sketch_delete_available);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .sketches
            .same_recipe(&before)
    );
    assert!(!owner.gui_command_state().unwrap().sketch_delete_available);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(
        geometry_for_relation(&owner, document, sketch)
            .lines()
            .len(),
        1
    );
}
#[test]
fn native_delete_is_inert_without_an_editor_or_during_a_tool() {
    let (mut empty, _) = owner();
    assert!(!empty.delete_sketch_selection_from_gui().unwrap());
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 20.]], true);
    assert!(!owner.delete_sketch_selection_from_gui().unwrap());
    owner
        .constrain_sketch_from_gui(GuiSketchConstraint::Coincident)
        .unwrap();
    deliver(&mut owner, &[button([0, 0], true)]);
    let history = owner.gui_command_state().unwrap().undo_label;
    assert!(!owner.delete_sketch_selection_from_gui().unwrap());
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        1
    );
    assert_eq!(owner.gui_command_state().unwrap().undo_label, history);
}
#[test]
fn native_delete_refuses_caller_transactions_without_losing_selection() {
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 20.]], true);
    deliver(&mut owner, &[button([50, 10], true)]);
    Client::new(owner.document_host())
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Caller".into(),
        })
        .unwrap();
    assert!(
        owner
            .delete_sketch_selection_from_gui()
            .unwrap_err()
            .contains("no open transaction")
    );
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        1
    );
    Client::new(owner.document_host())
        .transactions()
        .abort()
        .unwrap();
}
#[test]
fn native_delete_observer_veto_retains_geometry_and_picks() {
    let (mut owner, _) = super::tool_veto::veto_owner("Delete Sketch Entities");
    owner.create_part_document().unwrap();
    owner.edit_sketch_from_gui(true).unwrap();
    owner
        .place_sketch_tool_from_gui(GuiSketchTool::Point)
        .unwrap();
    let window = PlanarInputWindow::default();
    owner
        .process_window_inputs(&[button([100, 100], true)], &window)
        .unwrap();
    owner
        .process_window_inputs(&[button([100, 100], true)], &window)
        .unwrap();
    let document = owner.gui_command_state().unwrap().document.unwrap();
    let before = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    assert!(owner.delete_sketch_selection_from_gui().is_err());
    assert!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .sketches
            .same_recipe(&before)
    );
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        1
    );
}
