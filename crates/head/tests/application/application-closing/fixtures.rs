// SPDX-License-Identifier: GPL-2.0-only
//! Disposable fixtures for the application exit scenarios (#4030).
use super::super::{diagnostics, document_opening, scenario_root};
use super::ApplicationWorld;
use cucumber::given;
use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_PART, DocumentType};
use std::path::{Path, PathBuf};

const PART: &str = "fixtures/cad/extrusion/extrusion-rect-8mm.opd";
const ASSEMBLY: &str = "fixtures/cad/blank-metric-assembly.oad";

#[given(
    "a writable copy of `fixtures/cad/extrusion/extrusion-rect-8mm.opd` is open in the MCAD Software"
)]
fn open_part_copy(world: &mut ApplicationWorld) {
    open_clean_copy(world, PART, DOCUMENT_PART);
}

#[given("a writable copy of `fixtures/cad/blank-metric-assembly.oad` is open in the MCAD Software")]
#[given(
    "the saved assembly fixture \"fixtures/cad/blank-metric-assembly.oad\" is the only open model document and is unchanged in the MCAD Software"
)]
fn open_assembly_copy(world: &mut ApplicationWorld) {
    open_clean_copy(world, ASSEMBLY, DOCUMENT_ASSEMBLY);
}

fn copy_fixture(world: &mut ApplicationWorld, relative: &str) -> PathBuf {
    let source = scenario_root().join(relative);
    let destination = world.scratch_folder().join(source.file_name().unwrap());
    std::fs::copy(&source, &destination)
        .unwrap_or_else(|cause| panic!("fixture {source:?}: expected a writable copy: {cause}"));
    destination
}

fn open_clean_copy(world: &mut ApplicationWorld, relative: &str, kind: DocumentType) {
    let path = copy_fixture(world, relative);
    world.open_fixture(&path);
    let document = world.active_id();
    let state = world.state();
    assert_eq!(state.documents.len(), 1);
    assert_eq!(state.document_type, Some(kind));
    assert!(!state.document_storage[&document].unsaved_edits);
    world.closed_document = Some(document);
}

#[given("the user creates a sketch on a datum plane without saving the document")]
fn create_unsaved_sketch(world: &mut ApplicationWorld) {
    world.start_sketch();
    assert_dirty(world);
}

#[given(
    "the user inserts `fixtures/cad/extrusion/extrusion-rect-8mm.opd` as an occurrence without saving the assembly"
)]
fn insert_unsaved_occurrence(world: &mut ApplicationWorld) {
    let component = copy_fixture(world, PART);
    place_from_file(world, &component);
    assert_dirty(world);
    world.expect_text("extrusion-rect-8mm.opd:1");
}

fn place_from_file(world: &mut ApplicationWorld, component: &Path) {
    world.click_text("Place");
    world.click_text("Browse");
    document_opening::look_in(world, component.parent().unwrap());
    document_opening::select_row(world, component.file_name().unwrap().to_str().unwrap());
    document_opening::choose_open(world);
    let place = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Place")
        .max_by(|a, b| a.center()[1].total_cmp(&b.center()[1]))
        .expect("expected the Place Component confirmation");
    world.click(diagnostics::pointer_pixels(place.center()));
    document_opening::settle(world);
}

fn assert_dirty(world: &mut ApplicationWorld) {
    let document = world
        .closed_document
        .expect("expected the writable fixture");
    assert_eq!(world.active_id(), document);
    assert!(world.state().document_storage[&document].unsaved_edits);
}
