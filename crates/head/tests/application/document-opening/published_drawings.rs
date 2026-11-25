// SPDX-License-Identifier: GPL-2.0-only
//! Stage the published drawing variants and save their previews through the desktop.
use super::{
    ApplicationWorld, DRAWING_FILE, PART_FILE, SECOND_DRAWING_FILE, expect_name_and_preview,
    holds_text, look_in, published_entry::fixture, settle,
};
use cucumber::given;
use oblikovati_persistence::package::{DocumentPackage, preview_picture_of_file};
use std::{
    fs,
    path::{Path, PathBuf},
};

const FIXTURE_FOLDER: &str = "document-opening";
const STAGED: &str = "published-opening";
pub(super) const FIRST: &str = "OpenDrawingA.odd";
const SECOND: &str = "OpenDrawingB.odd";
const PART: &str = "OpenPart.opd";

#[given(
    "writable copies of `fixtures/cad/document-opening/OpenDrawingA.odd`, `OpenDrawingB.odd` and `OpenPart.opd` are staged in the same working folder"
)]
fn stage_drawings(world: &mut ApplicationWorld) {
    let folder = world.scratch_folder().join(STAGED);
    fs::create_dir(&folder).unwrap();
    for name in [FIRST, SECOND, PART] {
        let from = fixture(&format!("{FIXTURE_FOLDER}/{name}"));
        let to = folder.join(name);
        fs::copy(&from, &to).unwrap();
        let mut permissions = fs::metadata(&to).unwrap().permissions();
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            permissions.set_mode(permissions.mode() | 0o200);
        }
        #[cfg(not(unix))]
        permissions.set_readonly(false);
        fs::set_permissions(to, permissions).unwrap();
    }
}

pub(super) fn staged_folder(world: &ApplicationWorld) -> Option<PathBuf> {
    let folder = world.scratch.as_ref()?.path().join(STAGED);
    folder.is_dir().then_some(folder)
}

pub(super) fn part_name(world: &ApplicationWorld) -> &'static str {
    staged_folder(world).map_or(PART_FILE, |_| PART)
}

pub(super) fn first_drawing_name(world: &ApplicationWorld) -> &'static str {
    staged_folder(world).map_or(DRAWING_FILE, |_| FIRST)
}

pub(super) fn second_drawing_name(world: &ApplicationWorld) -> &'static str {
    staged_folder(world).map_or(SECOND_DRAWING_FILE, |_| SECOND)
}

pub(super) fn staged_drawing_id(world: &mut ApplicationWorld, name: &str) -> Option<u64> {
    let path = staged_folder(world)?.join(name);
    let state = world.state();
    state
        .drawing_documents
        .keys()
        .copied()
        .find(|id| state.document_storage[id].path.as_deref() == path.to_str())
}

#[given("the user opens both staged drawings in separate tabs")]
fn open_both_staged(world: &mut ApplicationWorld) {
    let folder = staged_folder(world).unwrap();
    world.open_fixture(&folder.join(FIRST));
    world.open_fixture(&folder.join(SECOND));
    staged_tabs_open(world);
}

#[given("the user opens only the staged OpenDrawingA.odd")]
fn open_only_first(world: &mut ApplicationWorld) {
    let folder = staged_folder(world).unwrap();
    world.open_fixture(&folder.join(FIRST));
    staged_first_only(world);
}

#[given("`OpenDrawingA.odd` and `OpenDrawingB.odd` are open from their package paths")]
fn staged_tabs_open(world: &mut ApplicationWorld) {
    let first = staged_drawing_id(world, FIRST).expect("expected staged Drawing A to be open");
    let second = staged_drawing_id(world, SECOND).expect("expected staged Drawing B to be open");
    assert_ne!(first, second);
    world.previous_document = Some(first);
    world.other_document = Some(second);
}

#[given("`OpenDrawingA.odd` is open and `OpenDrawingB.odd` is closed")]
fn staged_first_only(world: &mut ApplicationWorld) {
    world.previous_document =
        Some(staged_drawing_id(world, FIRST).expect("expected staged Drawing A"));
    assert!(staged_drawing_id(world, SECOND).is_none());
}

pub(super) fn show_staged_drawings(world: &mut ApplicationWorld) -> bool {
    let Some(folder) = staged_folder(world) else {
        return false;
    };
    look_in(world, &folder);
    staged_drawing_rows(world);
    true
}

#[given("`OpenDrawingA.odd` and `OpenDrawingB.odd` are visible in the same folder")]
fn staged_drawing_rows(world: &mut ApplicationWorld) {
    for name in [FIRST, SECOND] {
        world.expect_text(name);
    }
}

#[given("`OpenDrawingB.odd` is selected in the Options pane")]
fn second_drawing_options(world: &mut ApplicationWorld) {
    expect_name_and_preview(world, SECOND);
    world.expect_text("Load in Defer Mode");
    world.expect_text("Skip all unresolved files");
}

#[given(
    "the user opens and saves each drawing to establish its sheet preview, then closes the drawings"
)]
fn save_drawing_previews(world: &mut ApplicationWorld) {
    let folder = staged_folder(world).unwrap();
    for name in [FIRST, SECOND] {
        let drawing = folder.join(name);
        world.open_fixture(&drawing);
        save_preview(world, &drawing);
        world.close_active_tab();
        settle(world);
    }
    assert!(world.state().drawing_documents.is_empty());
}

pub(super) fn save_preview(world: &mut ApplicationWorld, path: &Path) {
    world.click_text("File");
    let position = world.run("Save As").center();
    world.hover(position.map(|coordinate| coordinate as i32));
    settle(world);
    world.click_text("Save As...");
    settle(world);
    world.enter_in_field("Save in", path.parent().unwrap().to_str().unwrap());
    world.click_text("Active Window on Save");
    world.click_text("Capture");
    settle(world);
    confirm_preview_save(world, path);
}

fn confirm_preview_save(world: &mut ApplicationWorld, path: &Path) {
    let name = path.file_name().unwrap().to_str().unwrap();
    world.click_text(name);
    world.click_text("Save");
    world.expect_text("Do you want to replace it?");
    world.click_text("Yes");
    settle(world);
    world.expect_no_text("Save in");
    assert!(
        preview_picture_of_file(path).is_some(),
        "expected the saved sheet preview in {path:?}"
    );
}

#[given("the folder contains `OpenPart.opd` and `OpenDrawingA.odd` with saved previews")]
fn staged_previews(world: &mut ApplicationWorld) {
    let folder = staged_folder(world).unwrap();
    for file in [PART, FIRST] {
        let path = folder.join(file);
        assert!(
            preview_picture_of_file(&path).is_some(),
            "expected a saved preview in {path:?}"
        );
        world.expect_text(file);
    }
    assert!(holds_text(world, "Type: Part"));
}

#[given(
    "the user opens each drawing, inserts a base view referencing the staged OpenPart.opd, saves it and closes it"
)]
fn create_staged_base_views(world: &mut ApplicationWorld) {
    let folder = staged_folder(world).unwrap();
    let part_path = folder.join(PART);
    world.open_fixture(&part_path);
    let part = world.active_id();
    for name in [FIRST, SECOND] {
        let path = folder.join(name);
        world.open_fixture(&path);
        insert_and_save_base_view(world, &path, &part_path);
        world.close_active_tab();
        settle(world);
        world.expect_active(part);
    }
}

fn insert_and_save_base_view(world: &mut ApplicationWorld, path: &Path, part: &Path) {
    let id = world.active_id();
    world.click_text("Base View");
    world.expect_text("Orientation");
    world.click_text("Apply");
    settle(world);
    assert!(!world.state().drawing_projections[&id].lines.is_empty());
    save_preview(world, path);
    let package = DocumentPackage::decode(&fs::read(path).unwrap()).unwrap();
    let drawing = package.restore_drawing().unwrap().unwrap();
    assert_eq!(drawing.model_reference(), part.to_str().unwrap());
    let saved = package.restore_saved_drawing_views().unwrap().unwrap();
    assert_eq!(saved.views.len(), 1);
    assert!(!saved.views[0].lines.is_empty());
}

#[given("`OpenPart.opd` is the active part and `OpenDrawingA.odd` is closed")]
#[given("`OpenPart.opd` is active and the closed `OpenDrawingA.odd` drawing is available")]
fn staged_part_active(world: &mut ApplicationWorld) {
    let folder = staged_folder(world).unwrap();
    let active = world.active_id();
    let state = world.state();
    assert_eq!(
        state.document_storage[&active].path.as_deref(),
        folder.join(PART).to_str()
    );
    assert!(state.drawing_documents.is_empty());
    assert_eq!(world.saved_path.as_ref(), Some(&folder.join(FIRST)));
}
