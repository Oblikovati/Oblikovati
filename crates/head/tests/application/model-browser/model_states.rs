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
    let state = world.app().owner.gui_command_state().unwrap();
    let states = state.browser.unwrap().model_states.unwrap();
    assert_eq!(states.active().unwrap().id, member.id);
    world.undo();
    world.expect_text("Primary (Active)");
    world.redo();
    world.expect_text("Variant##1 (Active)");
    world.close();
}
