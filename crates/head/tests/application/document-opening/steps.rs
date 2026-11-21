// SPDX-License-Identifier: GPL-2.0-only
//! `features/application/document-opening`: choosing, inspecting and opening files.
use super::ApplicationWorld;
use crate::HeadWork;
use cucumber::{given, then, when};
use oblikovati_api::{
    client::Client,
    types::{Variant, VariantValue},
    wire::SetPropertyArgs,
};
use oblikovati_gui::GuiTemplateKind;
use oblikovati_viewport::ViewportPhysicalKey;
use std::path::{Path, PathBuf};

#[path = "drawing_fixture.rs"]
mod drawing_fixture;
#[path = "express_fixture.rs"]
mod express_fixture;
#[path = "lists.rs"]
mod lists;
#[cfg(target_os = "linux")]
#[path = "native_existing.rs"]
mod native_existing;
#[path = "selections.rs"]
mod selections;

pub(super) const PART_FILE: &str = "bracket.opd";
pub(super) const DRAWING_FILE: &str = "sheet.odd";
pub(super) const SECOND_DRAWING_FILE: &str = "layout.odd";
pub(super) const ASSEMBLY_FILE: &str = "gearbox.oad";
const ROW_READ_OFFSET: usize = 20;
const EMPTY_LIST_BELOW: usize = 100;
const NO_PREVIEW: &str = "No preview image";
const AUTHOR: &str = "Ada Lovelace";
const PART_NUMBER: &str = "PN-1001";
const VIEW_NAME: &str = "Exploded";
const SUMMARY_SET: &str = "Summary Information";
const TRACKING_SET: &str = "Design Tracking Properties";

pub(super) fn settle(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    world.pump(Vec::new());
}
pub(super) fn models_folder(world: &mut ApplicationWorld) -> PathBuf {
    let folder = world.scratch_folder().join("models");
    std::fs::create_dir_all(&folder).unwrap();
    folder
}
/// Save a new document of this kind where a scenario will look for it; it stays open.
pub(super) fn saved_document(
    world: &mut ApplicationWorld,
    kind: GuiTemplateKind,
    folder: &Path,
    file: &str,
) -> u64 {
    let id = world.add_template(kind);
    let path = folder.join(file);
    world
        .app()
        .owner
        .complete_save_path(id, Some(&path))
        .unwrap();
    settle(world);
    assert!(path.exists(), "saved {path:?}: expected a file on disk");
    world.saved_path = Some(path);
    id
}
fn record_property(world: &mut ApplicationWorld, id: u64, set: &str, name: &str, value: &str) {
    Client::new(world.app().owner.document_host())
        .documents()
        .set_property(&SetPropertyArgs {
            document: id,
            set: set.into(),
            name: name.into(),
            value: Variant {
                value: VariantValue::String(value.into()),
                ..Default::default()
            },
        })
        .unwrap();
}
/// A saved part with recorded properties and a design view; a second save keeps them on disk.
pub(super) fn saved_part(world: &mut ApplicationWorld, folder: &Path) -> u64 {
    let id = saved_document(world, GuiTemplateKind::Part, folder, PART_FILE);
    record_property(world, id, SUMMARY_SET, "Author", AUTHOR);
    record_property(world, id, TRACKING_SET, "Part Number", PART_NUMBER);
    world
        .app()
        .owner
        .document_commands(id)
        .create_design_view(VIEW_NAME)
        .unwrap();
    world
        .app()
        .owner
        .complete_save_path(id, Some(&folder.join(PART_FILE)))
        .unwrap();
    settle(world);
    id
}
pub(super) fn open_dialog(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::O, true);
    settle(world);
    world.expect_text("Look in");
}
pub(super) fn look_in(world: &mut ApplicationWorld, folder: &Path) {
    world.enter_in_field("Look in", folder.to_str().unwrap());
    settle(world);
}
pub(super) fn select_row(world: &mut ApplicationWorld, file: &str) {
    let at = world.run(file).center().map(|value| value as i32);
    world.click(at);
    settle(world);
}
pub(super) fn choose_open(world: &mut ApplicationWorld) {
    let open = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| run.text == "Open")
        .cloned()
        .expect("expected the dialog's Open");
    world.click(open.center().map(|value| value as i32));
    settle(world);
    world.pump(Vec::new());
}
pub(super) fn cancel_dialog(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    settle(world);
}
/// The field's text; a label and a field on one baseline render as one run.
pub(super) fn file_name_shown(world: &ApplicationWorld) -> String {
    let label = world.run("File name").clone();
    let line: String = world
        .frame
        .runs
        .iter()
        .filter(|run| (run.center()[1] - label.center()[1]).abs() < 8.)
        .map(|run| run.text.as_str())
        .collect();
    line.replace("Filename", "")
}
/// A chosen row against the list's own empty space lower down; the dialog's plain background
/// around the list differs from both.
pub(super) fn row_lit(world: &mut ApplicationWorld, text: &str) -> bool {
    let run = world.run(text).clone();
    let x = run.maximum[0] as usize + ROW_READ_OFFSET;
    let y = run.center()[1] as usize;
    world.pixel_at([x, y]) != world.pixel_at([x, y + EMPTY_LIST_BELOW])
}
pub(super) fn expect_name_and_preview(world: &ApplicationWorld, file: &str) {
    let shown = file_name_shown(world);
    assert!(shown.contains(file), "file name {shown:?}: expected {file}");
    world.expect_no_text(NO_PREVIEW);
}
pub(super) fn expect_placeholder(world: &ApplicationWorld) {
    world.expect_text(NO_PREVIEW);
}
pub(super) fn holds_text(world: &ApplicationWorld, text: &str) -> bool {
    let all: String = world
        .frame
        .runs
        .iter()
        .map(|run| run.text.as_str())
        .collect();
    all.contains(&text.replace(' ', ""))
}
fn open_on_saved_part(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    saved_part(world, &folder);
    open_dialog(world);
    look_in(world, &folder);
    select_row(world, PART_FILE);
}
/// The Details pane's control, right of the list: the toolbar has a Details of its own.
fn choose_pane(world: &mut ApplicationWorld, label: &str) {
    let at = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == label)
        .max_by(|first, second| first.minimum[0].total_cmp(&second.minimum[0]))
        .cloned()
        .unwrap_or_else(|| panic!("expected the {label} pane control"))
        .center();
    world.click(at.map(|value| value as i32));
    settle(world);
}
// Inspect a part file before opening it
#[given("a folder contains a saved part with a preview and recorded metadata")]
fn part_in_folder(world: &mut ApplicationWorld) {
    let active = world.active_id();
    let folder = models_folder(world);
    let id = saved_part(world, &folder);
    world.opened_value = Some(id.to_string());
    world.click_tab(active);
}
#[when("I press Control+O and navigate to that folder")]
#[when("the user presses Control+O and navigates to that folder")]
fn control_o_to_folder(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    open_dialog(world);
    look_in(world, &folder);
}
#[when("the user selects the saved part")]
fn select_saved_part(world: &mut ApplicationWorld) {
    let path = world
        .saved_path
        .clone()
        .filter(|path| path.extension().is_some_and(|extension| extension == "opd"));
    let Some(path) = path else {
        select_row(world, PART_FILE);
        return;
    };
    look_in(world, path.parent().expect("expected a saved part folder"));
    select_row(world, path.file_name().unwrap().to_str().unwrap());
}
#[then("the part row is selected and the file name field identifies it")]
fn part_row_selected(world: &mut ApplicationWorld) {
    assert!(row_lit(world, PART_FILE), "expected the row's accent");
    let shown = file_name_shown(world);
    assert!(shown.contains(PART_FILE), "file name {shown:?}");
}
#[then("the preview displays the part")]
#[then("the preview remains visible")]
#[then("the part preview remains visible")]
fn preview_shown(world: &mut ApplicationWorld) {
    world.expect_no_text(NO_PREVIEW);
}
#[then("the Options pane displays Model State and Design View controls")]
fn part_options(world: &mut ApplicationWorld) {
    assert!(holds_text(world, "Model State"), "expected Model State");
    world.expect_text("Design View");
}
#[then("the details include the file type, modification date, size, author and part number")]
fn part_details(world: &mut ApplicationWorld) {
    for row in ["Type: Part", "Modified:", "Size:"] {
        assert!(holds_text(world, row), "expected {row} in the details");
    }
    assert!(
        holds_text(world, &format!("Author: {AUTHOR}")),
        "expected the author"
    );
    assert!(
        holds_text(world, &format!("Part Number: {PART_NUMBER}")),
        "expected the part number"
    );
}
#[when("I cancel the Open dialog")]
#[when("the user cancels the Open dialog")]
fn cancel_open(world: &mut ApplicationWorld) {
    cancel_dialog(world);
}
#[then("the previously active part document remains displayed")]
fn part_still_shown(world: &mut ApplicationWorld) {
    world.expect_no_text("Look in");
    world.expect_active(world.previous_document.expect("expected the original part"));
}
// Inspect a drawing after selecting a part
#[given("the Open dialog is displaying a folder containing a saved part and drawing with previews")]
fn folder_with_part_and_drawing(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    saved_document(world, GuiTemplateKind::Drawing, &folder, DRAWING_FILE);
    saved_part(world, &folder);
    open_dialog(world);
    look_in(world, &folder);
}
#[given("the saved part is selected and its details are displayed")]
fn part_selected_details(world: &mut ApplicationWorld) {
    select_row(world, PART_FILE);
    choose_pane(world, "Details");
}
#[when("I select the drawing")]
#[when("the user selects the drawing")]
fn select_drawing(world: &mut ApplicationWorld) {
    select_row(world, DRAWING_FILE);
}
#[then("the file name field identifies the drawing")]
fn name_is_drawing(world: &mut ApplicationWorld) {
    let shown = file_name_shown(world);
    assert!(shown.contains(DRAWING_FILE), "file name {shown:?}");
}
#[then("the preview displays the drawing sheet")]
fn drawing_preview(world: &mut ApplicationWorld) {
    world.expect_no_text(NO_PREVIEW);
}
#[then("the Details pane identifies the drawing and displays its metadata")]
fn drawing_details(world: &mut ApplicationWorld) {
    for row in ["Type: Drawing", "Modified:", "Size:", "Author:"] {
        assert!(holds_text(world, row), "expected {row} in the details");
    }
}
#[when("I choose Options")]
#[when("the user chooses Options")]
fn choose_options(world: &mut ApplicationWorld) {
    choose_pane(world, "Options");
}
#[then("the drawing options include Load in Defer Mode and Skip all unresolved files")]
fn drawing_options(world: &mut ApplicationWorld) {
    world.expect_text("Load in Defer Mode");
    world.expect_text("Skip all unresolved files");
}
#[then("the dialog closes without opening the selected drawing")]
fn closed_unopened(world: &mut ApplicationWorld) {
    world.expect_no_text("Look in");
}
// Design views and properties
#[given("the Open dialog is open in the MCAD Software")]
fn dialog_open(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    world.add_part();
    open_dialog(world);
    look_in(world, &folder);
}
#[given("a saved part with named design views is selected")]
#[given("a saved part with a preview and metadata is selected")]
#[given("a saved part with a preview is selected")]
#[given("a saved part is selected with its file name and preview displayed")]
#[given("a saved part is selected with its name and preview displayed")]
#[given("a saved part with a preview is selected in the detailed file list")]
fn saved_part_selected(world: &mut ApplicationWorld) {
    cancel_dialog(world);
    open_on_saved_part(world);
    expect_name_and_preview(world, PART_FILE);
}
#[given("its Design View field displays Last Active")]
#[then("the Design View field still displays Last Active")]
fn view_last_active(world: &mut ApplicationWorld) {
    world.expect_text("Design View: Last Active");
}
#[then("Last Active and the part's named design views are displayed")]
#[then("Last Active, the primary view, and the part's named design views are displayed")]
fn view_rows(world: &mut ApplicationWorld) {
    for row in ["Last Active", "Primary", VIEW_NAME] {
        world.expect_text(row);
    }
}
#[then("the Design View list closes")]
fn view_list_closed(world: &mut ApplicationWorld) {
    settle(world);
    world.expect_no_text(VIEW_NAME);
}
#[then("the Open dialog remains open")]
fn dialog_stays(world: &mut ApplicationWorld) {
    world.expect_text("Look in");
}
#[when("I choose the selected file's properties link")]
#[when("the user chooses the selected file's properties link")]
fn choose_properties(world: &mut ApplicationWorld) {
    let at = world
        .frame
        .runs
        .iter()
        .rfind(|run| run.text == "Properties")
        .expect("expected the properties link")
        .center();
    world.click(at.map(|value| value as i32));
    settle(world);
}
#[then("a separate properties window opens for that part")]
fn properties_open(world: &mut ApplicationWorld) {
    world.expect_text("File Properties");
}
#[then("its General tab shows file type, location, size and file dates")]
fn general_rows(world: &mut ApplicationWorld) {
    for row in [
        "Type: Part",
        "Location:",
        "Size:",
        "Created:",
        "Modified:",
        "Accessed:",
    ] {
        assert!(holds_text(world, row), "expected {row} on General");
    }
}
fn choose_tab(world: &mut ApplicationWorld, tab: &str) {
    world.click_text(tab);
    settle(world);
}
#[when("I choose the Summary tab")]
#[when("the user chooses the Summary tab")]
fn summary_tab(world: &mut ApplicationWorld) {
    choose_tab(world, "Summary");
}
#[then("fields including Title, Subject, Author, Category and Comments are displayed")]
fn summary_rows(world: &mut ApplicationWorld) {
    for row in ["Title:", "Subject:", "Author:", "Category:", "Comments:"] {
        assert!(holds_text(world, row), "expected {row} on Summary");
    }
}
#[when("I choose the Project tab")]
#[when("the user chooses the Project tab")]
fn project_tab(world: &mut ApplicationWorld) {
    choose_tab(world, "Project");
}
#[then("fields including Part Number, Description, Revision Number and Designer are displayed")]
fn project_rows(world: &mut ApplicationWorld) {
    for row in [
        "Part Number:",
        "Description:",
        "Revision Number:",
        "Designer:",
    ] {
        assert!(holds_text(world, row), "expected {row} on Project");
    }
}
#[when("I choose the Thumbnail tab")]
#[when("the user chooses the Thumbnail tab")]
fn thumbnail_tab(world: &mut ApplicationWorld) {
    choose_tab(world, "Thumbnail");
}
#[then("the saved part preview is displayed")]
fn thumbnail_shown(world: &mut ApplicationWorld) {
    world.expect_no_text("No saved image");
}
#[when("I cancel the properties window without editing")]
#[when("the user cancels the properties window without editing")]
fn cancel_properties(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    settle(world);
}

#[test]
fn file_properties_title_close_preserves_the_open_dialog_selection() {
    let mut world = ApplicationWorld::new();
    dialog_open(&mut world);
    saved_part_selected(&mut world);
    choose_properties(&mut world);
    let (position, size) = super::diagnostics::window_geometry("File Properties").unwrap();
    let close =
        super::diagnostics::pointer_pixels([position[0] + size[0] - 13., position[1] + 10.]);
    world.click(close);
    open_again(&mut world);
    part_stays_selected(&mut world);
    cancel_dialog(&mut world);
    world.close();
}
#[then("the Open dialog is displayed again")]
fn open_again(world: &mut ApplicationWorld) {
    world.expect_no_text("File Properties");
    world.expect_text("Look in");
}
#[then("the same part remains selected with its file name and preview displayed")]
#[then("the selected part remains selected with its file name and preview displayed")]
#[then("the selected part, file name and preview remain displayed")]
fn part_stays_selected(world: &mut ApplicationWorld) {
    expect_name_and_preview(world, PART_FILE);
}
