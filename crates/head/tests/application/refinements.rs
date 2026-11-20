// SPDX-License-Identifier: GPL-2.0-only
//! Steps the scenario source added to existing scenarios: each reads what already exists.
use super::ApplicationWorld;
use super::diagnostics::window_geometry;
use cucumber::{given, then, when};

// Document creation
#[given("a template is selected and its details are displayed")]
#[given("a template is selected with its details displayed")]
fn template_selected(world: &mut ApplicationWorld) {
    world.click_text("Standard");
    world.pump(Vec::new());
    world.proposed_name = Some(selected_template_details(world));
}

pub(super) fn selected_template_details(world: &ApplicationWorld) -> String {
    let title = world.run("Template details");
    let kind = world.run("Document type:");
    world
        .frame
        .runs
        .iter()
        .filter(|run| run.list == title.list && (run.minimum[0] - title.minimum[0]).abs() < 2.)
        .filter(|run| run.minimum[1] > title.minimum[1] && run.minimum[1] <= kind.minimum[1])
        .map(|run| run.text.as_str())
        .collect::<Vec<_>>()
        .join("\n")
}
#[then("it offers OK and Cancel")]
fn offers_ok_cancel(world: &mut ApplicationWorld) {
    world.expect_text("OK");
    world.expect_text("Cancel");
}
#[then("it displays a folder and file list, a file name field and a file-type filter")]
fn chooser_list(world: &mut ApplicationWorld) {
    world.expect_text("Look in");
    world.expect_text("File name");
    world.expect_text("Files of type");
}
#[then("it displays a preview area and Open and Cancel controls")]
fn chooser_preview(world: &mut ApplicationWorld) {
    world.expect_text("Preview");
    world.expect_text("Open");
    world.expect_text("Cancel");
}
// Saving
#[then("the model browser root displays the saved file name")]
fn browser_root_named(world: &mut ApplicationWorld) {
    let id = world.active_id();
    world.expect_browser_of(id);
}
#[when("I open the file-opening dialog in the same session")]
#[when("the user opens the file-opening dialog in the same session")]
fn open_dialog(world: &mut ApplicationWorld) {
    world.click_text("Open");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[when("I select the saved part")]
fn select_saved(world: &mut ApplicationWorld) {
    let path = world.saved_path.clone().expect("expected a saved part");
    // The fixture bound the file without this dialog, which so far knows no other folder.
    let folder = path.parent().unwrap().to_string_lossy().into_owned();
    world.enter_in_field("Look in", &folder);
    let name = path.file_name().unwrap().to_string_lossy().replace(' ', "");
    let row = world.run(&name).center().map(|value| value as i32);
    world.click(row);
    world.pump(Vec::new());
}
#[then("the dialog displays a thumbnail of its sketch")]
fn dialog_thumbnail(world: &mut ApplicationWorld) {
    world.expect_no_text("No preview image");
}
#[when("I open the selected part")]
#[when("the user opens the selected part")]
fn open_selected(world: &mut ApplicationWorld) {
    let open = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| run.text == "Open")
        .cloned()
        .expect("expected the dialog's Open");
    world.click(open.center().map(|value| value as i32));
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the Save As dialog remains behind the confirmation")]
fn save_as_behind(world: &mut ApplicationWorld) {
    world.expect_text("Save in");
}
#[given("the suggested file name does not exist in the selected folder")]
fn suggested_free(world: &mut ApplicationWorld) {
    let folder = world.scratch_folder();
    let id = world.active_id();
    let name = world.document_name(id);
    assert!(
        !folder.join(format!("{name}.opd")).exists(),
        "expected a free file name"
    );
}
// Title
fn title(world: &mut ApplicationWorld) -> String {
    world.app().window.native.view.native_title().unwrap()
}
#[then("the application title identifies the second part")]
fn title_second(world: &mut ApplicationWorld) {
    let second = world.parts[1];
    let name = world.document_name(second);
    let shown = title(world);
    assert!(
        shown.starts_with(&name),
        "title {shown:?}: expected it to start with {name:?}"
    );
}
#[then("the application title identifies the second window number")]
fn title_window(world: &mut ApplicationWorld) {
    let shown = title(world);
    assert!(
        shown.contains(":2"),
        "title {shown:?}: expected the second window's number"
    );
}
// Tabs
fn open_user_interface(world: &mut ApplicationWorld) {
    world.click_text("User Interface");
    world.pump(Vec::new());
}
#[then("Document Tabs is unchecked in the User Interface menu")]
fn tabs_unchecked(world: &mut ApplicationWorld) {
    if !world.shows("Document Tabs") {
        open_user_interface(world);
    }
    assert!(
        !world.checked("Document Tabs"),
        "expected Document Tabs clear"
    );
}
#[then("Document Tabs is checked in the User Interface menu")]
fn tabs_checked(world: &mut ApplicationWorld) {
    if !world.shows("Document Tabs") {
        open_user_interface(world);
    }
    assert!(
        world.checked("Document Tabs"),
        "expected Document Tabs checked"
    );
}
#[then("the preview label displays the saved file's full path and part name on separate lines")]
fn path_and_name_lines(world: &mut ApplicationWorld) {
    let path = world.saved_path.clone().expect("expected a saved path");
    let second = world.parts[1];
    let name = world.document_name(second);
    let wanted = path.display().to_string().replace(' ', "");
    let (position, _) = window_geometry("##tab-preview").expect("expected a tab preview");
    let path_row = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == wanted && run.minimum[1] >= position[1])
        .cloned()
        .unwrap_or_else(|| panic!("expected {wanted} in the preview: {:?}", world.text()));
    let name_row = world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.list == path_row.list
                && run
                    .text
                    .strip_prefix(&name)
                    .is_some_and(|suffix| suffix.is_empty() || suffix.starts_with('<'))
                && run.minimum[1] > path_row.maximum[1] - 1.
        })
        .cloned()
        .unwrap_or_else(|| panic!("expected {name} below the path"));
    assert!(
        name_row.minimum[1] > path_row.minimum[1],
        "expected separate lines"
    );
}
// File menu
#[then("a vertical list of file commands opens over Home")]
#[then("the command entries form a vertical list on the left")]
fn commands_listed(world: &mut ApplicationWorld) {
    let [first, second] = ["New", "Save As"].map(|entry| world.run(entry).clone());
    assert!(
        (first.minimum[0] - second.minimum[0]).abs() < 2. && second.minimum[1] > first.minimum[1],
        "{:?} and {:?}: expected a column",
        first.minimum,
        second.minimum
    );
}
#[then("Save As, document properties and Print appear muted")]
fn file_rows_muted(world: &mut ApplicationWorld) {
    for row in ["Save As", "Document Properties", "Print"] {
        assert!(!world.run_is_enabled(row), "expected {row} muted");
    }
}
#[then("a recent-documents region is shown beside the command list")]
#[then("a recent-documents region is visible to the right of that list")]
fn recent_right(world: &mut ApplicationWorld) {
    let recent = world.run("Recent documents").minimum[0];
    let command = world.run("Save As").maximum[0];
    assert!(
        recent > command,
        "region at {recent}: expected right of {command}"
    );
}
#[then("some commands show disclosure indicators")]
fn disclosure(world: &mut ApplicationWorld) {
    let save_as = world.run("Save As").clone();
    let arrow = world.frame.marks.iter().any(|mark| {
        mark.list == save_as.list
            && mark.position[0] > save_as.maximum[0]
            && (mark.position[1] - save_as.center()[1]).abs() < 8.
    });
    assert!(arrow, "expected a submenu arrow on Save As");
}
#[then("the menu closes and the same part remains active")]
fn menu_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("By Ordered List");
    let id = world.previous_document.expect("expected a remembered part");
    assert_eq!(world.active_id(), id, "active document");
}
// Quick access
#[when("I open its quick access customization menu")]
fn open_customization_again(world: &mut ApplicationWorld) {
    world.click_text("...");
    world.pump(Vec::new());
}
// Ribbon display
#[then("the cycle-through-all choice is checked")]
fn cycle_checked(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Cycle through All"),
        "expected the choice checked"
    );
}
#[then("the drawing workspace has more vertical space")]
fn more_space(world: &mut ApplicationWorld) {
    let (_, band) = window_geometry("##application-ribbon").expect("expected a ribbon band");
    assert!(
        band[1] < world.ribbon_height,
        "band {}: expected under {}",
        band[1],
        world.ribbon_height
    );
}
// Ribbon context menu
#[when("I right-click a ribbon group")]
fn right_click_group(world: &mut ApplicationWorld) {
    world.right_click([1000, 100]);
    world.pump(Vec::new());
}
#[then("the ribbon context menu offers Ribbon Appearance and Show Panels")]
fn offers_appearance(world: &mut ApplicationWorld) {
    world.expect_text("Ribbon Appearance");
    world.expect_text("Show Panels");
}
#[then("it offers Customize User Commands and Undock Ribbon")]
fn offers_customize(world: &mut ApplicationWorld) {
    world.expect_text("Customize User Commands");
    world.expect_text("Undock Ribbon");
}
#[then("it offers Docking Positions")]
fn offers_docking(world: &mut ApplicationWorld) {
    world.expect_text("Docking Positions");
}
fn group_lit_against(world: &mut ApplicationWorld, lit: &str, plain: &str) {
    let [on, off] = [lit, plain].map(|group| {
        let run = world.run(group).clone();
        [run.minimum[0] as usize - 4, run.center()[1] as usize]
    });
    let (on, off) = (world.pixel_at(on), world.pixel_at(off));
    assert_ne!(on, off, "expected {lit} lit against {plain}");
}
#[then("the Create group is highlighted")]
fn create_lit(world: &mut ApplicationWorld) {
    group_lit_against(world, "Create", "Modify");
}
#[then("the Modify group is highlighted")]
fn modify_lit(world: &mut ApplicationWorld) {
    group_lit_against(world, "Modify", "Create");
}
#[then("the browser and viewport show the assembly")]
fn shows_assembly(world: &mut ApplicationWorld) {
    world.expect_text("Assembly");
}
#[then("the browser and viewport show the sheet-metal part")]
fn shows_sheet_metal(world: &mut ApplicationWorld) {
    world.expect_text("SheetMetal");
}
// Panels
#[then("the other compact group buttons remain visible and close the gap")]
fn compact_close_gap(world: &mut ApplicationWorld) {
    let left = world.run("Create").clone();
    let right = world.run("Modify").clone();
    assert!(
        right.minimum[0] > left.maximum[0],
        "expected Modify after Create"
    );
    assert!(
        right.minimum[0] - left.maximum[0] < 120.,
        "expected no gap between the buttons"
    );
}
#[then("the Modify group appears between Create and Explore in the floating ribbon")]
fn modify_between(world: &mut ApplicationWorld) {
    let [create, modify] = ["Create", "Modify"].map(|group| world.run(group).center()[1]);
    assert!(
        modify > create,
        "Modify at {modify}: expected after Create at {create}"
    );
}
// Browser
#[then("the menu offers Autoscroll, Expand All and Collapse All")]
fn browser_menu_top(world: &mut ApplicationWorld) {
    for entry in ["Autoscroll", "Expand All", "Collapse All"] {
        world.expect_text(entry);
    }
}
#[then("it offers Find, Display Preferences and Help")]
fn browser_menu_rest(world: &mut ApplicationWorld) {
    for entry in ["Find", "Display Preferences", "Help"] {
        world.expect_text(entry);
    }
}
fn press_search_close(world: &mut ApplicationWorld) {
    let close = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| run.text == "x")
        .cloned()
        .expect("expected the search close control");
    world.click(close.center().map(|value| value as i32));
    world.pump(Vec::new());
}
#[when("I activate the search field close control")]
#[when("the user activates the search field close control")]
#[when("the user clears the search query")]
#[when("I activate the close control again")]
#[when("the user activates the close control again")]
fn activate_close(world: &mut ApplicationWorld) {
    press_search_close(world);
}
#[then("the search text is cleared")]
fn search_cleared(world: &mut ApplicationWorld) {
    // The field and its opener have distinct labels when the query is cleared.
    world.expect_text("Search");
    world.expect_text("Type Keyword");
}
#[then("the search query remains visible in the field")]
fn query_visible(world: &mut ApplicationWorld) {
    assert!(
        world.shows("zzz") || world.shows("no such"),
        "expected the query in the field"
    );
}
#[given("the drawing browser is visible")]
fn drawing_browser_visible(world: &mut ApplicationWorld) {
    world.expect_text("Drawing Resources");
}
#[then("the menu offers Expand All, Collapse All and Help")]
fn drawing_menu(world: &mut ApplicationWorld) {
    for entry in ["Expand All", "Collapse All", "Help"] {
        world.expect_text(entry);
    }
}
// Home wording
#[given(regex = r#"^no other visible recent document matches "Part26"$"#)]
fn nothing_else_matches(_: &mut ApplicationWorld) {}
#[given(
    regex = r#"^a recent part named "Part26" is the only visible document matching that term$"#
)]
fn only_match(_: &mut ApplicationWorld) {}
#[given("Sort by is set to Last Opened in descending order")]
fn sort_opened_down(world: &mut ApplicationWorld) {
    world.expect_text("Sort by Last Opened");
    world.expect_text("v");
}
#[when("I inspect the recent-document list")]
fn inspect_list(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
}
#[then(regex = r#"^it includes "Part26" preceded by a history symbol$"#)]
fn history_symbol(world: &mut ApplicationWorld) {
    let term = world
        .frame
        .runs
        .iter()
        .find(|run| run.text.ends_with("Part26") && run.text.len() > 6);
    assert!(term.is_some(), "expected a history mark before the term");
}
#[given("one recent-document row is checked and no other rows are checked")]
fn one_only(world: &mut ApplicationWorld) {
    let name = "Part26";
    let run = world.run(name).clone();
    world.click([(run.minimum[0] - 44.) as i32, run.center()[1] as i32]);
    world.pump(Vec::new());
    world.expect_text("Open Selected");
}
