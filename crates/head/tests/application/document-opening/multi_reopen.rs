// SPDX-License-Identifier: GPL-2.0-only
//! Exercise existing and newly opened drawing identities in a multi-file selection.
use super::*;

#[given("two saved drawings are already open in separate tabs in the MCAD Software")]
fn two_open(world: &mut ApplicationWorld) {
    if published_drawings::staged_folder(world).is_some() {
        world.observed_counts = vec![documents(world)];
        assert!(world.document_is_open(world.previous_document.unwrap()));
        assert!(world.document_is_open(world.other_document.unwrap()));
        return;
    }
    let folder = models_folder(world);
    saved_document(world, GuiTemplateKind::Drawing, &folder, DRAWING_FILE);
    saved_document(
        world,
        GuiTemplateKind::Drawing,
        &folder,
        SECOND_DRAWING_FILE,
    );
    world.observed_counts = vec![documents(world)];
}
#[given("one of those drawings is active")]
fn one_active(world: &mut ApplicationWorld) {
    one_tab_active(world);
}
#[given("the Open dialog displays their exact saved files")]
fn dialog_shows_files(world: &mut ApplicationWorld) {
    let folder = published_drawings::staged_folder(world).unwrap_or_else(|| models_folder(world));
    open_dialog(world);
    look_in(world, &folder);
    switch_to_tiles(world);
}
#[when("I select both drawings and choose Open")]
#[when("the user selects both drawings and chooses Open")]
fn select_both_open(world: &mut ApplicationWorld) {
    rectangle_over_drawings(world);
    two_selected(world);
    choose_open(world);
}
#[then("both existing drawing tabs remain available")]
#[then("no duplicate tab is created for either drawing")]
fn tabs_kept(world: &mut ApplicationWorld) {
    assert_eq!(documents(world), world.observed_counts[0], "documents");
    if published_drawings::staged_folder(world).is_some() {
        for id in [
            world.previous_document.unwrap(),
            world.other_document.unwrap(),
        ] {
            let title = world.document_name(id);
            world.expect_tab(&title);
        }
    }
}
#[then("one of those drawing tabs becomes active")]
fn one_tab_active(world: &mut ApplicationWorld) {
    let active = world.active_id();
    if published_drawings::staged_folder(world).is_some() {
        assert!([world.previous_document, world.other_document].contains(&Some(active)));
    }
}
#[given("a saved drawing is already open in a tab in the MCAD Software")]
fn one_open(world: &mut ApplicationWorld) {
    if published_drawings::staged_folder(world).is_some() {
        let name = published_drawings::first_drawing_name(world);
        let id = published_drawings::staged_drawing_id(world, name)
            .expect("expected the staged drawing");
        assert_eq!(world.active_id(), id);
        return;
    }
    let folder = models_folder(world);
    saved_document(world, GuiTemplateKind::Drawing, &folder, DRAWING_FILE);
}
#[given("a second saved drawing with available references is not open")]
fn second_closed(world: &mut ApplicationWorld) {
    if published_drawings::staged_folder(world).is_some() {
        let name = published_drawings::second_drawing_name(world);
        assert!(published_drawings::staged_drawing_id(world, name).is_none());
        world.observed_counts = vec![documents(world)];
        return;
    }
    let folder = models_folder(world);
    saved_document(
        world,
        GuiTemplateKind::Drawing,
        &folder,
        SECOND_DRAWING_FILE,
    );
    close_active(world);
    world.observed_counts = vec![documents(world)];
}
#[given("the Open dialog displays the exact files for both drawings")]
fn dialog_both_files(world: &mut ApplicationWorld) {
    dialog_shows_files(world);
}
#[then("the previously open drawing retains its existing tab without a duplicate")]
fn no_duplicate(world: &mut ApplicationWorld) {
    assert_eq!(documents(world), world.observed_counts[0] + 1, "documents");
    if published_drawings::staged_folder(world).is_some() {
        let title = world.document_name(world.previous_document.unwrap());
        world.expect_tab(&title);
    }
}
#[then("the second drawing opens in a new tab")]
fn second_new_tab(world: &mut ApplicationWorld) {
    assert_eq!(documents(world), world.observed_counts[0] + 1, "documents");
}
#[then("the second drawing is active with its sheet displayed")]
fn second_active(world: &mut ApplicationWorld) {
    let state = world.state();
    let active = state.document.expect("expected an active document");
    let path = state.document_storage[&active]
        .path
        .clone()
        .unwrap_or_default();
    assert!(
        path.ends_with(published_drawings::second_drawing_name(world)),
        "active file {path:?}"
    );
    world.expect_text("Sheet 1 of 1");
}
