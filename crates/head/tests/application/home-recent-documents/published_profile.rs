// SPDX-License-Identifier: GPL-2.0-only
//! Prepare the published Home package in an isolated Recent profile.
use super::*;
use oblikovati_model::application_recent::{RecentDocument, RecentDocumentAccess};
use std::fs::{self, File, FileTimes};
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

const PACKAGE: &str = "fixtures/cad/home-recent-native";
const DISPOSABLE: &str = "home-recent-documents";
const FILES: [&str; 5] = [
    "Part2.opd",
    "Part23.opd",
    "Part26.opd",
    "RecentAssembly.oad",
    "RecentDrawing.odd",
];
const COMPONENT: &str = "extrusion/extrusion-rect-8mm.opd";

#[given(
    "the MCAD Software starts with an isolated Recent profile prepared from a disposable copy of `fixtures/cad/home-recent-native`"
)]
fn prepare_published_profile(world: &mut ApplicationWorld) {
    let source = super::super::scenario_root().join(PACKAGE);
    let package = world.scratch_folder().join(DISPOSABLE);
    let now = SystemTime::now();
    for file in FILES.into_iter().chain([COMPONENT]) {
        copy_recent_file(&source, &package, file, now);
    }
    world.close();
    let mut options =
        crate::native::preferences::load_native_application_options(world.preferences_directory())
            .unwrap();
    options
        .set_recent_documents(recent_rows(&package, now))
        .unwrap();
    world.reopen_application();
    assert!(world.state().documents.is_empty());
    expect_recent_files(world);
}

fn copy_recent_file(source: &Path, package: &Path, file: &str, now: SystemTime) {
    let destination = package.join(file);
    fs::create_dir_all(destination.parent().unwrap()).unwrap();
    fs::copy(source.join(file), &destination)
        .unwrap_or_else(|error| panic!("copy {file}: {error}"));
    File::options()
        .write(true)
        .open(destination)
        .unwrap()
        .set_times(FileTimes::new().set_modified(now))
        .unwrap();
}

fn recent_rows(package: &Path, now: SystemTime) -> Vec<RecentDocument> {
    let now = now.duration_since(UNIX_EPOCH).unwrap().as_secs();
    FILES
        .iter()
        .zip(0_u64..)
        .map(|(file, offset)| RecentDocument {
            path: package.join(file).to_str().unwrap().to_owned(),
            last_opened: now - offset,
            pinned: false,
        })
        .collect()
}

fn published_profile_folder(world: &ApplicationWorld) -> Option<PathBuf> {
    world
        .scratch
        .as_ref()
        .map(|scratch| scratch.path().join(DISPOSABLE))
}

pub(super) fn uses_published_profile(world: &ApplicationWorld) -> bool {
    published_profile_folder(world).is_some_and(|folder| folder.is_dir())
}

pub(super) fn other_recent_names(world: &ApplicationWorld) -> &'static [&'static str] {
    if uses_published_profile(world) {
        &["Part2", "Part23", "RecentAssembly", "RecentDrawing"]
    } else {
        &[OTHERS[0], OTHERS[1], ASSEMBLY]
    }
}

pub(super) fn assembly_name(world: &ApplicationWorld) -> &'static str {
    if uses_published_profile(world) {
        "RecentAssembly"
    } else {
        ASSEMBLY
    }
}

pub(super) fn recent_name(world: &ApplicationWorld, name: &str) -> Option<RenderedRun> {
    let wanted = super::super::world::squeezed(name);
    home_run(world, name)
        .filter(|run| run.text == wanted)
        .or_else(|| clipped_recent_name(world, &wanted))
}

fn clipped_recent_name(world: &ApplicationWorld, name: &str) -> Option<RenderedRun> {
    let mark = match name {
        "RecentAssembly" => "A",
        "RecentDrawing" => "D",
        _ => return None,
    };
    world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text
                .strip_suffix("...")
                .is_some_and(|prefix| !prefix.is_empty() && name.starts_with(prefix))
                && in_home(run)
                && world.frame.runs.iter().any(|kind| {
                    let [x, y] = kind.center();
                    kind.text == mark
                        && in_home(kind)
                        && (y - run.center()[1]).abs() < ROW_REACH
                        && (0. ..32.).contains(&(run.minimum[0] - x))
                })
        })
        .cloned()
}

pub(super) fn expect_recent_files(world: &mut ApplicationWorld) {
    let folder = published_profile_folder(world).unwrap();
    let mut paths: Vec<_> = world
        .state()
        .recent_documents
        .into_iter()
        .map(|row| PathBuf::from(row.path))
        .collect();
    paths.sort();
    let mut expected = FILES.map(|file| folder.join(file));
    expected.sort();
    assert_eq!(paths, expected);
}

#[given("Part23.opd from the disposable package is open with Sketch1 being edited")]
fn edit_part23(world: &mut ApplicationWorld) {
    use super::super::workspace_layout::browser_feature::{choose_menu_row, right_click_row};
    let path = published_profile_folder(world).unwrap().join("Part23.opd");
    world.open_fixture(&path);
    let part = world.active_id();
    let state = world.state();
    assert_eq!(
        state.document_storage[&part].path.as_deref().map(Path::new),
        Some(path.as_path())
    );
    right_click_row(world, "Sketch1");
    choose_menu_row(world, "Edit Sketch");
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    world.previous_document = Some(part);
}

#[given("no other visible recent document matches \"Part26\"")]
#[given("a recent part named \"Part26\" is the only visible document matching that term")]
fn no_other_matches(world: &mut ApplicationWorld) {
    let matches = world
        .state()
        .recent_documents
        .into_iter()
        .filter(|row| row.name.contains(SEARCHED) && recent_name(world, &row.name).is_some())
        .count();
    assert_eq!(matches, 1, "expected only {SEARCHED} to match");
}
