// SPDX-License-Identifier: GPL-2.0-only
//! Model state rows read from and activate the document's canonical members.
use super::*;
use crate::HeadWork;
use cucumber::{given, then, when};
use oblikovati_api::{client::Client, wire::CreateModelStateArgs};

#[given("a new empty part is active in the MCAD Software")]
fn new_empty_part(world: &mut ApplicationWorld) {
    if world.state().document.is_none() {
        world.add_part();
    }
    if let Some(document) = world.previous_document {
        assert_eq!(world.active_id(), document, "expected the named Part");
    }
}

#[given("the model browser shows the expanded Model State branch with Primary active")]
fn expanded_primary(world: &mut ApplicationWorld) {
    world.click_text("Model State");
    world.expect_text("Primary (Active)");
    assert!(world.checked("Primary (Active)"));
}

#[when("I open the Model State branch context menu")]
#[when("the user opens the Model State branch context menu")]
fn open_state_menu(world: &mut ApplicationWorld) {
    right_click_row(world, "Model State");
}

#[then("it offers New Model State, Expand All Children and Collapse All Children")]
#[then("it offers New Model State, Expand All Children and collapses All Children")]
fn state_menu_options(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &[
            "New Model State",
            "Expand All Children",
            "Collapse All Children",
        ],
    );
}

#[cfg(target_os = "linux")]
#[test]
fn native_model_state_menu_shows_creation_and_tree_actions() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let point = super::super::diagnostics::pointer_pixels(world.run("Model State").center());
    super::super::x11_input::right_click(point);
    world.pump(Vec::new());
    state_menu_options(&mut world);
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    world.close();
}

#[when("I open the New Model State submenu")]
#[when("the user opens the New Model State submenu")]
fn open_state_submenu(world: &mut ApplicationWorld) {
    let position = super::super::diagnostics::pointer_pixels(world.run("New Model State").center());
    world.hover(position);
    world.pump(Vec::new());
}

#[then("it offers New Model State and Duplicate Primary")]
fn state_submenu_options(world: &mut ApplicationWorld) {
    expect_rows(world, &["New Model State", "Duplicate Primary"]);
}

#[when("I choose New Model State")]
#[when("the user chooses New Model State")]
fn create_state(world: &mut ApplicationWorld) {
    world.click_text("New Model State");
    world.pump(Vec::new());
}

#[then("a new model-state entry appears and is active")]
#[then("its generated name is selected for inline editing")]
fn generated_name_selected(world: &mut ApplicationWorld) {
    assert!(world.text_is_selected("Model State1"));
    assert_eq!(
        world
            .state()
            .browser
            .unwrap()
            .model_states
            .unwrap()
            .active()
            .unwrap()
            .name,
        "Model State1"
    );
}

#[when("I enter a new state name and press Enter")]
#[when("the user enters a new state name and presses Enter")]
fn rename_state(world: &mut ApplicationWorld) {
    world.type_text("Large");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}

#[then("the new name appears in the model-state row and branch label")]
fn renamed_row_and_branch(world: &mut ApplicationWorld) {
    world.expect_text("Large (Active)");
    assert!((world.run("<Large>").center()[1] - world.run("Model State").center()[1]).abs() < 3.);
}

#[then("the browser root and document tab identify the new state")]
fn large_caption(world: &mut ApplicationWorld) {
    expect_state_caption(world, "Large");
}

#[then("the browser root and document tab identify Primary")]
fn primary_caption(world: &mut ApplicationWorld) {
    expect_state_caption(world, "Primary");
}

fn expect_state_caption(world: &mut ApplicationWorld, name: &str) {
    let expected = format!("<{name}>");
    let root = super::feature_rows::root_caption(world);
    assert!(
        root.text.contains(&expected),
        "root {:?}: expected {expected}",
        root.text
    );
    let id = world.active_id();
    let document = world.document_name(id);
    world.expect_tab(&format!("{document} {expected}"));
}

#[when("I double-click the Primary model-state entry")]
#[when("the user double-clicks the Primary model-state entry")]
fn activate_primary(world: &mut ApplicationWorld) {
    world.double_click(world.run("Primary").center().map(|v| v as i32));
}

#[then("Primary is marked as active")]
fn primary_active(world: &mut ApplicationWorld) {
    world.expect_text("Primary (Active)");
    assert!(world.checked("Primary (Active)"));
}

#[when("I double-click the new model-state entry")]
#[when("the user double-clicks the new model-state entry")]
fn activate_large(world: &mut ApplicationWorld) {
    world.double_click(world.run("Large").center().map(|v| v as i32));
}

#[then("the new state is marked as active")]
fn large_active(world: &mut ApplicationWorld) {
    world.expect_text("Large (Active)");
    assert!(world.checked("Large (Active)"));
}

#[then("the viewport remains empty")]
fn viewport_empty(world: &mut ApplicationWorld) {
    super::super::viewport_background::expect_empty_part(world);
}

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

#[test]
fn model_states_reopen_with_the_active_member_and_stable_names() {
    use oblikovati_api::wire::{CloseDocumentArgs, OpenDocumentArgs};
    let mut world = ApplicationWorld::new();
    let document = world.add_part();
    world.click_text("Model State");
    open_new_state_menu(&mut world);
    world.click_text("New Model State");
    world.type_text("Large");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.expect_text("Large (Active)");
    let before = world.state().browser.unwrap().model_states.unwrap();
    let path = world.save_part_to_file(document, "states.opd");
    let client = Client::new(world.app().owner.document_host());
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: false,
        })
        .unwrap();
    let reopened = client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    assert_ne!(reopened, document);
    world.pump(Vec::new());
    let state = world.state();
    assert_eq!(state.browser.unwrap().model_states.unwrap(), before);
    assert_eq!(
        state
            .document_model_states
            .get(&reopened)
            .map(String::as_str),
        Some("Large")
    );
    world.expect_text("<Large>");
    world.close();
}
