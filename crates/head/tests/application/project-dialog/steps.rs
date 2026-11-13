// SPDX-License-Identifier: GPL-2.0-only
//! `features/application/project-dialog`: what the chooser's Projects action shows.
use super::ApplicationWorld;
use cucumber::{given, then, when};

const PROJECT_NAME: &str = "Gearbox";

#[test]
fn projects_window_closes_to_the_document_chooser() {
    let mut world = ApplicationWorld::new();
    project_active(&mut world);
    world.add_part();
    world.key(oblikovati_viewport::ViewportPhysicalKey::N, true);
    world.expect_text("Template folders");
    choose_projects(&mut world);
    projects_above(&mut world);
    choose_done(&mut world);
    projects_closed(&mut world);
    chooser_again(&mut world);
    world.close();
}
#[given("a project is active")]
fn project_active(world: &mut ApplicationWorld) {
    let path = world.scratch_folder().join("gearbox.opj");
    std::fs::write(&path, "schemaVersion: 1\nname: Gearbox\nworkspace: work\n").unwrap();
    world.app().owner.load_project_for_test(&path);
    world.pump(Vec::new());
}
#[when("I choose Projects in the chooser")]
fn choose_projects(world: &mut ApplicationWorld) {
    world.click_text("Projects");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("a Projects dialog opens above the chooser")]
fn projects_above(world: &mut ApplicationWorld) {
    world.expect_text("Project Settings");
    let dialog = world.run("Project Settings").list;
    let chooser = world.run("Template folders").list;
    assert!(
        dialog > chooser,
        "Projects: expected it drawn above the chooser"
    );
}
#[then("it lists project names and locations")]
#[then("the dialog lists project names and locations")]
fn project_rows_listed(world: &mut ApplicationWorld) {
    world.expect_text(PROJECT_NAME);
    world.expect_text("gearbox.opj");
}
#[then("the active project is marked")]
fn project_marked(world: &mut ApplicationWorld) {
    world.expect_text(&format!("(active) {PROJECT_NAME}"));
}
#[then("the dialog displays a project settings tree")]
fn project_tree(world: &mut ApplicationWorld) {
    world.expect_text("Workspace: work");
}
#[when("I choose Done without changing the project")]
fn choose_done(world: &mut ApplicationWorld) {
    world.click_text("Done");
    world.pump(Vec::new());
}
#[then("the Projects dialog closes")]
fn projects_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Project Settings");
}
#[then("the template chooser is displayed again")]
fn chooser_again(world: &mut ApplicationWorld) {
    world.expect_text("Template folders");
}
#[then("New, Browse, Save, Apply and Done controls are displayed")]
fn project_controls(world: &mut ApplicationWorld) {
    for control in ["New", "Browse", "Save", "Apply", "Done"] {
        world.expect_text(control);
    }
}
// The inactive project, the wizard and the settings tree
const OTHER_PROJECT: &str = "Turbine";
const NEW_PROJECT_ROW: &str = "New Single User Project";
fn load_project(world: &mut ApplicationWorld, file: &str, name: &str) {
    let path = world.scratch_folder().join(file);
    let text = format!(
        "schemaVersion: 1\nname: {name}\nworkspace: work\ntemplates: Forms\ndesignData: Design\n"
    );
    std::fs::write(&path, text).unwrap();
    world.app().owner.load_project_for_test(&path);
    world.pump(Vec::new());
}
fn open_projects(world: &mut ApplicationWorld) {
    world.click_text("Projects");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn open_projects_dialog(world: &mut ApplicationWorld) {
    load_project(world, "gearbox.opj", PROJECT_NAME);
    world.key(oblikovati_viewport::ViewportPhysicalKey::N, true);
    open_projects(world);
    world.expect_text("Project Settings");
}
fn open_wizard_on_type_page(world: &mut ApplicationWorld) {
    open_projects_dialog(world);
    choose_button(world, "New");
    world.expect_text(NEW_PROJECT_ROW);
}
fn open_wizard_on_file_page(world: &mut ApplicationWorld) {
    open_wizard_on_type_page(world);
    choose_button(world, "Next");
    world.expect_text("Project File to be Created");
}
fn choose_button(world: &mut ApplicationWorld, label: &str) {
    world.click_text(label);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn expect_project_marked(world: &ApplicationWorld) {
    world.expect_text(&format!("(active) {PROJECT_NAME}"));
}
fn expect_projects_dialog(world: &ApplicationWorld) {
    world.expect_no_text(NEW_PROJECT_ROW);
    world.expect_no_text("Project File to be Created");
    world.expect_text("Project Settings");
}
/// The run that holds this text; rendered runs have no spaces.
fn run_holding(world: &ApplicationWorld, text: &str) -> String {
    let wanted = text.replace(' ', "");
    world
        .frame
        .runs
        .iter()
        .find(|run| run.text.replace(' ', "").contains(&wanted))
        .unwrap_or_else(|| panic!("expected {text:?} in a rendered run"))
        .text
        .clone()
}
/// A wrapped path comes in several runs, so this reads them as one line.
fn holds_text(world: &ApplicationWorld, text: &str) -> bool {
    let all: String = world
        .frame
        .runs
        .iter()
        .map(|run| run.text.as_str())
        .collect();
    all.replace(' ', "").contains(&text.replace(' ', ""))
}
// Inspect an inactive project without activating it
#[given("the Projects dialog lists an active project and another project")]
fn two_projects(world: &mut ApplicationWorld) {
    load_project(world, "gearbox.opj", PROJECT_NAME);
    load_project(world, "turbine.opj", OTHER_PROJECT);
    open_projects(world);
    world.expect_text(&run_holding(world, OTHER_PROJECT));
}
#[given("the active project is marked in the list")]
fn marked_in_list(world: &mut ApplicationWorld) {
    expect_project_marked(world);
}
#[when("I select the other project with a single click")]
fn select_other(world: &mut ApplicationWorld) {
    let row = run_holding(world, "turbine.opj");
    choose_button(world, &row);
}
#[then("the other project row is highlighted")]
fn other_highlighted(world: &mut ApplicationWorld) {
    let row = run_holding(world, "turbine.opj");
    assert!(world.row_highlighted(&row), "expected the row's accent");
}
#[then("the settings tree displays information for the selected project")]
fn tree_shows_other(world: &mut ApplicationWorld) {
    let location = run_holding(world, "Location:");
    assert!(location.contains("turbine.opj"), "tree shows {location:?}");
}
#[then("the active-project mark remains beside the original project")]
#[then("the active project remains marked in the project list")]
#[then("the original project remains marked active")]
fn mark_remains(world: &mut ApplicationWorld) {
    expect_project_marked(world);
}
#[when("I choose Done")]
#[when("the user chooses Done")]
fn choose_done_button(world: &mut ApplicationWorld) {
    choose_button(world, "Done");
    world.pump(Vec::new());
}
#[then("the chooser still displays the original active project")]
fn chooser_keeps_project(world: &mut ApplicationWorld) {
    world.expect_text(&format!("Project: {PROJECT_NAME}"));
}
// The new single-user project wizard
#[given("the Projects dialog is open in the MCAD Software")]
fn dialog_open(world: &mut ApplicationWorld) {
    open_projects_dialog(world);
}
#[given("an existing project is active")]
fn existing_active(world: &mut ApplicationWorld) {
    expect_project_marked(world);
}
#[when("I choose New")]
fn choose_new(world: &mut ApplicationWorld) {
    choose_button(world, "New");
}
#[then("the new-project wizard opens on a project-type page")]
#[then("the project-type page is displayed")]
fn type_page(world: &mut ApplicationWorld) {
    world.expect_text(NEW_PROJECT_ROW);
}
#[then("New Single User Project is selected")]
#[then("New Single User Project remains selected")]
fn single_user_selected(world: &mut ApplicationWorld) {
    assert!(
        world.row_highlighted(NEW_PROJECT_ROW),
        "expected the row's accent"
    );
}
#[then("the Next and Cancel controls are available")]
fn next_and_cancel(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Next"), "expected Next available");
    assert!(world.run_is_enabled("Cancel"), "expected Cancel available");
}
#[when("I choose Next")]
fn choose_next(world: &mut ApplicationWorld) {
    choose_button(world, "Next");
}
#[then("the wizard displays a project name, workspace folder and project file to be created")]
#[then("the project-file page displays the project name and workspace folder")]
#[then("the project-file page is displayed")]
#[then("its project name and workspace folder remain displayed")]
fn file_page(world: &mut ApplicationWorld) {
    world.expect_text("Name");
    world.expect_text("Project (Workspace) Folder");
    world.expect_text("Project File to be Created");
    assert!(
        holds_text(world, "New Project.opj"),
        "expected the derived file"
    );
}
#[then("it provides a folder browsing control")]
fn browse_control(world: &mut ApplicationWorld) {
    world.expect_text("...");
}
#[then("the wizard displays a Select Libraries page")]
#[then("the Select Libraries page is displayed")]
fn libraries(world: &mut ApplicationWorld) {
    world.expect_text("Select Libraries");
}
#[then("the page has All Projects and New Project lists with transfer controls")]
fn library_lists(world: &mut ApplicationWorld) {
    for row in ["All Projects", "New Project", ">>", "<<"] {
        world.expect_text(row);
    }
}
#[then("the wizard closes and the Projects dialog is displayed")]
fn wizard_closes(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    expect_projects_dialog(world);
}
#[then("the provisional project entry is no longer listed")]
fn provisional_gone(world: &mut ApplicationWorld) {
    assert!(
        !holds_text(world, "New Project"),
        "expected no provisional row"
    );
}
#[given("the new single-user project wizard is on its project-type page in the MCAD Software")]
fn wizard_on_type(world: &mut ApplicationWorld) {
    open_wizard_on_type_page(world);
}
#[then("Back and Finish are unavailable")]
fn back_finish_off(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("Back"), "expected Back unavailable");
    assert!(
        !world.run_is_enabled("Finish"),
        "expected Finish unavailable"
    );
}
// The workspace folder browser. It starts where the system keeps documents, so HOME names a
// folder of this scenario's own; the machine's folders are no part of what it checks.
fn point_home_at_scratch(world: &mut ApplicationWorld) {
    let home = world.scratch_folder();
    std::fs::create_dir_all(home.join("Documents").join("Drawings")).unwrap();
    // SAFETY: the scenarios run one at a time on this thread, which also owns the window.
    unsafe { std::env::set_var("HOME", home) };
}
fn wizard_text(world: &ApplicationWorld) -> String {
    world
        .frame
        .runs
        .iter()
        .filter(|run| run.text.contains(".opj") || run.text.contains("Documents"))
        .map(|run| run.text.as_str())
        .collect::<Vec<_>>()
        .join("|")
}
#[given("the new single-user project wizard is on its project-file page in the MCAD Software")]
fn wizard_on_file(world: &mut ApplicationWorld) {
    point_home_at_scratch(world);
    open_wizard_on_file_page(world);
}
#[given("a workspace folder and project file location are displayed")]
fn locations_shown(world: &mut ApplicationWorld) {
    world.expect_text("Project (Workspace) Folder");
    world.observed_rows = Some(wizard_text(world));
}
#[given("the project has a displayed name")]
fn name_shown(world: &mut ApplicationWorld) {
    assert!(
        holds_text(world, "New Project.opj"),
        "expected a named project file"
    );
}
#[when("I choose the workspace folder browse control")]
fn browse_folder(world: &mut ApplicationWorld) {
    choose_button(world, "...");
}
#[then("a folder browser opens with a folder tree and a Folder field")]
fn folder_browser_open(world: &mut ApplicationWorld) {
    world.expect_text("Documents");
    assert!(holds_text(world, "Folder:"), "expected the Folder field");
}
#[then("it offers Make New Folder, OK and Cancel")]
fn folder_controls(world: &mut ApplicationWorld) {
    for row in ["Make New Folder", "OK", "Cancel"] {
        world.expect_text(row);
    }
}
#[when("I select Documents in the folder tree")]
#[when("I select the Documents folder")]
fn select_documents(world: &mut ApplicationWorld) {
    choose_button(world, "Documents");
}
#[then("Documents is selected and its child folders are displayed")]
fn documents_open(world: &mut ApplicationWorld) {
    assert!(
        world.row_highlighted("Documents"),
        "expected the row's accent"
    );
    world.expect_text("Drawings");
}
#[then("the Folder field displays Documents")]
fn folder_field(world: &mut ApplicationWorld) {
    world.expect_text("Folder: Documents");
}
#[when("I choose Cancel in the folder browser")]
fn cancel_browser(world: &mut ApplicationWorld) {
    choose_button(world, "Cancel");
}
#[then("the folder browser closes")]
fn browser_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Make New Folder");
}
#[then("the wizard still displays the previous workspace folder and project file location")]
fn locations_kept(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_rows
        .clone()
        .expect("expected remembered locations");
    assert_eq!(wizard_text(world), earlier);
}
#[when("I cancel the project wizard")]
fn cancel_wizard(world: &mut ApplicationWorld) {
    choose_button(world, "Cancel");
}
#[then("the Projects dialog is displayed")]
fn projects_shown(world: &mut ApplicationWorld) {
    expect_projects_dialog(world);
}
#[when("I choose OK in the folder browser")]
fn accept_folder(world: &mut ApplicationWorld) {
    choose_button(world, "OK");
}
#[then("the wizard displays Documents as the workspace folder")]
fn workspace_is_documents(world: &mut ApplicationWorld) {
    assert!(
        holds_text(world, "/Documents"),
        "expected the folder ending in Documents"
    );
}
#[then(
    "the displayed project file location is inside Documents and uses the displayed project name"
)]
fn file_in_documents(world: &mut ApplicationWorld) {
    assert!(
        holds_text(world, "/Documents/New Project.opj"),
        "expected the derived file"
    );
}
// The settings tree and the arrow keys
fn press_arrow(world: &mut ApplicationWorld, key: oblikovati_viewport::ViewportPhysicalKey) {
    world.key(key, false);
    world.pump(Vec::new());
}
fn select_group(world: &mut ApplicationWorld, group: &str) {
    choose_button(world, group);
}
#[given("an active project with Folder Options and Options groups is displayed")]
fn groups_shown(world: &mut ApplicationWorld) {
    world.expect_text("Folder Options");
    world.expect_text("Options");
}
#[when("I select the collapsed Folder Options group")]
fn select_folder_options(world: &mut ApplicationWorld) {
    assert!(
        !holds_text(world, "Presets"),
        "expected the group collapsed"
    );
    select_group(world, "Folder Options");
}
#[when("I press the Right Arrow key")]
fn press_right(world: &mut ApplicationWorld) {
    press_arrow(world, oblikovati_viewport::ViewportPhysicalKey::RightArrow);
}
#[when("I press the Left Arrow key")]
fn press_left(world: &mut ApplicationWorld) {
    press_arrow(world, oblikovati_viewport::ViewportPhysicalKey::LeftArrow);
}
#[then("the group expands to show folder entries including Templates, Design Data and Presets")]
fn folder_entries(world: &mut ApplicationWorld) {
    for row in ["Templates:", "Design Data:", "Presets:"] {
        assert!(holds_text(world, row), "expected {row} in the tree");
    }
}
#[then("the Folder Options children are hidden")]
fn folder_entries_hidden(world: &mut ApplicationWorld) {
    assert!(
        !holds_text(world, "Presets:"),
        "expected the group collapsed"
    );
}
#[when("I select the collapsed Options group")]
fn select_options(world: &mut ApplicationWorld) {
    select_group(world, "Options");
}
#[then(
    "the group expands to show project options including old versions to keep and unique file names"
)]
fn option_entries(world: &mut ApplicationWorld) {
    for row in ["Old Versions To Keep", "Using Unique File Names"] {
        assert!(holds_text(world, row), "expected {row} in the tree");
    }
}
#[given(
    "the selected project has collapsed Appearance Libraries and Material Libraries groups with entries"
)]
fn library_groups(world: &mut ApplicationWorld) {
    world.expect_text("Appearance Libraries");
    world.expect_text("Material Libraries");
    assert!(
        !holds_text(world, "Default Appearances"),
        "expected the group collapsed"
    );
}
#[when("I select Appearance Libraries and press the Right Arrow key")]
fn open_appearances(world: &mut ApplicationWorld) {
    select_group(world, "Appearance Libraries");
    press_right(world);
}
#[then("the appearance library entries are visible beneath their group")]
#[then("the appearance library entries remain visible")]
fn appearance_entries(world: &mut ApplicationWorld) {
    assert!(
        holds_text(world, "Default Appearances"),
        "expected the entries"
    );
}
#[when("I select Material Libraries and press the Right Arrow key")]
fn open_materials(world: &mut ApplicationWorld) {
    select_group(world, "Material Libraries");
    press_right(world);
}
#[then("the material library entries are visible beneath their group")]
fn material_entries(world: &mut ApplicationWorld) {
    assert!(
        holds_text(world, "Default Materials"),
        "expected the entries"
    );
}
#[then("the material library entries are hidden")]
fn material_hidden(world: &mut ApplicationWorld) {
    assert!(
        !holds_text(world, "Default Materials"),
        "expected the group collapsed"
    );
}
