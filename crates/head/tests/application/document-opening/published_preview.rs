// SPDX-License-Identifier: GPL-2.0-only
//! Keep the published preview target separate from the already active part.
use super::{
    ApplicationWorld, expect_name_and_preview, holds_text, look_in, published_entry::fixture,
    select_row, settle,
};
use cucumber::{given, then};
use std::path::{Path, PathBuf};

const PART_ONE: &str = "quick-access-toolbar/PartOne.opd";
const EXTRUSION: &str = "extrusion/extrusion-rect-8mm.opd";
const PREVIEW_FILE: &str = "extrusion-rect-8mm.opd";
const NAMED_VIEWS_FOLDER: &str = "named-views";
const NAMED_VIEWS: [&str; 2] = ["Open Dialog Blue", "Open Dialog Neutral"];

#[given("`fixtures/cad/quick-access-toolbar/PartOne.opd` is open and active in the MCAD Software")]
fn part_one_active(world: &mut ApplicationWorld) {
    let path = fixture(PART_ONE);
    world.open_fixture(&path);
    let id = world.active_id();
    assert_eq!(
        world.state().document_storage[&id].path.as_deref(),
        path.to_str()
    );
    world.previous_document = Some(id);
}

#[given("a folder contains a saved part with a preview and metadata")]
fn saved_preview_part(world: &mut ApplicationWorld) {
    world.saved_path = Some(fixture(EXTRUSION));
}

#[given(
    "the folder contains `fixtures/cad/extrusion/extrusion-rect-8mm.opd` with a saved preview and metadata"
)]
fn published_preview_part(world: &mut ApplicationWorld) {
    assert_eq!(world.saved_path.as_ref(), Some(&fixture(EXTRUSION)));
}

pub(super) fn inspection_path(world: &ApplicationWorld) -> Option<&Path> {
    world
        .saved_path
        .as_deref()
        .filter(|path| path.ends_with(EXTRUSION))
}

pub(super) fn published_part_details(world: &ApplicationWorld) {
    let path = inspection_path(world).unwrap();
    let size = path.metadata().unwrap().len();
    for row in ["Author:", "Part Number:", &format!("Size: {size} B")] {
        assert!(
            holds_text(world, row),
            "expected {row} in the published part details"
        );
    }
}

#[given("`fixtures/cad/extrusion/extrusion-rect-8mm.opd` is selected with its name and preview")]
#[given("`fixtures/cad/extrusion/extrusion-rect-8mm.opd` is selected in the detailed file list")]
#[given("`fixtures/cad/extrusion/extrusion-rect-8mm.opd` is selected in Details view")]
#[given(
    "`fixtures/cad/extrusion/extrusion-rect-8mm.opd` is selected with saved document properties and a thumbnail"
)]
pub(super) fn preview_target_selected(world: &mut ApplicationWorld) {
    select_preview_path(world, fixture(EXTRUSION));
}

pub(super) fn select_preview_path(world: &mut ApplicationWorld, path: PathBuf) {
    let name = path.file_name().unwrap().to_str().unwrap();
    look_in(world, path.parent().unwrap());
    select_row(world, name);
    expect_name_and_preview(world, name);
    world.saved_path = Some(path);
}

#[given(
    "the user opens a writable copy of `fixtures/cad/extrusion/extrusion-rect-8mm.opd`, creates named views Open Dialog Blue and Open Dialog Neutral, saves and closes it"
)]
fn prepare_named_views(world: &mut ApplicationWorld) {
    let folder = world.scratch_folder().join(NAMED_VIEWS_FOLDER);
    std::fs::create_dir(&folder).unwrap();
    let path = folder.join(PREVIEW_FILE);
    std::fs::write(&path, std::fs::read(fixture(EXTRUSION)).unwrap()).unwrap();
    world.open_fixture(&path);
    let active = world.active_id();
    for view in NAMED_VIEWS {
        world
            .app()
            .owner
            .document_commands(active)
            .create_design_view(view)
            .unwrap();
    }
    world.press_save_shortcut();
    settle(world);
    world.close_active_tab();
    settle(world);
    assert!(!world.document_is_open(active));
    world.saved_path = Some(path);
}

pub(super) fn named_views_path(world: &ApplicationWorld) -> Option<&Path> {
    let path = world.saved_path.as_deref()?;
    (path.parent()?.file_name()?.to_str() == Some(NAMED_VIEWS_FOLDER)).then_some(path)
}

#[given(
    "the saved working native part has named design views `Open Dialog Blue` and `Open Dialog Neutral`"
)]
fn named_views_saved(world: &mut ApplicationWorld) {
    let path = named_views_path(world).unwrap();
    let views = oblikovati_persistence::package::design_view_names_of_file(path);
    for name in NAMED_VIEWS {
        assert!(
            views.iter().any(|view| view == name),
            "missing saved view {name:?}"
        );
    }
}

#[then("Last Active and the part's named design views are displayed")]
#[then("Last Active, the primary view, and the part's named design views are displayed")]
fn view_rows(world: &mut ApplicationWorld) {
    for row in ["Last Active", "Primary"]
        .into_iter()
        .chain(named_view_rows(world).iter().copied())
    {
        world.expect_text(row);
    }
}

fn named_view_rows(world: &ApplicationWorld) -> &'static [&'static str] {
    if named_views_path(world).is_some() {
        &NAMED_VIEWS
    } else {
        &[super::VIEW_NAME]
    }
}

#[then("the Design View list closes")]
fn view_list_closed(world: &mut ApplicationWorld) {
    settle(world);
    for view in named_view_rows(world) {
        world.expect_no_text(view);
    }
}
