// SPDX-License-Identifier: GPL-2.0-only
use super::plane_selection::{area, inside, label_position};
use super::*;

#[test]
fn appearance_browser_key_tip_requires_an_active_part() {
    let mut toolbar = QuickAccessToolbar::default();
    let mut state = GuiCommandState {
        document: Some(1),
        document_type: Some(oblikovati_api::types::DOCUMENT_PART),
        authoring: crate::GuiAuthoringContext::Part,
        ..Default::default()
    };
    for command in [
        StandardCommand::New,
        StandardCommand::Open,
        StandardCommand::Save,
    ] {
        toolbar.selection(&state).toggle(command);
    }
    toolbar.activate_tip(8, &state, &mut Vec::new());
    assert!(toolbar.take_requests().open_appearance_browser);
    state.authoring = crate::GuiAuthoringContext::Sketch;
    toolbar.activate_tip(8, &state, &mut Vec::new());
    assert!(!toolbar.take_requests().open_appearance_browser);
}

#[test]
fn material_key_tip_opens_only_for_an_idle_part() {
    let mut toolbar = QuickAccessToolbar::default();
    let mut state = GuiCommandState {
        document: Some(1),
        document_type: Some(oblikovati_api::types::DOCUMENT_PART),
        authoring: crate::GuiAuthoringContext::Part,
        ..Default::default()
    };
    toolbar.selection(&state).toggle(StandardCommand::New);
    toolbar.selection(&state).toggle(StandardCommand::Open);
    toolbar.activate_tip(8, &state, &mut Vec::new());
    assert!(toolbar.open_materials);
    toolbar.open_materials = false;
    state.transaction_open = true;
    toolbar.activate_tip(8, &state, &mut Vec::new());
    assert!(!toolbar.open_materials);
}

#[test]
fn the_default_row_leaves_print_out_and_a_toggle_applies_once_each_way() {
    let mut selection = KindSelection::default();
    assert!(selection.shows(StandardCommand::Home));
    assert!(!selection.shows(StandardCommand::Print));
    selection.toggle(StandardCommand::Print);
    assert!(selection.shows(StandardCommand::Print));
    selection.toggle(StandardCommand::Print);
    assert!(!selection.shows(StandardCommand::Print));
}
#[test]
fn a_separator_lands_just_after_its_command_and_stays_when_the_command_goes() {
    let mut selection = KindSelection::default();
    selection.added.push(AddedEntry::Command("Extrude".into()));
    selection.added.push(AddedEntry::Command("Revolve".into()));
    add_separator_after(&mut selection, "Extrude");
    assert_eq!(selection.added[1], AddedEntry::Separator);
    selection.remove_command("Extrude");
    assert_eq!(selection.added[0], AddedEntry::Separator);
    assert!(!selection.has_command("Extrude"));
    assert!(selection.has_command("Revolve"));
}
#[test]
fn each_document_kind_keeps_its_own_choices() {
    let mut toolbar = QuickAccessToolbar::default();
    let part = GuiCommandState {
        document_type: Some(oblikovati_api::types::DOCUMENT_PART),
        ..Default::default()
    };
    let assembly = GuiCommandState {
        document_type: Some(oblikovati_api::types::DOCUMENT_ASSEMBLY),
        ..Default::default()
    };
    toolbar.selection(&part).toggle(StandardCommand::Print);
    assert!(toolbar.selection(&part).shows(StandardCommand::Print));
    assert!(!toolbar.selection(&assembly).shows(StandardCommand::Print));
}
#[test]
fn a_plane_outline_holds_its_centre_and_reports_its_area() {
    let square = [[0., 0.], [4., 0.], [4., 4.], [0., 4.]];
    assert!(inside(&square, [2., 2.]));
    assert!(!inside(&square, [5., 2.]));
    assert_eq!(area(&square), 16.);
}

#[test]
fn plane_label_stays_inside_the_clickable_workspace() {
    use crate::application::document_creation::workspaces::WorkspaceArea;
    let workspace = WorkspaceArea {
        left: 100.,
        top: 200.,
        right: 500.,
        bottom: 400.,
    };
    let above = [[200., 100.], [300., 100.], [300., 150.], [200., 150.]];
    let right = [[490., 300.], [600., 300.], [600., 450.], [490., 450.]];
    assert_eq!(label_position(&above, [60., 16.], workspace), [280., 200.]);
    assert_eq!(label_position(&right, [60., 16.], workspace), [440., 384.]);
}

#[test]
fn measure_is_optional_and_key_tip_uses_the_inspect_action() {
    let mut toolbar = QuickAccessToolbar::default();
    let state = GuiCommandState {
        document: Some(1),
        document_type: Some(oblikovati_api::types::DOCUMENT_PART),
        authoring: crate::GuiAuthoringContext::Part,
        ..Default::default()
    };
    assert!(!toolbar.selection(&state).shows(StandardCommand::Measure));
    toolbar.selection(&state).toggle(StandardCommand::Measure);
    let mut actions = Vec::new();
    toolbar.activate_tip(13, &state, &mut actions);
    assert!(matches!(
        actions.as_slice(),
        [GuiShellAction::Inspect(
            crate::GuiInspectAction::StartMeasure
        )]
    ));
}

#[test]
fn return_is_optional_and_only_acts_while_sketch_editing() {
    let mut toolbar = QuickAccessToolbar::default();
    let mut state = GuiCommandState {
        document: Some(1),
        document_type: Some(oblikovati_api::types::DOCUMENT_PART),
        authoring: crate::GuiAuthoringContext::Part,
        ..Default::default()
    };
    assert!(!toolbar.selection(&state).shows(StandardCommand::Return));
    toolbar.selection(&state).toggle(StandardCommand::Return);
    let mut actions = Vec::new();
    toolbar.activate_tip(13, &state, &mut actions);
    assert!(actions.is_empty());
    state.authoring = crate::GuiAuthoringContext::Sketch;
    toolbar.activate_tip(13, &state, &mut actions);
    assert_eq!(actions, [GuiShellAction::FinishSketch]);
}

#[test]
fn return_cannot_finish_while_a_transaction_is_open() {
    let mut state = GuiCommandState {
        authoring: crate::GuiAuthoringContext::Sketch,
        transaction_open: true,
        ..Default::default()
    };
    assert!(!GuiShellAction::FinishSketch.enabled(&state));
    state.transaction_open = false;
    assert!(GuiShellAction::FinishSketch.enabled(&state));
    state.authoring = crate::GuiAuthoringContext::OtherDocument;
    assert!(!GuiShellAction::FinishSketch.enabled(&state));
}
