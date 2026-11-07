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
    let (minimum, maximum) = sheet_rectangle(AREA, [420., 297.]);
    let (width, height) = (maximum[0] - minimum[0], maximum[1] - minimum[1]);
    assert!((width / height - SHEET_RATIO).abs() < 1e-4);
    assert!(minimum[0] >= AREA.left && maximum[0] <= AREA.right);
    assert!(minimum[1] >= AREA.top && maximum[1] <= AREA.bottom);
}
#[test]
fn a_tiny_area_never_gives_a_negative_sheet() {
    let (minimum, maximum) = sheet_rectangle(
        WorkspaceArea {
            left: 0.,
            top: 0.,
            right: 4.,
            bottom: 4.,
        },
        [420., 297.],
    );
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

#[test]
fn authored_sheet_size_and_border_use_sheet_coordinates() {
    let (minimum, maximum) = sheet_rectangle(AREA, [100., 200.]);
    let width = maximum[0] - minimum[0];
    let height = maximum[1] - minimum[1];
    assert!((width / height - 0.5).abs() < 1e-4);
    let (inner_min, inner_max) =
        sheet_inner_bounds(minimum, maximum, [100., 200.], Some([20., 10., 10., 10.]));
    assert!((inner_min[0] - minimum[0] - width * 0.2).abs() < 1e-3);
    assert!((maximum[0] - inner_max[0] - width * 0.1).abs() < 1e-3);
    assert!((inner_min[1] - minimum[1] - height * 0.05).abs() < 1e-3);
    assert_eq!(
        sheet_inner_bounds(minimum, maximum, [100., 200.], None),
        (minimum, maximum)
    );
}

#[test]
fn title_block_uses_the_authored_corner() {
    use oblikovati_api::types::*;
    let min = [10., 20.];
    let max = [600., 400.];
    assert_eq!(title_bounds(min, max, TOP_LEFT_TITLE_BLOCK).0, min);
    assert_eq!(title_bounds(min, max, BOTTOM_RIGHT_TITLE_BLOCK).1, max);
    let top_right = title_bounds(min, max, TOP_RIGHT_TITLE_BLOCK);
    assert_eq!((top_right.1[0], top_right.0[1]), (max[0], min[1]));
    let bottom_left = title_bounds(min, max, BOTTOM_LEFT_TITLE_BLOCK);
    assert_eq!((bottom_left.0[0], bottom_left.1[1]), (min[0], max[1]));
}
