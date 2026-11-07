// SPDX-License-Identifier: GPL-2.0-only
//! Model state rows read from and activate the document's canonical members.
use super::*;
use crate::HeadWork;
use oblikovati_api::{client::Client, wire::CreateModelStateArgs};

#[test]
fn model_state_rows_follow_document_and_double_click_activates_member() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let member = Client::new(world.app().owner.document_host())
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Variant##1".into(),
            ..Default::default()
        })
        .unwrap()
        .model_state;
    world.pump(Vec::new());
    world.expect_text("Representations");
    world.click_text("Model State");
    world.expect_text("Primary (Active)");
    let position = super::super::diagnostics::pointer_pixels(world.run("Variant##1").center());
    world.double_click(position);
    world.pump(Vec::new());
    world.expect_text("Variant##1 (Active)");
    assert!(
        (world.run("<Variant##1>").center()[1] - world.run("Model State").center()[1]).abs() < 3.
    );
    let state = world.app().owner.gui_command_state().unwrap();
    let states = state.browser.unwrap().model_states.unwrap();
    assert_eq!(states.active().unwrap().id, member.id);
    world.undo();
    world.expect_text("Primary (Active)");
    world.redo();
    world.expect_text("Variant##1 (Active)");
    open_new_state_menu(&mut world);
    world.expect_text("Duplicate Variant##1");
    world.click_text("Duplicate Variant##1");
    world.pump(Vec::new());
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.expect_text("Variant##1 Copy1");
    world.expect_text("Variant##1 (Active)");
    world.close();
}

fn open_new_state_menu(world: &mut ApplicationWorld) {
    right_click_row(world, "Model State");
    let menu = super::super::diagnostics::pointer_pixels(world.run("New Model State").center());
    world.hover(menu);
    world.pump(Vec::new());
}

#[test]
fn model_state_menu_creates_active_and_copies_inactive_with_undo() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Model State");
    open_new_state_menu(&mut world);
    world.click_text("New Model State");
    world.pump(Vec::new());
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.expect_text("Model State1 (Active)");
    world.undo();
    world.expect_text("Primary (Active)");
    world.expect_no_text("Model State1");
    world.redo();
    world.expect_text("Model State1 (Active)");
    open_new_state_menu(&mut world);
    world.click_text("Duplicate Model State1");
    world.pump(Vec::new());
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.expect_text("Model State1 (Active)");
    world.expect_text("Model State1 Copy1");
    world.close();
}

#[test]
fn created_model_state_name_can_be_edited_or_cancelled() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Model State");
    open_new_state_menu(&mut world);
    world.click_text("New Model State");
    world.pump(Vec::new());
    assert!(world.text_is_selected("Model State1"));
    world.type_text("Primary");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.expect_text("Enter a unique, nonempty name");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    world.pump(Vec::new());
    world.expect_text("Model State1 (Active)");
    open_new_state_menu(&mut world);
    world.click_text("New Model State");
    world.pump(Vec::new());
    assert!(world.text_is_selected("Model State2"));
    world.type_text("Large");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.expect_text("Large (Active)");
    assert!((world.run("<Large>").center()[1] - world.run("Model State").center()[1]).abs() < 3.);
    world.expect_text("Part1 <Large>");
    assert_eq!(
        world
            .frame
            .runs
            .iter()
            .filter(|run| run.text.contains("Part1") && run.text.contains("Large"))
            .count(),
        2,
        "expected the browser root and document tab to show the member"
    );
    assert_eq!(
        world.app().window.native.document_title.as_deref(),
        Some("Part1 <Large>")
    );
    let state = world.app().owner.gui_command_state().unwrap();
    assert_eq!(state.documents[0].name, "Part1");
    assert_eq!(
        state
            .document_model_states
            .get(&state.documents[0].id)
            .unwrap(),
        "Large"
    );
    world.close();
}
