// SPDX-License-Identifier: GPL-2.0-only
//! Disposable fixtures for the application exit scenarios (#4030).
use super::super::{diagnostics, document_opening, scenario_root, window_geometry};
use super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_PART, DocumentType};
use std::path::{Path, PathBuf};

const PART: &str = "fixtures/cad/extrusion/extrusion-rect-8mm.opd";
const ASSEMBLY: &str = "fixtures/cad/blank-metric-assembly.oad";

#[given(
    "`fixtures/cad/extrusion/extrusion-rect-8mm.opd` has been staged as a writable `SavePart.opd` in a disposable directory"
)]
fn stage_save_part(world: &mut ApplicationWorld) {
    stage_part(world, "SavePart.opd");
}

#[given(
    "`fixtures/cad/extrusion/extrusion-rect-8mm.opd` has been staged as a writable `DiscardPart.opd` in a disposable directory"
)]
fn stage_discard_part(world: &mut ApplicationWorld) {
    stage_part(world, "DiscardPart.opd");
}

fn stage_part(world: &mut ApplicationWorld, name: &str) {
    let path = world.scratch_folder().join(name);
    let bytes = std::fs::read(scenario_root().join(PART)).unwrap();
    std::fs::write(&path, &bytes).unwrap();
    world.saved_path = Some(path);
    world.bytes_before_save = Some(bytes);
}

#[given("that staged part is the only open model document")]
fn open_staged_part(world: &mut ApplicationWorld) {
    let path = world.saved_path.clone().expect("expected a staged Part");
    open_clean_model_copy(world, &path, DOCUMENT_PART);
    let document = world.closed_document.unwrap();
    assert_eq!(
        world.state().document_storage[&document].path.as_deref(),
        path.to_str()
    );
}

#[given("the user creates a new unsaved metric part as the only open model document")]
fn new_unsaved_metric_part(world: &mut ApplicationWorld) {
    let document = world.add_template(oblikovati_gui::GuiTemplateKind::Part);
    let state = world.state();
    assert_eq!(state.documents.len(), 1);
    assert_eq!(state.document_type, Some(DOCUMENT_PART));
    assert_eq!(state.document_units.unwrap().length_unit, "mm");
    assert!(state.document_storage[&document].path.is_none());
    world.closed_document = Some(document);
}

#[given("the user creates and finishes a new XY sketch without saving")]
#[given("the user creates and finishes a new XY sketch")]
fn finish_unsaved_sketch(world: &mut ApplicationWorld) {
    let before = browser_sketch_names(world);
    world.add_finished_sketch();
    assert_dirty(world);
    let after = browser_sketch_names(world);
    let added: Vec<_> = after
        .into_iter()
        .filter(|name| !before.contains(name))
        .collect();
    assert_eq!(added.len(), 1, "expected exactly one new XY sketch");
    world.proposed_name = added.into_iter().next();
}

fn browser_sketch_names(world: &mut ApplicationWorld) -> Vec<String> {
    world
        .state()
        .browser
        .expect("expected the Part browser")
        .sketches
        .into_iter()
        .map(|sketch| sketch.name)
        .collect()
}

pub(in super::super) fn save_staged_part_on_exit(world: &mut ApplicationWorld) -> bool {
    let closing_feature = world
        .scenario_feature_path
        .as_deref()
        .is_some_and(|path| path.ends_with("application-closing/application-closing.feature"));
    if !closing_feature || !world.application_close_requested || world.bytes_before_save.is_none() {
        return false;
    }
    answer_staged_part_exit(world, "Yes");
    true
}

#[when("the user chooses No in the save-changes prompt")]
fn discard_staged_part_on_exit(world: &mut ApplicationWorld) {
    answer_staged_part_exit(world, "No");
}

fn answer_staged_part_exit(world: &mut ApplicationWorld, answer: &str) {
    super::prompt_identifies_document(world);
    world.save_capture("application-closing-ready");
    #[cfg(target_os = "linux")]
    super::super::x11_input::click(super::super::rendered_label_center(answer));
    #[cfg(not(target_os = "linux"))]
    {
        let position = diagnostics::pointer_pixels(world.run(answer).center());
        world.app().window.scripted = super::super::pointer_click_events(position, 1);
    }
    let mut app = world.app.take().expect("expected an open application");
    let outcome = crate::frames::run_frames(
        &mut app.window,
        crate::HeadOptions { frames: Some(30) },
        &mut app.owner,
    )
    .and_then(|summary| {
        if summary.attempts == 30 || !app.owner.documents().workspace().documents().is_empty() {
            return Err(format!(
                "application exit did not finish after choosing {answer}"
            ));
        }
        Ok(summary)
    });
    crate::frames::retire_head_run(outcome, &mut app.owner, &mut app.window).unwrap();
    world.application_close_requested = false;
}

#[then("the changed part is saved to its existing path and the MCAD Software closes")]
fn saved_part_and_closed(world: &mut ApplicationWorld) {
    assert!(world.app.is_none());
    let path = world.saved_path.as_ref().unwrap();
    assert_ne!(
        &std::fs::read(path).unwrap(),
        world.bytes_before_save.as_ref().unwrap(),
        "expected the changed Part at its existing path"
    );
}

#[then("the MCAD Software closes without saving the new sketch")]
fn discarded_part_and_closed(world: &mut ApplicationWorld) {
    assert!(world.app.is_none());
    let path = world.saved_path.as_ref().unwrap();
    assert_eq!(
        &std::fs::read(path).unwrap(),
        world.bytes_before_save.as_ref().unwrap(),
        "expected Discard to preserve the original Part bytes"
    );
}

#[when("the user starts the MCAD Software again and opens the working part")]
fn reopen_staged_part(world: &mut ApplicationWorld) {
    assert!(world.app.is_none());
    world.reopen_application();
    open_staged_part(world);
}

#[then("the new sketch is present in the reopened part")]
fn saved_sketch_is_present(world: &mut ApplicationWorld) {
    let sketches = browser_sketch_names(world);
    assert!(sketches.contains(world.proposed_name.as_ref().unwrap()));
}

#[then("the new sketch is absent from the reopened part")]
fn discarded_sketch_is_absent(world: &mut ApplicationWorld) {
    let sketches = browser_sketch_names(world);
    assert!(!sketches.contains(world.proposed_name.as_ref().unwrap()));
}

#[then("the MCAD Software remains open with the unsaved part and its sketch")]
fn first_save_cancellation_preserves_part(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    assert!(window_geometry("Save As").is_none());
    world.expect_no_text("before closing");
    let document = world.closed_document.unwrap();
    let state = world.state();
    assert_eq!(state.document, Some(document));
    assert_eq!(state.documents.len(), 1);
    assert!(state.document_storage[&document].path.is_none());
    assert!(state.document_storage[&document].unsaved_edits);
    saved_sketch_is_present(world);
    world.application_close_requested = false;
}

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
    open_clean_model_copy(world, &path, kind);
}

fn open_clean_model_copy(world: &mut ApplicationWorld, path: &Path, kind: DocumentType) {
    world.open_fixture(path);
    let document = world.active_id();
    let state = world.state();
    assert_eq!(state.documents.len(), 1);
    assert_eq!(state.document_type, Some(kind));
    assert!(!state.document_storage[&document].unsaved_edits);
    world.closed_document = Some(document);
}

#[given("the user creates a sketch on a datum plane without saving the document")]
#[given("the user creates a sketch on the XY datum plane without saving the document")]
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
