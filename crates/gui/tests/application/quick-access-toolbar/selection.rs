// SPDX-License-Identifier: GPL-2.0-only
use super::plane_selection::{area, inside, label_position};
use super::*;

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
