// SPDX-License-Identifier: GPL-2.0-only
//! `features/application/project-dialog`: what the chooser's Projects action shows.
use super::ApplicationWorld;
use cucumber::{given, then, when};

#[path = "browse_native.rs"]
mod browse_native;
#[path = "cancellation.rs"]
mod cancellation;
#[cfg(target_os = "linux")]
#[path = "native_input.rs"]
pub(super) mod native_input;
#[path = "role_fixtures.rs"]
mod role_fixtures;
#[path = "settings_tree.rs"]
mod settings_tree;
#[path = "workspace_folder.rs"]
mod workspace_folder;
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
    world.app().owner.load_desktop_project(&path).unwrap();
    world.pump(Vec::new());
}
#[when("I choose Projects in the chooser")]
#[when("the user chooses Projects in the chooser")]
pub(super) fn choose_projects(world: &mut ApplicationWorld) {
    let cancel = world.run_in_window("New document", "Cancel");
    let button = world
        .frame
        .runs
        .iter()
        .find(|run| run.list == cancel.list && run.text.starts_with("Projects"))
        .expect("expected Projects beside Cancel in the template chooser");
    // The active-project caption may share the rendered text run with the button.
    let first_glyph = [button.minimum[0] + 4., button.center()[1]];
    world.click(super::diagnostics::pointer_pixels(first_glyph));
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
    let project = role_fixtures::original_project(world);
    world.expect_text(&format!("(active) {project}"));
    expect_list_location(world, role_fixtures::active_file(project));
}

pub(super) fn expect_list_location(world: &ApplicationWorld, file: &str) {
    let column = project_column_text(world, "Location", f32::INFINITY);
    assert!(
        column.contains(file),
        "location column {column:?}: expected {file}"
    );
}

pub(super) fn expect_list_name(world: &ApplicationWorld, label: &str) {
    let column = project_column_text(world, "Name", world.run("Location").minimum[0]);
    assert!(
        column.contains(&label.replace(' ', "")),
        "name column {column:?}: expected {label}"
    );
}

fn project_column_text(world: &ApplicationWorld, header: &str, right: f32) -> String {
    let heading = world.run(header);
    let settings_y = world.run("Project Settings").minimum[1];
    world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.minimum[0] >= heading.minimum[0]
                && run.minimum[0] < right
                && run.minimum[1] > heading.maximum[1]
                && run.maximum[1] < settings_y
        })
        .map(|run| run.text.as_str())
        .collect()
}
#[then("the active project is marked")]
fn project_marked(world: &mut ApplicationWorld) {
    expect_project_marked(world);
}
#[then("the dialog displays a project settings tree")]
fn project_tree(world: &mut ApplicationWorld) {
    let state = world.state();
    let project = state
        .projects
        .iter()
        .find(|project| project.active)
        .expect("expected the active project");
    let (_, workspace) = project
        .settings
        .iter()
        .find(|(name, _)| name == "Workspace")
        .expect("expected the project's workspace setting");
    world.expect_text(&format!("Workspace: {workspace}"));
    world.expect_text("Folder Options");
}
#[when("I choose Done without changing the project")]
#[when("the user chooses Done without changing the project")]
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
    world.app().owner.load_desktop_project(&path).unwrap();
    world.pump(Vec::new());
}
fn open_projects(world: &mut ApplicationWorld) {
    choose_projects(world);
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
    let project = role_fixtures::original_project(world);
    world.expect_text(&format!("(active) {project}"));
}
fn expect_projects_dialog(world: &ApplicationWorld) {
    world.expect_no_text(NEW_PROJECT_ROW);
    world.expect_no_text("Project File to be Created");
    world.expect_text("Project Settings");
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
    world.expect_text(OTHER_PROJECT);
}
#[given("the active project is marked in the list")]
fn marked_in_list(world: &mut ApplicationWorld) {
    expect_project_marked(world);
}
#[when("I select the other project with a single click")]
#[when("the user selects the other project with a single click")]
fn select_other(world: &mut ApplicationWorld) {
    choose_button(world, role_fixtures::alternate_name(world));
    let point = world
        .run("Project Settings")
        .center()
        .map(|coord| coord as i32);
    world.hover(point);
    world.pump(Vec::new());
}
#[then("the other project row is highlighted")]
fn other_highlighted(world: &mut ApplicationWorld) {
    assert!(
        world.row_highlighted(role_fixtures::alternate_name(world)),
        "expected the alternate project's row accent"
    );
}
#[then("the settings tree displays information for the selected project")]
fn tree_shows_other(world: &mut ApplicationWorld) {
    let file = role_fixtures::alternate_file(world);
    let label = world.run("Location:");
    let workspace = world.run("Workspace:");
    let location: String = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.list == label.list
                && run.minimum[1] >= label.minimum[1]
                && run.minimum[1] < workspace.minimum[1]
        })
        .map(|run| run.text.as_str())
        .collect();
    assert!(
        location.contains(file),
        "tree shows {location:?}: expected {file}"
    );
    if file == "AlternateProject.opj" {
        world.expect_text("Workspace: workspace-alternate");
    }
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
#[when("the user chooses New")]
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
#[then("the Next and cancels controls are available")]
fn next_and_cancel(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Next"), "expected Next available");
    assert!(world.run_is_enabled("Cancel"), "expected Cancel available");
}
#[when("I choose Next")]
#[when("the user chooses Next")]
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
