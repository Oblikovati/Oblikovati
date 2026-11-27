// SPDX-License-Identifier: GPL-2.0-only
//! Open real native fixture packages before checking Save As replacement.
use super::*;
use oblikovati_api::{
    client::{Caller, Client},
    wire::BodyPhysicalPropertiesArgs,
};
use std::{
    fs,
    path::{Path, PathBuf},
};

const SAVED_EMPTY: &[u8] = include_bytes!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../../test-utilities/fixtures/document-saving/SavedEmptyPart.opd"
));
const EXISTING_EMPTY: &[u8] = include_bytes!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../../test-utilities/fixtures/document-saving/SaveAsFolder/ExistingPart.opd"
));
const EXISTING_SOLID: &[u8] = include_bytes!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../../test-utilities/fixtures/document-saving/SaveAsFolder/ExistingSolidPart.opd"
));
pub(super) const EXISTING_NAME: &str = "ExistingPart.opd";
pub(super) const SOLID_NAME: &str = "ExistingSolidPart.opd";
const SUBFOLDER_NAME: &str = "Subfolder";
const SAVE_AS_PACKAGE: &str = "fixtures/cad/document-saving/SaveAsNativeFolder";

pub(super) fn native_save_as_folder() -> Option<PathBuf> {
    let folder = std::env::current_dir().ok()?;
    (folder.ends_with("SaveAsNativeFolder") && folder.join(EXISTING_NAME).is_file())
        .then_some(folder)
}

pub(super) fn published_save_as_parts() -> PathBuf {
    super::super::scenario_root().join(SAVE_AS_PACKAGE)
}

#[given(
    "the current folder is a writable copy of `fixtures/cad/document-saving/SaveAsNativeFolder` with its existing part files"
)]
#[given(
    "the initial folder is a writable copy of `fixtures/cad/document-saving/SaveAsNativeFolder` with its `Subfolder` child"
)]
#[given(
    "the current folder is a writable copy of `fixtures/cad/document-saving/SaveAsNativeFolder`"
)]
#[given(
    "the current folder is a writable copy of `fixtures/cad/document-saving/SaveAsNativeFolder` with its part files and subfolder"
)]
#[given(
    "the current folder is a writable copy of `fixtures/cad/document-saving/SaveAsNativeFolder` with its part file and subfolder"
)]
fn published_save_as_folder(world: &mut ApplicationWorld) {
    let folder = native_save_as_folder()
        .expect("expected the test process in its disposable SaveAsNativeFolder");
    for name in [EXISTING_NAME, SOLID_NAME] {
        let path = folder.join(name);
        assert_eq!(
            fs::read(&path).unwrap(),
            fs::read(published_save_as_parts().join(name)).unwrap()
        );
        assert!(!path.metadata().unwrap().permissions().readonly());
        world.expect_text(name);
    }
    assert!(folder.join(SUBFOLDER_NAME).is_dir());
    dialog_navigation::observe_native_save_folder(world);
}

#[given(
    "a writable copy of `fixtures/cad/blank-metric-part.opd` is open as a saved empty part in the MCAD Software"
)]
#[given(
    "a writable copy of `fixtures/cad/blank-metric-part.opd` is open as a saved active part in the MCAD Software"
)]
fn open_published_empty_part(world: &mut ApplicationWorld) {
    let source = super::super::scenario_root().join("fixtures/cad/blank-metric-part.opd");
    let path = world.scratch_folder().join("blank-metric-part.opd");
    fs::write(&path, fs::read(source).unwrap()).unwrap();
    world.reopen_through_dialog(&path);
    let id = world.active_id();
    assert_eq!(
        world.state().document_storage[&id].path.as_deref(),
        path.to_str()
    );
    assert_eq!(world.browser_sketch_count(), 0);
    world.saved_path = Some(path);
}

#[given("`fixtures/cad/quick-access-toolbar/PartOne.opd` is open in the MCAD Software")]
fn open_other_published_part(world: &mut ApplicationWorld) {
    let path = super::super::scenario_root().join("fixtures/cad/quick-access-toolbar/PartOne.opd");
    world.reopen_through_dialog(&path);
    let id = world.active_id();
    assert_eq!(
        world.state().document_storage[&id].path.as_deref(),
        path.to_str()
    );
    world.other_document = Some(id);
}

#[given("the user creates a new unsaved metric part")]
fn new_unsaved_metric_part(world: &mut ApplicationWorld) {
    world.add_part();
}

#[given("the user creates and finishes an empty sketch on the XY datum plane")]
fn finish_empty_sketch(world: &mut ApplicationWorld) {
    world.add_finished_sketch();
}

#[given("the unsaved part containing that sketch is active")]
#[given("the unsaved modified part is active")]
fn active_modified_part(world: &mut ApplicationWorld) {
    let id = world.active_id();
    let state = world.state();
    assert!(state.document_storage[&id].path.is_none());
    assert!(state.document_storage[&id].unsaved_edits);
    assert_eq!(world.browser_sketch_count(), 1);
}

#[test]
fn document_saving_native_fixtures_reopen_and_survive_declined_replacement() {
    let mut world = ApplicationWorld::new();
    let root = world.scratch_folder();
    let folder = root.join("SaveAsFolder");
    fs::create_dir_all(folder.join(SUBFOLDER_NAME)).unwrap();
    fs::write(root.join("SavedEmptyPart.opd"), SAVED_EMPTY).unwrap();
    fs::write(folder.join(EXISTING_NAME), EXISTING_EMPTY).unwrap();
    fs::write(folder.join(SOLID_NAME), EXISTING_SOLID).unwrap();
    open_empty_part(
        &mut world,
        &root.join("SavedEmptyPart.opd"),
        "SavedEmptyPart",
    );
    open_empty_part(&mut world, &folder.join(EXISTING_NAME), "ExistingPart");
    verify_solid_part(&mut world, &folder.join(SOLID_NAME));
    decline_overwrite(&mut world, &folder);
    world.close();
}

fn open_empty_part(world: &mut ApplicationWorld, path: &Path, title: &str) {
    world.reopen_through_dialog(path);
    let document = world.active_id();
    assert_eq!(world.document_name(document), title);
    assert_eq!(world.browser_sketch_count(), 0);
    let bodies = Client::new(world.app().owner.document_host())
        .body()
        .list(&Default::default())
        .unwrap();
    assert!(bodies.bodies.unwrap_or_default().is_empty());
}

fn verify_solid_part(world: &mut ApplicationWorld, path: &Path) {
    world.reopen_through_dialog(path);
    let document = world.active_id();
    assert_eq!(world.document_name(document), "ExistingSolidPart");
    super::super::workspace_layout::browser_feature::expect_solid_visible(world);
    expect_40_by_25_by_20_solid(world);
}

pub(super) fn expect_40_by_25_by_20_solid(world: &mut ApplicationWorld) {
    let client = Client::new(world.app().owner.document_host());
    let bodies = client.body().list(&Default::default()).unwrap();
    let bodies = bodies.bodies.unwrap();
    let [solid] = bodies.as_slice() else {
        panic!("expected one fixture body, got {}", bodies.len());
    };
    assert!(solid.solid);
    assert_eq!(solid.faces, 6);
    assert_eq!(solid.name, "Solid1");
    verify_body_dimensions(&client, solid.index);
}

fn verify_body_dimensions<T: Caller>(client: &Client<T>, index: i64) {
    let body = client
        .body()
        .physical_properties(&BodyPhysicalPropertiesArgs {
            body_index: index,
            ..Default::default()
        })
        .unwrap();
    assert_measure(body.volume_mm3, 20_000.);
    assert_measure(body.surface_area_mm2, 4_600.);
    // The principal moments of a rectangular prism recover its three side lengths.
    let mut moments = body.principal_moments_gmm2;
    moments.sort_by(f64::total_cmp);
    for (moment, sum_of_squares) in moments.into_iter().zip([1025., 2000., 2225.]) {
        assert_measure(moment / body.mass_g, sum_of_squares / 12.);
    }
}

fn assert_measure(measured: f64, expected: f64) {
    assert!(
        (measured - expected).abs() < 1e-6,
        "measured {measured}: expected {expected}"
    );
}

fn decline_overwrite(world: &mut ApplicationWorld, folder: &Path) {
    let unsaved = world.add_part();
    world.press_save_shortcut();
    world.enter_in_field(SAVE_IN, folder.to_str().unwrap());
    world.expect_text(SOLID_NAME);
    world.expect_text(SUBFOLDER_NAME);
    world.click_text(EXISTING_NAME);
    world.click_text("Save");
    world.expect_text("already exists");
    world.click_text("No");
    world.expect_text(SAVE_IN);
    world.click_text("Cancel");
    assert_eq!(world.active_id(), unsaved);
    assert!(world.state().document_storage[&unsaved].path.is_none());
    assert_eq!(
        fs::read(folder.join(EXISTING_NAME)).unwrap(),
        EXISTING_EMPTY
    );
    assert_eq!(fs::read(folder.join(SOLID_NAME)).unwrap(), EXISTING_SOLID);
}
