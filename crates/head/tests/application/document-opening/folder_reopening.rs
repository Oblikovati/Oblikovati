// SPDX-License-Identifier: GPL-2.0-only
//! Verify the Open dialog restores the original child folder after cancellation.
use super::*;

pub(super) const NAVIGATION_DRAWING: &str = "NavigationDrawing.odd";

#[given(
    "a writable copy of `fixtures/cad/document-opening/folder-navigation/NativeParent/NativeChild/NavigationDrawing.odd` is staged in a child folder"
)]
fn stage_navigation_drawing(world: &mut ApplicationWorld) {
    let path = published_entry::fixture(
        "document-opening/folder-navigation/NativeParent/NativeChild/NavigationDrawing.odd",
    );
    let folder = world
        .scratch_folder()
        .join("NativeParent")
        .join(NATIVE_CHILD);
    std::fs::create_dir_all(&folder).unwrap();
    let staged = folder.join(NAVIGATION_DRAWING);
    std::fs::write(&staged, std::fs::read(path).unwrap()).unwrap();
    world.saved_path = Some(staged);
}

#[given("the user opens and saves that drawing to establish its preview")]
fn save_navigation_drawing(world: &mut ApplicationWorld) {
    let path = world.saved_path.clone().unwrap();
    world.reopen_through_dialog(&path);
    published_drawings::save_preview(world, &path);
    world.previous_document = Some(world.active_id());
}

#[given("a drawing was opened from a child folder in the MCAD Software")]
#[given("a drawing was opened from that child folder in the MCAD Software")]
#[given("`NavigationDrawing.odd` was opened from that child folder")]
fn opened_from_child(world: &mut ApplicationWorld) {
    if navigation_variant(world) {
        let id = world.previous_document.unwrap();
        assert_eq!(
            world.state().document_storage[&id].path.as_deref(),
            world.saved_path.as_ref().and_then(|path| path.to_str())
        );
        return;
    }
    let folder = child_folder(world);
    let saved = saved_document(world, GuiTemplateKind::Drawing, &folder, DRAWING_FILE);
    world.previous_document = Some(saved);
    open_dialog(world);
    look_in(world, &folder);
    select_row(world, DRAWING_FILE);
    choose_open(world);
}
#[given("that drawing remains active")]
fn drawing_active(world: &mut ApplicationWorld) {
    let state = world.state();
    assert_eq!(state.document, world.previous_document);
    assert_eq!(
        state.document_type,
        Some(oblikovati_api::types::DOCUMENT_DRAWING)
    );
    world.expect_no_text("Look in");
}
#[given("the Open dialog is showing the same child folder")]
fn showing_child(world: &mut ApplicationWorld) {
    open_dialog(world);
    original_folder(world);
}
#[when("I reopen the Open dialog in the same session")]
#[when("the user reopens the Open dialog in the same session")]
fn reopen(world: &mut ApplicationWorld) {
    open_dialog(world);
}
#[then("it displays the original child folder")]
fn original_folder(world: &mut ApplicationWorld) {
    world.expect_text(navigation_drawing_name(world));
}
fn navigation_drawing_name(world: &ApplicationWorld) -> &'static str {
    if navigation_variant(world) {
        NAVIGATION_DRAWING
    } else {
        DRAWING_FILE
    }
}
#[then("the file name field identifies the previously opened drawing")]
fn name_previous(world: &mut ApplicationWorld) {
    let shown = file_name_shown(world);
    let name = navigation_drawing_name(world);
    assert!(shown.contains(name), "file name {shown:?}: expected {name}");
}
#[then("the drawing preview is displayed")]
fn drawing_preview(world: &mut ApplicationWorld) {
    world.expect_no_text("No preview image");
}
