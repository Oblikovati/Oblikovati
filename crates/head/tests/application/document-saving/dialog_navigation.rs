// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[given("the Save As dialog is open for a new empty part in the MCAD Software")]
fn dialog_open_on_empty_part(world: &mut ApplicationWorld) {
    world.add_part();
    world.workspace_color = Some(world.pixel_at(VIEWPORT_CENTRE));
    world.press_save_shortcut();
    world.expect_text(SAVE_IN);
}
#[when("I choose Active Window as the preview source")]
fn choose_active_window(world: &mut ApplicationWorld) {
    world.click_text(WINDOW);
}
#[then("a capture control appears beside Active Window")]
fn capture_beside(world: &mut ApplicationWorld) {
    let option = world.run(WINDOW).clone();
    let capture = world.run("Capture").clone();
    assert!(
        capture.minimum[0] > option.maximum[0]
            && (capture.center()[1] - option.center()[1]).abs() < 4.,
        "capture {capture:?}: expected a control beside {option:?}"
    );
}
#[when("I activate the capture control")]
fn activate_capture(world: &mut ApplicationWorld) {
    world.click_text("Capture");
    world.pump(Vec::new());
}
#[then("the preview area displays a dark image matching the empty workspace")]
fn dark_preview(world: &mut ApplicationWorld) {
    world.expect_no_text(NO_PREVIEW);
    // The preview area sits directly above its checkbox.
    let checkbox = world.run("Save Preview Picture").clone();
    let at = [
        checkbox.center()[0] as usize,
        checkbox.minimum[1] as usize - 40,
    ];
    let preview = world.pixel_at(at);
    let workspace = world.workspace_color.expect("expected a workspace colour");
    let difference = preview[..3]
        .iter()
        .zip(&workspace[..3])
        .map(|(left, right)| left.abs_diff(*right))
        .max()
        .unwrap();
    assert!(
        difference < 40,
        "preview {preview:?}, workspace {workspace:?}: expected the same dark colours"
    );
}
#[given("the current folder contains an existing part file")]
fn folder_with_part(world: &mut ApplicationWorld) {
    let folder = world.scratch_folder();
    std::fs::write(folder.join(EXISTING_PART), b"not opened by this scenario").unwrap();
    world.enter_in_field(SAVE_IN, folder.to_str().unwrap());
    world.expect_text(EXISTING_PART);
}
#[when("I select that part file in the file list")]
fn select_existing(world: &mut ApplicationWorld) {
    world.click_text(EXISTING_PART);
}
#[then("the selected file row is highlighted")]
fn row_highlighted(world: &mut ApplicationWorld) {
    // The field below repeats the name after a click, so the row is the higher of the two.
    let row = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == EXISTING_PART.replace(' ', ""))
        .min_by(|left, right| left.minimum[1].total_cmp(&right.minimum[1]))
        .expect("expected the listed file row")
        .clone();
    let y = row.center()[1] as usize;
    let beside = world.pixel_at([row.maximum[0] as usize + 10, y]);
    let empty_row = world.pixel_at([row.maximum[0] as usize + 10, y + 60]);
    assert_ne!(
        beside, empty_row,
        "row {row:?}: expected a highlight beyond its text"
    );
}
#[then("the file name field displays the selected file's name")]
fn name_follows(world: &mut ApplicationWorld) {
    let rows = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == EXISTING_PART.replace(' ', ""))
        .count();
    assert_eq!(rows, 2, "expected the name in the row and the field");
}
#[then("the status area reports one selected item")]
fn one_selected(world: &mut ApplicationWorld) {
    world.expect_text("1 item selected");
}
#[given("the initial folder contains a subfolder")]
fn initial_subfolder(world: &mut ApplicationWorld) {
    // The dialog starts in the crate's folder, whose `src` is always present.
    world.expect_text(SUBFOLDER);
    world.initial_breadcrumb = Some(world.breadcrumb_text());
}
#[given("the dialog displays a proposed file name")]
fn proposal_shown(world: &mut ApplicationWorld) {
    let id = world.active_id();
    let name = format!("{}.opd", world.document_name(id));
    world.expect_text(&name);
    world.proposed_name = Some(name);
}
fn open_subfolder_row(world: &mut ApplicationWorld) {
    world.click_text(SUBFOLDER);
    world.expect_text(NESTED_FOLDER);
}
#[when("I open the subfolder")]
fn open_subfolder(world: &mut ApplicationWorld) {
    open_subfolder_row(world);
}
#[then("the location breadcrumb and file list show the subfolder")]
fn subfolder_shown(world: &mut ApplicationWorld) {
    let crumb = world.breadcrumb_text();
    assert!(
        crumb.ends_with(">src"),
        "breadcrumb {crumb:?}: expected the subfolder last"
    );
    world.expect_text(NESTED_FOLDER);
}
#[then("the proposed file name remains in the file name field")]
fn proposal_remains(world: &mut ApplicationWorld) {
    let name = world
        .proposed_name
        .clone()
        .expect("expected a proposed name");
    world.expect_text(&name);
}
#[when("I choose Back")]
fn choose_back(world: &mut ApplicationWorld) {
    world.click_text("Back");
}
#[then("the initial folder and its file list are displayed")]
fn initial_folder(world: &mut ApplicationWorld) {
    let crumb = world.breadcrumb_text();
    assert_eq!(Some(crumb), world.initial_breadcrumb);
    world.expect_text(SUBFOLDER);
}
#[given("I have navigated from the initial folder into a subfolder")]
fn navigated(world: &mut ApplicationWorld) {
    world.initial_breadcrumb = Some(world.breadcrumb_text());
    let id = world.active_id();
    world.proposed_name = Some(format!("{}.opd", world.document_name(id)));
    open_subfolder_row(world);
}
#[when("I press Control+S again for the same part")]
fn save_again(world: &mut ApplicationWorld) {
    world.press_save_shortcut();
    world.expect_text(SAVE_IN);
}
#[then("the Save As dialog opens in the initial folder")]
fn reopened_initial(world: &mut ApplicationWorld) {
    let crumb = world.breadcrumb_text();
    assert_eq!(Some(crumb), world.initial_breadcrumb);
}
#[then("the original proposed file name is selected")]
fn original_selected(world: &mut ApplicationWorld) {
    let name = world
        .proposed_name
        .clone()
        .expect("expected a proposed name");
    assert!(
        world.text_is_selected(&name),
        "file name {name:?}: expected a selected proposal"
    );
}
#[then("Save Preview Picture is selected")]
fn preview_still_selected(world: &mut ApplicationWorld) {
    world.expect_text(ICONIC);
}
#[then("the preview source choices are displayed")]
fn sources_displayed(world: &mut ApplicationWorld) {
    world.expect_text(ICONIC);
    world.expect_text(WINDOW);
}
#[given("Active Component Iso View on Save is selected")]
#[then("Active Component Iso View on Save is selected")]
fn iso_selected(world: &mut ApplicationWorld) {
    assert!(world.radio_selected(ICONIC), "expected {ICONIC:?} selected");
}
#[then("Active Window on Save is not selected")]
fn window_not_selected(world: &mut ApplicationWorld) {
    assert!(!world.radio_selected(WINDOW), "expected {WINDOW:?} cleared");
}
#[given("the navigation pane shows libraries and recent folders")]
#[then("the navigation pane displays libraries and recent folders")]
#[then("the navigation pane shows libraries and recent folders")]
fn pane_recent(world: &mut ApplicationWorld) {
    world.expect_text("Libraries");
    world.expect_text("Recent Folders");
    world.expect_no_text("System Locations");
}
fn proposed_name_of(world: &mut ApplicationWorld) -> String {
    let id = world.active_id();
    format!("{}.opd", world.document_name(id))
}
#[given("the dialog displays a current folder and proposed file name")]
#[given("a proposed file name is displayed")]
fn current_and_proposed(world: &mut ApplicationWorld) {
    world.proposed_name = Some(proposed_name_of(world));
    world.initial_breadcrumb = Some(world.breadcrumb_text());
    let name = world.proposed_name.clone().unwrap();
    world.expect_text(&name);
}
#[when("I switch to system-location navigation")]
fn switch_system(world: &mut ApplicationWorld) {
    world.click_text("Locations");
}
#[then("the navigation pane displays system locations and drives")]
fn pane_system(world: &mut ApplicationWorld) {
    world.expect_text("System Locations");
    world.expect_text("Computer");
}
#[then("the current folder and proposed file name remain displayed")]
fn folder_and_name_remain(world: &mut ApplicationWorld) {
    assert_eq!(Some(world.breadcrumb_text()), world.initial_breadcrumb);
    let name = world
        .proposed_name
        .clone()
        .expect("expected a proposed name");
    world.expect_text(&name);
}
#[then("the preview settings remain displayed")]
fn preview_settings_remain(world: &mut ApplicationWorld) {
    world.expect_text("Save Preview Picture");
    world.expect_text(ICONIC);
}
#[when("I switch back to recent-folder navigation")]
fn switch_recent(world: &mut ApplicationWorld) {
    world.click_text("Recent");
}
#[when("I open Save As again for the same part")]
fn save_as_again(world: &mut ApplicationWorld) {
    save_again(world);
}
#[then("the proposed file name is selected")]
fn proposed_is_selected(world: &mut ApplicationWorld) {
    let name = proposed_name_of(world);
    assert!(
        world.text_is_selected(&name),
        "file name {name:?}: expected a selected proposal"
    );
}
#[given("the current folder contains part files and subfolders")]
fn folder_with_entries(world: &mut ApplicationWorld) {
    let folder = world.scratch_folder();
    std::fs::write(folder.join(EXISTING_PART), b"listed, never opened").unwrap();
    std::fs::create_dir(folder.join("archive")).unwrap();
    world.enter_in_field(SAVE_IN, folder.to_str().unwrap());
    world.expect_text(EXISTING_PART);
}
#[given("the file list is displayed in Details view")]
#[then("the file list displays rows with name, modification date, type and size columns")]
fn details_columns(world: &mut ApplicationWorld) {
    for heading in ["Name", "Date modified", "Type", "Size"] {
        world.expect_text(heading);
    }
}
#[when("I activate the file-list view control")]
fn activate_view(world: &mut ApplicationWorld) {
    world.click_text("Tiles");
}
#[then("the file list displays thumbnail tiles with names and file information")]
fn tiles_shown(world: &mut ApplicationWorld) {
    world.expect_no_text("Date modified");
    world.expect_text(EXISTING_PART);
    world.expect_text("Folder 4096 B");
}
#[then("the proposed file name and preview settings remain displayed")]
fn name_and_preview_remain(world: &mut ApplicationWorld) {
    let name = world
        .proposed_name
        .clone()
        .expect("expected a proposed name");
    world.expect_text(&name);
    preview_settings_remain(world);
}
#[when("I choose the Details view shortcut")]
fn choose_details_shortcut(world: &mut ApplicationWorld) {
    world.click_text("Details");
}
#[then("the proposed file name remains displayed")]
fn name_remains(world: &mut ApplicationWorld) {
    let name = world
        .proposed_name
        .clone()
        .expect("expected a proposed name");
    world.expect_text(&name);
}
#[when("I select the existing part file")]
fn select_existing_part(world: &mut ApplicationWorld) {
    world.click_text(EXISTING_PART);
}
#[when("I choose Save")]
fn choose_save(world: &mut ApplicationWorld) {
    world.click_text("Save");
    world.pump(Vec::new());
}
#[then("a confirmation identifies the existing file and asks whether to replace it")]
fn replace_asked(world: &mut ApplicationWorld) {
    world.expect_text("already exists");
    world.expect_text(EXISTING_PART);
    world.expect_text("replace it");
}
#[then("the confirmation offers Yes and No")]
fn replace_choices(world: &mut ApplicationWorld) {
    world.expect_text("Yes");
    world.expect_text("No");
}
#[when("I choose No")]
fn choose_no(world: &mut ApplicationWorld) {
    world.click_text("No");
    world.pump(Vec::new());
}
#[then("the confirmation closes")]
fn replace_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("already exists");
}
#[then("Save As remains open with the existing file name displayed")]
fn save_as_remains(world: &mut ApplicationWorld) {
    world.expect_text(SAVE_IN);
    world.expect_text(EXISTING_PART);
    let folder = world.scratch_folder();
    let bytes = std::fs::read(folder.join(EXISTING_PART)).unwrap();
    assert_eq!(
        bytes, b"not opened by this scenario",
        "No: expected the file untouched"
    );
}
#[then("the new unsaved part remains active")]
fn new_unsaved_remains(world: &mut ApplicationWorld) {
    unsaved_remains(world);
}
#[given("a saved part is active in the MCAD Software")]
fn saved_part_active(world: &mut ApplicationWorld) {
    let id = world.add_part();
    world.saved_path = Some(world.save_part_to_file(id, "saved part.opd"));
}
#[when("I open the application menu")]
fn open_app_menu(world: &mut ApplicationWorld) {
    world.click_text("File");
    world.pump(Vec::new());
}
#[when("I expand Save As")]
fn expand_save_as(world: &mut ApplicationWorld) {
    let center = world.run("Save As").center().map(|value| value as i32);
    world.hover(center);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the menu offers Save As and Save Copy As")]
fn offers_save_as(world: &mut ApplicationWorld) {
    world.expect_text("Save As...");
    world.expect_text("Save Copy As");
}
#[then("it offers Save Copy As Template and Pack and Go")]
fn offers_template(world: &mut ApplicationWorld) {
    world.expect_text("Save Copy As Template");
    world.expect_text("Pack and Go");
}
#[then("the Save Copy As description says the original file remains open")]
fn copy_description(world: &mut ApplicationWorld) {
    world.expect_text("the original file remains open");
}
