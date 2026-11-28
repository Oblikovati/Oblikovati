// SPDX-License-Identifier: GPL-2.0-only
//! Check Link Parameters close transitions in a shared dock.
use super::dockspace_native::{open_toolbar_tool, viewport};
use super::dockspace_restart::select_dock_tab;
use super::orientation_close_native::drag_title;
use super::*;

fn open_picker(world: &mut ApplicationWorld) {
    let point = world.run_in_window("Parameters", "Link...").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    assert!(window_geometry("Link Parameters").is_some());
}

fn dock_picker(world: &mut ApplicationWorld) -> u32 {
    let area = viewport(world);
    open_toolbar_tool(world, "Material Browser");
    drag_title(
        world,
        "Material Browser",
        [area[2] as i32 - 15, ((area[1] + area[3]) / 2) as i32],
    );
    open_toolbar_tool(world, "Parameters");
    open_picker(world);
    let (position, size) = window_geometry("Material Browser").unwrap();
    drag_title(
        world,
        "Link Parameters",
        [
            (position[0] + size[0] * 0.5) as i32,
            (position[1] + 10.) as i32,
        ],
    );
    let node = super::super::diagnostics::window_dock_id("Material Browser").unwrap();
    assert_ne!(node, 0);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Link Parameters"),
        Some(node)
    );
    node
}

#[test]
fn link_cancel_selects_next_tab_and_reopens() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let node = dock_picker(&mut world);
    select_dock_tab(&mut world, "Link Parameters");
    let point = world.run_in_window("Link Parameters", "Cancel").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(window_geometry("Link Parameters").is_none());
    assert_eq!(
        super::super::diagnostics::window_dock_tab_visible("Material Browser"),
        Some(true)
    );
    open_picker(&mut world);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Link Parameters"),
        Some(node)
    );
    world.save_capture("link-restored-to-shared-dock");
    world.close();
}

fn add_link_source(world: &mut ApplicationWorld) {
    let client = oblikovati_api::client::Client::new(world.app().owner.document_host());
    client
        .documents()
        .create(&oblikovati_api::wire::CreateDocumentArgs {
            type_: "part".into(),
            name: "Source part".into(),
            ..Default::default()
        })
        .unwrap();
    client
        .parameters()
        .add(&oblikovati_api::wire::ParameterSetArgs {
            name: "span".into(),
            expression: "40 mm".into(),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
}

#[test]
fn link_acknowledgement_selects_next_tab_and_reopens() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    add_link_source(&mut world);
    super::browser_feature::open_visible_extruded_part(&mut world);
    let node = dock_picker(&mut world);
    select_dock_tab(&mut world, "Link Parameters");
    let point = world
        .run_in_window("Link Parameters", "(chooseadocument)")
        .center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    super::dockspace_restart::native_click_label(&mut world, "Source part");
    let row = world.run_in_window("Link Parameters", "span").clone();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels([
        row.minimum[0] - 10.,
        row.center()[1],
    ]));
    world.pump(Vec::new());
    let point = world.run_in_window("Link Parameters", "Link").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(window_geometry("Link Parameters").is_none());
    let snapshot = world.app().owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(
        snapshot
            .linked_tables
            .iter()
            .any(|table| table.source_document == "Source part")
    );
    assert_eq!(
        super::super::diagnostics::window_dock_tab_visible("Material Browser"),
        Some(true)
    );
    open_picker(&mut world);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Link Parameters"),
        Some(node)
    );
    world.save_capture("link-ack-restored-to-shared-dock");
    world.close();
}

#[test]
fn link_snapshot_loss_selects_next_tab_and_reopens() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    add_link_source(&mut world);
    super::browser_feature::open_visible_extruded_part(&mut world);
    let target = world.active_id();
    let target_name = world.document_name(target);
    let node = dock_picker(&mut world);
    select_dock_tab(&mut world, "Link Parameters");
    let point = world.expect_tab("Source").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    super::external_close_native::assert_closed_with_visible_tab(
        &mut world,
        "Link Parameters",
        "Material Browser",
    );
    assert_eq!(
        super::super::diagnostics::window_dock_tab_visible("Material Browser"),
        Some(true)
    );
    let point = world.expect_tab(&target_name).center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    open_picker(&mut world);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Link Parameters"),
        Some(node)
    );
    world.save_capture("link-snapshot-restored-to-shared-dock");
    world.close();
}
