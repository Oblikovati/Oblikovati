// SPDX-License-Identifier: GPL-2.0-only
//! Prepare the published model-state members on the saved Part.
use super::model_states::{create_state, open_state_menu, open_state_submenu};
use super::*;
use cucumber::given;
use oblikovati_viewport::ViewportPhysicalKey;

pub(super) const CAPTURE_STATE: &str = "Capture State";
pub(super) const COPY_STATE: &str = "Copy State";

#[given("the user creates model states Capture State and Copy State")]
#[given("the user creates model states Capture State and Copy State and activates Copy State")]
fn create_named_states(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected the saved Part");
    world.expect_active(part);
    expand_state_branch(world);
    for name in [CAPTURE_STATE, COPY_STATE] {
        create_named_state_in_browser(world, name);
    }
    assert!(has_named_states(world));
}

pub(super) fn create_named_state_in_browser(world: &mut ApplicationWorld, name: &str) {
    open_state_menu(world);
    open_state_submenu(world);
    create_state(world);
    let browser = world.state().browser.unwrap();
    let states = browser.model_states.unwrap();
    if states.active().unwrap().name != name {
        world.type_text(name);
    }
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    expect_state_active(world, name);
}

pub(super) fn expand_state_branch(world: &mut ApplicationWorld) {
    let views = world.state().browser.unwrap().model_states.unwrap();
    let name = &views.active().unwrap().name;
    let row = format!("{name} (Active)");
    if !world.shows(&row) {
        world.click_text("Model State");
    }
    expect_state_active(world, name);
}

fn expect_state_active(world: &mut ApplicationWorld, name: &str) {
    let states = world.state().browser.unwrap().model_states.unwrap();
    assert_eq!(states.active().unwrap().name, name);
    assert!(world.checked(&format!("{name} (Active)")));
}

pub(super) fn has_named_states(world: &mut ApplicationWorld) -> bool {
    let Some(part) = world.previous_document else {
        return false;
    };
    let state = world.state();
    let Some(states) = state.browser.and_then(|browser| browser.model_states) else {
        return false;
    };
    if !states
        .list()
        .iter()
        .any(|member| member.name == CAPTURE_STATE)
    {
        return false;
    }
    assert_eq!(state.document, Some(part));
    assert!(states.list().iter().any(|member| member.name == COPY_STATE));
    true
}

pub(super) fn expect_empty_named_states(world: &mut ApplicationWorld) -> bool {
    if !has_named_states(world) {
        return false;
    }
    let browser = world.state().browser.unwrap();
    assert!(browser.bodies.is_empty());
    assert!(browser.features.is_empty());
    assert_eq!(browser.model_states.unwrap().list().len(), 3);
    true
}
