// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::application::document_creation::templates::{
    GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits,
};
const AREA: WorkspaceArea = WorkspaceArea {
    left: 300.,
    top: 100.,
    right: 1300.,
    bottom: 900.,
};
#[test]
fn the_sheet_keeps_its_ratio_and_stays_inside_the_area() {
    let (minimum, maximum) = sheet_rectangle(AREA);
    let (width, height) = (maximum[0] - minimum[0], maximum[1] - minimum[1]);
    assert!((width / height - SHEET_RATIO).abs() < 1e-4);
    assert!(minimum[0] >= AREA.left && maximum[0] <= AREA.right);
    assert!(minimum[1] >= AREA.top && maximum[1] <= AREA.bottom);
}
#[test]
fn a_tiny_area_never_gives_a_negative_sheet() {
    let (minimum, maximum) = sheet_rectangle(WorkspaceArea {
        left: 0.,
        top: 0.,
        right: 4.,
        bottom: 4.,
    });
    assert!(maximum[0] > minimum[0] && maximum[1] > minimum[1]);
}
#[test]
fn only_creating_a_presentation_arms_the_insert_dialog() {
    let mut workspaces = DocumentWorkspaces::default();
    let create = |kind| {
        GuiShellAction::CreateDocument(GuiDocumentTemplate {
            kind,
            units: GuiTemplateUnits::Metric,
        })
    };
    workspaces.note_created(&[create(GuiTemplateKind::Assembly), GuiShellAction::Save]);
    assert!(!workspaces.needs_frame());
    workspaces.note_created(&[create(GuiTemplateKind::Presentation)]);
    assert!(workspaces.needs_frame());
}
#[test]
fn the_active_name_is_empty_without_a_row() {
    assert_eq!(active_name(&GuiCommandState::default()), "");
}
