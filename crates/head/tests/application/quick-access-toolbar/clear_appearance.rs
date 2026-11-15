// SPDX-License-Identifier: GPL-2.0-only
//! Clear an exact selected body override through the native quick access command.
use super::super::workspace_layout::browser_feature::{
    click_branch_arrow, open_visible_extruded_part,
};
use super::*;
use crate::HeadWork;
use oblikovati_api::{
    client::Client,
    wire::{AssignAppearanceArgs, AssignMaterialArgs},
};

fn body_with_override(world: &mut ApplicationWorld) -> String {
    open_visible_extruded_part(world);
    assign_part_assets(world);
    let body = world.state().browser.unwrap().bodies[0].key.clone();
    Client::new(world.app().owner.document_host())
        .appearances()
        .assign(&AssignAppearanceArgs {
            scope: "body".into(),
            key: body.clone(),
            appearance_id: "abs-black".into(),
        })
        .unwrap();
    world.pump(Vec::new());
    click_branch_arrow(world, "Solid Bodies");
    world.click_text("Solid1");
    world.pump(Vec::new());
    body
}
fn selected_appearance(world: &mut ApplicationWorld, key: &str) -> String {
    world
        .state()
        .browser
        .unwrap()
        .bodies
        .into_iter()
        .find(|body| body.key == key)
        .unwrap()
        .appearance
}
fn enable_clear(world: &mut ApplicationWorld) {
    open_customization(world);
    world.click_text("Clear");
    world.pump(Vec::new());
    assert!(on_row(world, "Clear"));
}
#[test]
fn clear_body_override_requires_ok_and_supports_undo_redo() {
    let mut world = ApplicationWorld::new();
    let key = body_with_override(&mut world);
    enable_clear(&mut world);
    assert_eq!(selected_appearance(&mut world, &key), "abs-black");
    let unchanged = part_assets(&mut world);
    verify_cancel(&mut world, &key);
    world.click(row_center(&world, "Clear"));
    world.click_text("OK");
    assert_eq!(selected_appearance(&mut world, &key), "");
    assert_eq!(part_assets(&mut world), unchanged);
    assert_eq!(
        world.state().selected_body_key.as_deref(),
        Some(key.as_str())
    );
    verify_clear_history(&mut world, &key);
    world.close();
}

fn verify_clear_history(world: &mut ApplicationWorld, key: &str) {
    world.click(row_center(world, "Undo"));
    assert_eq!(selected_appearance(world, key), "abs-black");
    world.click(row_center(world, "Redo"));
    assert_eq!(selected_appearance(world, key), "");
}

fn part_assets(world: &mut ApplicationWorld) -> (Option<String>, Option<String>) {
    let browser = world.state().browser.unwrap();
    (browser.part_material, browser.part_appearance)
}

#[test]
fn clear_cannot_apply_after_the_selection_changes() {
    let mut world = ApplicationWorld::new();
    let key = body_with_override(&mut world);
    enable_clear(&mut world);
    world.click(row_center(&world, "Clear"));
    world.click_text("Extrude1");
    world.pump(Vec::new());
    assert!(!world.run_is_enabled("OK"));
    let document = world.active_id();
    assert!(
        world
            .app()
            .owner
            .clear_selected_body_appearance_from_gui(document, &key)
            .is_err()
    );
    world.click_text("Cancel");
    assert_eq!(selected_appearance(&mut world, &key), "abs-black");
    world.close();
}

fn verify_cancel(world: &mut ApplicationWorld, key: &str) {
    let history = world.state().undo_labels;
    world.click(row_center(world, "Clear"));
    assert!(window_geometry("Clear Appearance").is_some());
    world.click_text("Cancel");
    assert_eq!(world.state().undo_labels, history);
    assert_eq!(selected_appearance(world, key), "abs-black");
}

fn assign_part_assets(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    let material = browser.materials.first().unwrap().id.clone();
    let appearance = browser
        .appearances
        .iter()
        .find(|item| item.id != "abs-black")
        .unwrap()
        .id
        .clone();
    assign_part_material(world, &material);
    assign_part_appearance(world, &appearance);
    world.pump(Vec::new());
}
fn assign_part_material(world: &mut ApplicationWorld, id: &str) {
    Client::new(world.app().owner.document_host())
        .materials()
        .assign(&AssignMaterialArgs {
            body_key: String::new(),
            material_id: id.into(),
        })
        .unwrap();
}
fn assign_part_appearance(world: &mut ApplicationWorld, id: &str) {
    Client::new(world.app().owner.document_host())
        .appearances()
        .assign(&AssignAppearanceArgs {
            scope: "part".into(),
            key: String::new(),
            appearance_id: id.into(),
        })
        .unwrap();
}
