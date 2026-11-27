// SPDX-License-Identifier: GPL-2.0-only
//! Observe picker option changes while selecting the same or a different drawing.
use super::{
    selections::{DEFER, click_option},
    *,
};

const SKIP: &str = "Skip all unresolved files";

#[given("a saved drawing is selected with its Options pane displayed")]
fn drawing_selected(world: &mut ApplicationWorld) {
    if published_drawings::show_staged_drawings(world) {
        let name = published_drawings::second_drawing_name(world);
        select_row(world, name);
        return;
    }
    cancel_dialog(world);
    let folder = models_folder(world);
    saved_document(world, GuiTemplateKind::Drawing, &folder, DRAWING_FILE);
    open_dialog(world);
    look_in(world, &folder);
    select_row(world, DRAWING_FILE);
}
#[given("Load in Defer Mode is unchecked")]
#[given("Load in Defer Mode is unchecked for the first drawing")]
#[then("Load in Defer Mode is unchecked")]
#[then("Load in Defer Mode is unchecked for the second drawing")]
fn defer_off(world: &mut ApplicationWorld) {
    assert!(!world.checked(DEFER), "expected {DEFER} unchecked");
}
#[given("Skip all unresolved files is checked")]
fn skip_on(world: &mut ApplicationWorld) {
    click_option(world, SKIP);
    assert!(world.checked(SKIP), "expected {SKIP} checked");
}
#[when("I select Load in Defer Mode")]
#[when("the user selects Load in Defer Mode")]
#[when("I clear Load in Defer Mode")]
#[when("the user clears Load in Defer Mode")]
fn toggle_defer(world: &mut ApplicationWorld) {
    click_option(world, DEFER);
}
#[then("Skip all unresolved files is unavailable")]
fn skip_off(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled(SKIP), "expected {SKIP} unavailable");
}
#[then("Skip all unresolved files is available and unchecked")]
fn skip_clear(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled(SKIP), "expected {SKIP} available");
    assert!(!world.checked(SKIP), "expected {SKIP} unchecked");
}
#[then("the selected drawing and its preview remain displayed")]
fn drawing_kept(world: &mut ApplicationWorld) {
    let name = if published_drawings::staged_folder(world).is_some() {
        published_drawings::second_drawing_name(world)
    } else {
        DRAWING_FILE
    };
    expect_name_and_preview(world, name);
}
#[given("two saved drawings are visible in the same folder")]
fn two_drawings(world: &mut ApplicationWorld) {
    if published_drawings::show_staged_drawings(world) {
        return;
    }
    cancel_dialog(world);
    let folder = models_folder(world);
    saved_document(world, GuiTemplateKind::Drawing, &folder, DRAWING_FILE);
    saved_document(
        world,
        GuiTemplateKind::Drawing,
        &folder,
        SECOND_DRAWING_FILE,
    );
    open_dialog(world);
    look_in(world, &folder);
}
#[given("the first drawing is selected in the Options pane")]
fn first_selected(world: &mut ApplicationWorld) {
    let name = published_drawings::first_drawing_name(world);
    select_row(world, name);
}
#[when("I select the second drawing")]
#[when("the user selects the second drawing")]
fn select_second(world: &mut ApplicationWorld) {
    let name = published_drawings::second_drawing_name(world);
    select_row(world, name);
}
#[then("the second drawing is selected and its file name is displayed")]
fn second_shown(world: &mut ApplicationWorld) {
    let shown = file_name_shown(world);
    let name = published_drawings::second_drawing_name(world);
    assert!(shown.contains(name), "file name {shown:?}: expected {name}");
}
#[when("I select the first drawing again")]
#[when("the user selects the first drawing again")]
fn select_first_again(world: &mut ApplicationWorld) {
    first_selected(world);
}
#[then("the drawing remains selected without being opened")]
fn still_selected(world: &mut ApplicationWorld) {
    world.expect_text("Look in");
    let shown = file_name_shown(world);
    let name = published_drawings::first_drawing_name(world);
    assert!(shown.contains(name), "file name {shown:?}: expected {name}");
}
