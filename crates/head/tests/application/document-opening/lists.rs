// SPDX-License-Identifier: GPL-2.0-only
//! Document-opening scenarios on the panes around the list: its views, order and folders.
use super::*;

const CHILD_FOLDER: &str = "brackets";
/// Larger than the fixture's documents, for a row that sorts first by size.
const LARGE_FILE_BYTES: usize = 500_000;
const LARGE_FILE: &str = "archive.odd";

fn rows_in_order(world: &ApplicationWorld, names: &[&str]) -> Vec<String> {
    let mut rows: Vec<_> = world
        .frame
        .runs
        .iter()
        .filter(|run| names.contains(&run.text.as_str()))
        .map(|run| (run.center()[1], run.text.clone()))
        .collect();
    rows.sort_by(|first, second| first.0.total_cmp(&second.0));
    rows.into_iter().map(|(_, text)| text).collect()
}
// Navigation panes
#[given("its project navigation pane shows libraries and recent folders")]
#[then("libraries and recent folders are displayed again")]
fn project_pane(world: &mut ApplicationWorld) {
    world.expect_text("Libraries");
    world.expect_text("Recent Folders");
}
#[when("I switch to the system folder navigation pane")]
#[when("the user switches to the system folder navigation pane")]
fn system_pane(world: &mut ApplicationWorld) {
    world.click_text("Locations");
    settle(world);
}
#[then("common system locations are displayed in the navigation pane")]
fn system_rows(world: &mut ApplicationWorld) {
    world.expect_text("System Locations");
    world.expect_text("Computer");
}
#[when("I switch back to the project navigation pane")]
#[when("the user switches back to the project navigation pane")]
fn back_project(world: &mut ApplicationWorld) {
    world.click_text("Recent");
    settle(world);
}
// File views
#[when("I choose the thumbnail view control")]
#[when("the user chooses the thumbnail view control")]
fn tiles_control(world: &mut ApplicationWorld) {
    world.click_text("Tiles");
    settle(world);
}
#[then("files are displayed as thumbnail tiles")]
fn shown_as_tiles(world: &mut ApplicationWorld) {
    world.expect_no_text("Date modified");
}
#[when("I choose the details view control")]
#[when("the user chooses the details view control")]
fn details_control(world: &mut ApplicationWorld) {
    let at = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Details")
        .expect("expected the view control")
        .center();
    world.click(at.map(|value| value as i32));
    settle(world);
}
#[then("files are displayed in rows with Name, Date modified, Type and Size columns")]
fn shown_as_rows(world: &mut ApplicationWorld) {
    for column in ["Name", "Date modified", "Type", "Size"] {
        world.expect_text(column);
    }
}
// Size order
#[given("the detailed file list contains files of different sizes")]
fn different_sizes(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    std::fs::write(folder.join(LARGE_FILE), vec![b'x'; LARGE_FILE_BYTES]).unwrap();
}
#[given("the file list is sorted by decreasing size")]
fn sorted_down(world: &mut ApplicationWorld) {
    world.click_text("Size");
    settle(world);
    let rows = rows_in_order(world, &[PART_FILE, LARGE_FILE]);
    assert_eq!(rows[0], LARGE_FILE, "largest first");
}
#[then("the file list is sorted by increasing size")]
fn sorted_up(world: &mut ApplicationWorld) {
    let rows = rows_in_order(world, &[PART_FILE, LARGE_FILE]);
    assert_eq!(rows[0], PART_FILE, "smallest first");
}
// Folders
fn child_folder(world: &mut ApplicationWorld) -> PathBuf {
    let folder = models_folder(world).join(CHILD_FOLDER);
    std::fs::create_dir_all(&folder).unwrap();
    folder
}
fn folder_row(world: &ApplicationWorld) -> String {
    world
        .frame
        .runs
        .iter()
        .find(|run| run.text == format!("{CHILD_FOLDER}/"))
        .expect("expected the child folder's row")
        .text
        .clone()
}
#[given("a saved part with a preview is selected in a child folder")]
fn part_in_child(world: &mut ApplicationWorld) {
    cancel_dialog(world);
    let folder = child_folder(world);
    saved_part(world, &folder);
    open_dialog(world);
    look_in(world, &folder);
    select_row(world, PART_FILE);
}
#[when("I choose Up to the parent folder")]
#[when("the user chooses Up to the parent folder")]
#[when("I navigate up to its parent folder")]
#[when("the user navigates up to its parent folder")]
fn choose_up(world: &mut ApplicationWorld) {
    world.click_text("Up");
    settle(world);
}
#[then("the parent folder contents are displayed")]
#[then("the parent folder contents are displayed again")]
fn parent_shown(world: &mut ApplicationWorld) {
    folder_row(world);
}
#[then("the child folder is highlighted in the list")]
#[then("the child folder is highlighted")]
fn child_highlighted(world: &mut ApplicationWorld) {
    let row = folder_row(world);
    assert!(row_lit(world, &row), "expected the folder's accent");
}
#[then("the file name field retains the part name")]
fn name_retained(world: &mut ApplicationWorld) {
    let shown = file_name_shown(world);
    assert!(shown.contains(PART_FILE), "file name {shown:?}");
}
#[then("the preview area shows a placeholder for the selected folder")]
fn folder_placeholder(world: &mut ApplicationWorld) {
    expect_placeholder(world);
}
#[when("I choose Forward")]
#[when("the user chooses Forward")]
fn choose_forward(world: &mut ApplicationWorld) {
    world.click_text("Forward");
    settle(world);
}
#[then("the child folder contents are displayed again")]
fn child_shown(world: &mut ApplicationWorld) {
    world.expect_text(PART_FILE);
}
#[then("the file name field and preview display the previously selected part")]
fn previous_part(world: &mut ApplicationWorld) {
    expect_name_and_preview(world, PART_FILE);
}
#[given("the file name field retains the name of a part in a child folder")]
fn retained_name(world: &mut ApplicationWorld) {
    part_in_child(world);
    choose_up(world);
    // What a later step counts tabs against.
    world.parts = world.state().documents.iter().map(|row| row.id).collect();
}
#[given("the parent folder displays that child folder")]
fn parent_lists_child(world: &mut ApplicationWorld) {
    folder_row(world);
}
#[when("I select the child folder in the file list")]
#[when("the user selects the child folder in the file list")]
fn select_child(world: &mut ApplicationWorld) {
    let row = folder_row(world);
    select_row(world, &row);
}
#[then("the dialog remains open and displays the child folder contents")]
fn in_child(world: &mut ApplicationWorld) {
    world.expect_text("Look in");
    world.expect_text(PART_FILE);
}
#[then("the file name field still contains the part name")]
fn name_kept(world: &mut ApplicationWorld) {
    name_retained(world);
}
#[then("the part preview is displayed")]
fn part_preview(world: &mut ApplicationWorld) {
    world.expect_no_text("No preview image");
}
// Reopening
#[given("a drawing was opened from a child folder in the MCAD Software")]
fn opened_from_child(world: &mut ApplicationWorld) {
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
    let name = world.state().document;
    assert!(name.is_some(), "expected a document active");
    world.expect_no_text("Look in");
}
#[given("the Open dialog is showing the same child folder")]
fn showing_child(world: &mut ApplicationWorld) {
    open_dialog(world);
    world.expect_text(DRAWING_FILE);
}
#[when("I reopen the Open dialog in the same session")]
#[when("the user reopens the Open dialog in the same session")]
fn reopen(world: &mut ApplicationWorld) {
    open_dialog(world);
}
#[then("it displays the original child folder")]
fn original_folder(world: &mut ApplicationWorld) {
    world.expect_text(DRAWING_FILE);
}
#[then("the file name field identifies the previously opened drawing")]
fn name_previous(world: &mut ApplicationWorld) {
    let shown = file_name_shown(world);
    assert!(shown.contains(DRAWING_FILE), "file name {shown:?}");
}
#[then("the drawing preview is displayed")]
fn drawing_preview(world: &mut ApplicationWorld) {
    world.expect_no_text("No preview image");
}
// Maximize
#[given("the Open dialog is open in the MCAD Software as a restored window")]
fn restored_window(world: &mut ApplicationWorld) {
    dialog_open(world);
}
fn dialog_geometry() -> ([f32; 2], [f32; 2]) {
    super::super::diagnostics::window_geometry("Open").expect("expected the Open dialog")
}
#[when("I maximize the Open dialog")]
#[when("the user maximizes the Open dialog")]
fn maximize(world: &mut ApplicationWorld) {
    world.observed_row_gap = Some(dialog_geometry().1[0]);
    world.click_text("Maximize");
    settle(world);
    world.pump(Vec::new());
}
#[then("the dialog expands to fill the available screen")]
fn fills_screen(_: &mut ApplicationWorld) {
    let (position, size) = dialog_geometry();
    assert!(
        position == [0., 0.] && size[0] >= 1400.,
        "dialog {position:?} {size:?}"
    );
}
#[then("the file list, navigation pane, preview and bottom actions remain visible")]
fn all_visible(world: &mut ApplicationWorld) {
    for row in ["Libraries", "Name", "Open", "Cancel"] {
        world.expect_text(row);
    }
    world.expect_no_text("No preview image");
}
#[when("I restore the Open dialog")]
#[when("the user restores the Open dialog")]
fn restore(world: &mut ApplicationWorld) {
    world.click_text("Restore");
    settle(world);
    world.pump(Vec::new());
}
#[then("it returns to its previous window size and position")]
fn restored_size(world: &mut ApplicationWorld) {
    let earlier = world.observed_row_gap.expect("expected the earlier width");
    assert_eq!(dialog_geometry().1[0], earlier, "width");
}
// File types
#[when("I open the file-type filter list")]
#[when("the user opens the file-type filter list")]
fn open_type_list(world: &mut ApplicationWorld) {
    let label = world.run("Files of type").clone();
    world.click([label.maximum[0] as i32 + 60, label.center()[1] as i32]);
    settle(world);
}
#[then("the list offers a combined native-document filter")]
fn combined_filter(world: &mut ApplicationWorld) {
    world.expect_text("All Documents");
}
#[then("it includes separate filters for assemblies, drawings, parts and presentations")]
fn separate_filters(world: &mut ApplicationWorld) {
    for row in [
        "Assembly Files",
        "Drawing Files",
        "Part Files",
        "Presentation Files",
    ] {
        assert!(holds_text(world, row), "expected {row} in the list");
    }
}
#[then("additional file-format filters are visible")]
fn other_filters(world: &mut ApplicationWorld) {
    assert!(
        holds_text(world, "STEP Files"),
        "expected an exchange format in the list"
    );
}
#[when("I dismiss the list without choosing a different filter")]
#[when("the user dismisses the list without choosing a different filter")]
fn dismiss_types(world: &mut ApplicationWorld) {
    world.escape();
    settle(world);
}
#[then("the previous file-type filter remains displayed")]
fn filter_kept(world: &mut ApplicationWorld) {
    world.expect_text("All Documents");
}
#[then("the selected part name and preview remain displayed")]
#[then("the part name and preview remain displayed")]
#[then("the part name and preview remain displayed behind the menu")]
fn part_name_kept(world: &mut ApplicationWorld) {
    expect_name_and_preview(world, PART_FILE);
}
// The display menu
#[given("files are displayed in Details view")]
fn in_details(world: &mut ApplicationWorld) {
    world.expect_text("Date modified");
}
#[given("a part name and preview are displayed")]
fn name_and_preview(world: &mut ApplicationWorld) {
    saved_part_selected_here(world);
}
fn saved_part_selected_here(world: &mut ApplicationWorld) {
    cancel_dialog(world);
    let folder = models_folder(world);
    saved_part(world, &folder);
    open_dialog(world);
    look_in(world, &folder);
    select_row(world, PART_FILE);
}
#[when("I open the file display menu")]
#[when("the user opens the file display menu")]
fn open_display(world: &mut ApplicationWorld) {
    world.click_text("Display");
    settle(world);
}
#[then("the menu offers extra large, large, medium and small icon sizes")]
fn icon_rows(world: &mut ApplicationWorld) {
    for row in [
        "Extra Large Icons",
        "Large Icons",
        "Medium Icons",
        "Small Icons",
    ] {
        world.expect_text(row);
    }
}
#[then("it offers List, Details, Tiles and Content views")]
fn view_rows(world: &mut ApplicationWorld) {
    for row in ["List", "Tiles", "Content"] {
        world.expect_text(row);
    }
}
#[then("Details is indicated as the current view")]
fn details_marked(world: &mut ApplicationWorld) {
    assert!(world.checked("Details"), "expected Details marked");
}
#[when("I dismiss the menu without choosing another view")]
#[when("the user dismisses the menu without choosing another view")]
fn dismiss_display(world: &mut ApplicationWorld) {
    world.escape();
    settle(world);
}
#[then("the file list remains in Details view")]
fn stays_details(world: &mut ApplicationWorld) {
    world.expect_text("Date modified");
}
