// SPDX-License-Identifier: GPL-2.0-only
//! Use the copied view prepared on the published saved Part fixture.
use super::part_views::{copy_primary_view, expanded_primary_view};
use super::*;
use cucumber::given;
use oblikovati_api::contract::PartDesignView as _;
use oblikovati_viewport::ViewportPhysicalKey;

pub(super) const CAPTURE_VIEW: &str = "Capture View";
const BROWSER_VIEW: &str = "BrowserView";

#[given("the user copies the Primary view representation as BrowserView and activates BrowserView")]
#[given("the user copies Primary as BrowserView and activates the copied view")]
fn copy_browser_view(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected the saved Part");
    world.expect_active(part);
    expanded_primary_view(world);
    copy_primary_view(world);
    rename_view(world, BROWSER_VIEW);
    activate_view(world, BROWSER_VIEW);
    expect_view_active(world, BROWSER_VIEW);
}

#[given("the user copies Primary as BrowserView, activates it and enables its Lock option")]
fn lock_browser_view(world: &mut ApplicationWorld) {
    copy_browser_view(world);
    super::view_origin::choose_view_lock(world);
    super::view_visibility::copied_view_is_locked(world);
}

pub(super) fn rename_view(world: &mut ApplicationWorld, name: &str) {
    world.type_text(name);
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}

pub(super) fn activate_view(world: &mut ApplicationWorld, name: &str) {
    world.double_click(diagnostics::pointer_pixels(world.run(name).center()));
    world.pump(Vec::new());
}

pub(super) fn expect_view_active(world: &mut ApplicationWorld, name: &str) {
    let views = world.state().browser.unwrap().part_design_views.unwrap();
    assert_eq!(views.active().unwrap().name(), name);
    assert!(world.checked(name));
}

pub(super) fn copy_name(world: &mut ApplicationWorld) -> &'static str {
    let state = world.state();
    let has_named_copy = state
        .browser
        .and_then(|browser| browser.part_design_views)
        .is_some_and(|views| views.list().iter().any(|view| view.name() == BROWSER_VIEW));
    if has_named_copy {
        let part = world.previous_document.expect("expected the saved Part");
        assert_eq!(state.document, Some(part));
        BROWSER_VIEW
    } else {
        CAPTURE_VIEW
    }
}

pub(super) fn expect_prepared_copy(world: &mut ApplicationWorld) -> bool {
    if copy_name(world) != BROWSER_VIEW {
        return false;
    }
    expect_view_active(world, BROWSER_VIEW);
    true
}

pub(super) fn right_click_copy(world: &mut ApplicationWorld) {
    let name = copy_name(world);
    right_click_row(world, name);
}

pub(super) fn expect_copy_active(world: &mut ApplicationWorld) {
    let name = copy_name(world);
    expect_view_active(world, name);
}
