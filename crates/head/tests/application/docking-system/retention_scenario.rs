// SPDX-License-Identifier: GPL-2.0-only
//! Verify workspace docking while the user switches document tabs.
use super::{ApplicationWorld, PANEL, diagnostics, panel_docked, viewport};
use cucumber::{given, then, when};

#[given("two documents are open in the MCAD Software")]
fn two_open_documents(world: &mut ApplicationWorld) {
    let first = super::open_docking_fixture(world);
    let second = world.add_part();
    assert_ne!(first, second);
    world.previous_document = Some(first);
    world.other_document = Some(second);
    select_document(world, first);
}

#[given("a dockable panel has been moved, resized, or tabbed in the workspace")]
fn arranged_panel(world: &mut ApplicationWorld) {
    panel_docked(world);
    let node = diagnostics::window_dock_id(PANEL).unwrap();
    assert_ne!(node, 0);
    world.docking_node = Some(node);
}

#[when("the user switches to the other document")]
fn switch_to_other(world: &mut ApplicationWorld) {
    select_document(world, world.other_document.unwrap());
}

#[then("the docked panel arrangement remains in place")]
#[then("the panel arrangement remains available")]
fn arrangement_remains(world: &mut ApplicationWorld) {
    let node = world.docking_node.unwrap();
    assert_eq!(diagnostics::window_dock_id(PANEL), Some(node));
    let (area, width) = world.docking_resize.unwrap();
    assert_eq!(viewport(world), area);
    assert_eq!(diagnostics::window_geometry(PANEL).unwrap().1[0], width);
    world.run_in_window(PANEL, "Assigntopart");
}

#[then("the active document changes in the viewport")]
fn other_document_active(world: &mut ApplicationWorld) {
    assert_eq!(world.state().document, world.other_document);
}

#[when("the user returns to the first document")]
fn return_to_first(world: &mut ApplicationWorld) {
    select_document(world, world.previous_document.unwrap());
    assert_eq!(world.state().document, world.previous_document);
}

fn select_document(world: &mut ApplicationWorld, document: u64) {
    let name = world.document_name(document);
    let point = diagnostics::pointer_pixels(world.expect_tab(&name).center());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    assert_eq!(world.state().document, Some(document));
}
