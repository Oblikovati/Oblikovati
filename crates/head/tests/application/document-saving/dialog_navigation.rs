// SPDX-License-Identifier: GPL-2.0-only
use super::native_fixtures::{
    EXISTING_NAME, SOLID_NAME, native_save_as_folder, published_save_as_parts,
};
use super::*;

#[given("the user presses Control+S to open Save As")]
fn save_with_workspace_preview_baseline(world: &mut ApplicationWorld) {
    world.workspace_color = Some(world.pixel_at(VIEWPORT_CENTRE));
    world.press_save_shortcut();
    world.expect_text(SAVE_IN);
}

#[given("the Save As dialog is open for a new empty part in the MCAD Software")]
fn dialog_open_on_empty_part(world: &mut ApplicationWorld) {
    if !world.shows(SAVE_IN) {
        world.add_part();
        save_with_workspace_preview_baseline(world);
    }
    world.expect_text(SAVE_IN);
}
#[when("I choose Active Window as the preview source")]
#[when("the user chooses Active Window as the preview source")]
fn choose_active_window(world: &mut ApplicationWorld) {
    choose_window(world);
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
#[when("the user activates the capture control")]
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

fn folder_pixel(world: &mut ApplicationWorld) -> [u8; 4] {
    let folder = world.run(selectable_folder()).clone();
    world.pixel_at([folder.maximum[0] as usize + 10, folder.center()[1] as usize])
}
#[when("I select that part file in the file list")]
#[when("the user selects that part file in the file list")]
#[when("the user selects `ExistingPart.opd` in the file list")]
fn select_existing(world: &mut ApplicationWorld) {
    world.click_text(existing_part());
    move_pointer_off_dialog(world);
}
#[then("the selected file row is highlighted")]
fn row_highlighted(world: &mut ApplicationWorld) {
    // The field below repeats the name after a click, so the row is the higher of the two.
    let row = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == existing_part().replace(' ', ""))
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
        .filter(|run| run.text == existing_part().replace(' ', ""))
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
pub(super) fn observe_native_save_folder(world: &mut ApplicationWorld) {
    assert!(world.breadcrumb_text().ends_with("SaveAsNativeFolder"));
    world.expect_text("Subfolder/");
    world.initial_breadcrumb = Some(world.breadcrumb_text());
    world.observed_counts = folder_pixel(world).into_iter().map(usize::from).collect();
}

fn existing_part() -> &'static str {
    native_save_as_folder().map_or(EXISTING_PART, |_| EXISTING_NAME)
}

fn navigable_folder() -> &'static str {
    native_save_as_folder().map_or(SUBFOLDER, |_| "Subfolder/")
}

fn selectable_folder() -> &'static str {
    native_save_as_folder().map_or("archive/", |_| "Subfolder/")
}
#[given("the dialog displays a proposed file name")]
fn proposal_shown(world: &mut ApplicationWorld) {
    let id = world.active_id();
    let name = format!("{}.opd", world.document_name(id));
    world.expect_text(&name);
    world.proposed_name = Some(name);
}
fn open_subfolder_row(world: &mut ApplicationWorld) {
    let at = world
        .run(navigable_folder())
        .center()
        .map(|value| value as i32);
    world.double_click(at);
    subfolder_shown(world);
}
#[when("I open the subfolder")]
#[when("the user opens the subfolder")]
fn open_subfolder(world: &mut ApplicationWorld) {
    open_subfolder_row(world);
}
#[then("the location breadcrumb and file list show the subfolder")]
fn subfolder_shown(world: &mut ApplicationWorld) {
    let crumb = world.breadcrumb_text();
    let child = navigable_folder().trim_end_matches('/');
    assert!(
        crumb.ends_with(&format!(">{child}")),
        "breadcrumb {crumb:?}: expected the subfolder last"
    );
    if native_save_as_folder().is_some() {
        world.expect_no_text(EXISTING_NAME);
        world.expect_no_text(SOLID_NAME);
    } else {
        world.expect_text(NESTED_FOLDER);
    }
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
#[when("the user chooses Back")]
fn choose_back(world: &mut ApplicationWorld) {
    world.click_text("Back");
}
#[then("the initial folder and its file list are displayed")]
fn initial_folder(world: &mut ApplicationWorld) {
    let crumb = world.breadcrumb_text();
    assert_eq!(Some(crumb), world.initial_breadcrumb);
    world.expect_text(navigable_folder());
    if native_save_as_folder().is_some() {
        world.expect_text(EXISTING_NAME);
        world.expect_text(SOLID_NAME);
    }
}
#[given("I have navigated from the initial folder into a subfolder")]
#[given("the user has navigated from the initial folder into a subfolder")]
fn navigated(world: &mut ApplicationWorld) {
    world.initial_breadcrumb = Some(world.breadcrumb_text());
    let id = world.active_id();
    world.proposed_name = Some(format!("{}.opd", world.document_name(id)));
    open_subfolder_row(world);
}
#[when("I press Control+S again for the same part")]
#[when("the user presses Control+S again for the same part")]
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
    assert!(
        world.checked("Save Preview Picture"),
        "expected preview checked"
    );
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
#[when("the user switches to system-location navigation")]
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
#[when("the user switches back to recent-folder navigation")]
fn switch_recent(world: &mut ApplicationWorld) {
    world.click_text("Recent");
}
#[when("I open Save As again for the same part")]
#[when("the user opens Save As again for the same part")]
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
#[given("the current folder contains a part file and a subfolder")]
fn folder_with_entries(world: &mut ApplicationWorld) {
    let folder = world.scratch_folder();
    std::fs::write(folder.join(EXISTING_PART), b"listed, never opened").unwrap();
    std::fs::create_dir(folder.join("archive")).unwrap();
    world.enter_in_field(SAVE_IN, folder.to_str().unwrap());
    world.expect_text(EXISTING_PART);
    world.observed_counts = folder_pixel(world).into_iter().map(usize::from).collect();
}

#[when("the user selects the part file in the file list")]
fn choose_listed_part(world: &mut ApplicationWorld) {
    select_existing(world);
}
#[then("the file is highlighted and its name appears in the file name field")]
fn listed_part_selected(world: &mut ApplicationWorld) {
    row_highlighted(world);
    name_follows(world);
}
#[when("the user selects the subfolder with a single click")]
fn select_folder_once(world: &mut ApplicationWorld) {
    world.click_text(selectable_folder());
    move_pointer_off_dialog(world);
}
fn move_pointer_off_dialog(world: &mut ApplicationWorld) {
    world.pump(vec![oblikovati_viewport::ViewportEvent::Input(
        oblikovati_viewport::ViewportInput::Motion {
            position: [20, 850],
            modifiers: Default::default(),
        },
    )]);
}
#[then("the subfolder is highlighted without opening")]
fn folder_selected_without_opening(world: &mut ApplicationWorld) {
    world.expect_text(existing_part());
    assert!(world.shows(selectable_folder()), "expected the folder row");
    let baseline: Vec<usize> = folder_pixel(world).into_iter().map(usize::from).collect();
    assert_ne!(baseline, world.observed_counts, "expected selected folder");
}
#[then("the file name field still displays the selected part file name")]
#[then("the file name field still displays the part file name")]
fn selected_name_retained(world: &mut ApplicationWorld) {
    world.expect_text(existing_part());
    let shown = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == existing_part().replace(' ', ""))
        .count();
    assert!(shown >= 2, "expected file name in the list and field");
}
#[when("the user clicks empty space in the file list")]
fn clear_list_selection(world: &mut ApplicationWorld) {
    let row = world.run(selectable_folder()).clone();
    world.click([row.center()[0] as i32 + 80, row.center()[1] as i32 + 100]);
}
#[then("no file or folder is highlighted")]
fn no_list_selection(world: &mut ApplicationWorld) {
    world.expect_no_text("1 item selected");
    let baseline: Vec<usize> = folder_pixel(world).into_iter().map(usize::from).collect();
    assert_eq!(
        baseline, world.observed_counts,
        "expected folder unselected"
    );
}
#[given("the file list is displayed in Details view")]
#[then("the file list displays rows with name, modification date, type and size columns")]
fn details_columns(world: &mut ApplicationWorld) {
    for heading in ["Name", "Date modified", "Type", "Size"] {
        world.expect_text(heading);
    }
}
#[when("I activate the file-list view control")]
#[when("the user activates the file-list view control")]
fn activate_view(world: &mut ApplicationWorld) {
    world.click_text("Tiles");
}
#[then("the file list displays thumbnail tiles with names and file information")]
fn tiles_shown(world: &mut ApplicationWorld) {
    world.expect_no_text("Date modified");
    world.expect_text(existing_part());
    if let Some(folder) = native_save_as_folder() {
        for name in [EXISTING_NAME, SOLID_NAME] {
            world.expect_text(name);
            let size = folder.join(name).metadata().unwrap().len();
            world.expect_text(&format!("Part {size} B"));
        }
    } else {
        world.expect_text("Folder 4096 B");
    }
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
#[when("the user chooses the Details view shortcut")]
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
#[when("the user selects the existing part file")]
#[when("the user selects `ExistingPart.opd`")]
fn select_existing_part(world: &mut ApplicationWorld) {
    select_existing(world);
}
#[when("I choose Save")]
#[when("the user chooses Save")]
fn choose_save(world: &mut ApplicationWorld) {
    world.click_text("Save");
    world.pump(Vec::new());
}
#[then("a confirmation identifies the existing file and asks whether to replace it")]
fn replace_asked(world: &mut ApplicationWorld) {
    world.expect_text("already exists");
    world.expect_text(existing_part());
    world.expect_text("replace it");
}
#[then("the confirmation offers Yes and No")]
fn replace_choices(world: &mut ApplicationWorld) {
    for label in ["Yes", "No"] {
        world.expect_text(label);
        assert!(
            world.run_is_enabled(label),
            "{label}: expected an enabled choice"
        );
    }
}
#[when("I choose No")]
#[when("the user chooses No")]
fn choose_no(world: &mut ApplicationWorld) {
    world.click_text("No");
    world.pump(Vec::new());
}
#[then("the confirmation closes")]
fn replace_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("already exists");
    assert!(super::super::diagnostics::window_geometry("Reset Ribbon").is_none());
}
#[then("Save As remains open with the existing file name displayed")]
fn save_as_remains(world: &mut ApplicationWorld) {
    world.expect_text(SAVE_IN);
    world.expect_text(existing_part());
    if let Some(folder) = native_save_as_folder() {
        for name in [EXISTING_NAME, SOLID_NAME] {
            let actual = std::fs::read(folder.join(name)).unwrap();
            let expected = std::fs::read(published_save_as_parts().join(name)).unwrap();
            assert_eq!(
                actual, expected,
                "expected {name} to survive declining replacement"
            );
        }
        return;
    }
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

#[path = "save_as_commands.rs"]
mod save_as_commands;
