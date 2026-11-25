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

#[given(
    "`fixtures/cad/quick-access-toolbar/PartOne.opd` and `fixtures/cad/docking-visible-solid-current.opd` are open in that order"
)]
#[given(
    "`fixtures/cad/quick-access-toolbar/PartOne.opd` and `fixtures/cad/docking-visible-solid-current.opd` are open"
)]
pub(super) fn two_saved_documents(world: &mut ApplicationWorld) {
    world.restore_layout_without_asset_tools();
    let other = super::super::scenario_root().join("fixtures/cad/quick-access-toolbar/PartOne.opd");
    world.open_fixture(&other);
    let first = world.active_id();
    let second = super::open_docking_part(world);
    let state = world.state();
    assert!(state.documents.iter().map(|row| row.id).eq([first, second]));
    world.other_document = Some(first);
}

#[given("`docking-visible-solid-current.opd` is active")]
#[given("`fixtures/cad/docking-visible-solid-current.opd` is active")]
fn docking_part_is_active(world: &mut ApplicationWorld) {
    assert_eq!(Some(world.active_id()), world.previous_document);
}

#[given("a dockable panel has been moved, resized, or tabbed in the workspace")]
fn arranged_panel(world: &mut ApplicationWorld) {
    panel_docked(world);
    let node = diagnostics::window_dock_id(PANEL).unwrap();
    assert_ne!(node, 0);
    world.docking_node = Some(node);
}

#[when("the user switches to the other document")]
#[when("the user switches to PartOne")]
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

#[then("the left Model Browser and right tabbed browsers remain in place")]
#[then("the left Model Browser and right tabbed browsers remain available")]
fn tabbed_arrangement_remains(world: &mut ApplicationWorld) {
    assert_eq!(diagnostics::window_dock_id(PANEL), world.docking_node);
    super::restart_scenario::restored_group(world);
}

#[then("the active document changes in the viewport")]
fn other_document_active(world: &mut ApplicationWorld) {
    assert_eq!(world.state().document, world.other_document);
}

#[when("the user returns to the first document")]
#[when("the user returns to `docking-visible-solid-current.opd`")]
fn return_to_first(world: &mut ApplicationWorld) {
    select_document(world, world.previous_document.unwrap());
    assert_eq!(world.state().document, world.previous_document);
}

fn select_document(world: &mut ApplicationWorld, document: u64) {
    let name = world.document_name(document);
    reveal_tab(world, &name);
    let point = diagnostics::pointer_pixels(world.expect_tab(&name).center());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    assert_eq!(world.state().document, Some(document));
}

fn reveal_tab(world: &mut ApplicationWorld, name: &str) {
    let attempts = world.state().documents.len() + 1;
    for arrow in [">", "<"] {
        for _ in 0..attempts {
            if world.tab_run(name).is_some() {
                return;
            }
            let point = world.exact_run_center(arrow);
            super::super::x11_input::click(point);
            world.pump(Vec::new());
        }
    }
    assert!(
        world.tab_run(name).is_some(),
        "expected document tab {name:?} after scrolling"
    );
}
