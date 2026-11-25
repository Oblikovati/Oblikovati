// SPDX-License-Identifier: GPL-2.0-only
//! Workspace folder selection from the new-project wizard.
use super::super::diagnostics;
use super::{
    ApplicationWorld, choose_button, expect_projects_dialog, holds_text, open_wizard_on_file_page,
    point_home_at_scratch,
};
use cucumber::{given, then, when};

#[path = "libraries_native.rs"]
mod libraries_native;

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
#[when("the user chooses the workspace folder browse control")]
fn browse_folder(world: &mut ApplicationWorld) {
    let page = world.run("Project File to be Created");
    let browse = world
        .frame
        .runs
        .iter()
        .find(|run| run.list == page.list && run.text.starts_with("..."))
        .expect("expected the workspace Browse button in the project wizard");
    // The workspace path can share a draw run with the button.
    let first_glyph = [browse.minimum[0] + 4., browse.center()[1]];
    world.click(diagnostics::pointer_pixels(first_glyph));
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("a folder browser opens with a folder tree and a Folder field")]
fn folder_browser_open(world: &mut ApplicationWorld) {
    world.run_in_window("Browse for Folder", "Documents");
    assert!(holds_text(world, "Folder:"), "expected the Folder field");
}
#[then("it offers Make New Folder, OK and Cancel")]
#[then("it offers Make New Folder, OK and cancels")]
fn folder_controls(world: &mut ApplicationWorld) {
    for label in ["Make New Folder", "OK", "Cancel"] {
        folder_button(world, label);
    }
}
#[when("I select Documents in the folder tree")]
#[when("the user selects Documents in the folder tree")]
#[when("I select the Documents folder")]
#[when("the user selects the Documents folder")]
fn select_documents(world: &mut ApplicationWorld) {
    let documents = world.run_in_window("Browse for Folder", "Documents");
    world.click(diagnostics::pointer_pixels(documents.center()));
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("Documents is selected and its child folders are displayed")]
fn documents_open(world: &mut ApplicationWorld) {
    let child = world.run_in_window("Browse for Folder", "Drawings");
    let documents = world
        .frame
        .runs
        .iter()
        .filter(|run| run.list == child.list && run.text == "Documents")
        .filter(|run| run.maximum[1] < child.minimum[1])
        .max_by(|left, right| left.minimum[1].total_cmp(&right.minimum[1]))
        .expect("expected the Documents row above its Drawings child");
    let x = documents.maximum[0].max(child.maximum[0]) as usize + 20;
    let selected_y = documents.center()[1] as usize;
    let plain_y = child.center()[1] as usize;
    assert!(
        world.pixel_at([x, selected_y]) != world.pixel_at([x, plain_y]),
        "expected Documents to be highlighted in the folder tree"
    );
}
#[then("the Folder field displays Documents")]
fn folder_field(world: &mut ApplicationWorld) {
    world.expect_text("Folder: Documents");
}
#[when("I choose Cancel in the folder browser")]
#[when("the user chooses Cancel in the folder browser")]
fn cancel_browser(world: &mut ApplicationWorld) {
    choose_folder_button(world, "Cancel");
}
#[then("the folder browser closes")]
fn browser_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Browse for Folder");
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
#[when("the user cancels the project wizard")]
fn cancel_wizard(world: &mut ApplicationWorld) {
    choose_button(world, "Cancel");
}
#[then("the Projects dialog is displayed")]
fn projects_shown(world: &mut ApplicationWorld) {
    expect_projects_dialog(world);
}
#[when("I choose OK in the folder browser")]
#[when("the user chooses OK in the folder browser")]
fn accept_folder(world: &mut ApplicationWorld) {
    choose_folder_button(world, "OK");
}
fn choose_folder_button(world: &mut ApplicationWorld, label: &str) {
    let at = diagnostics::pointer_pixels(folder_button(world, label).center());
    world.click(at);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn folder_button<'a>(world: &'a ApplicationWorld, label: &str) -> &'a diagnostics::RenderedRun {
    let title = world.run("Browse for Folder");
    let rendered_label = label.replace(' ', "");
    world
        .frame
        .runs
        .iter()
        .find(|run| run.list == title.list && run.text == rendered_label)
        .unwrap_or_else(|| panic!("expected {label} in Browse for Folder"))
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

#[test]
fn make_new_folder_changes_workspace_only_after_accepting_it() {
    let mut world = ApplicationWorld::new();
    point_home_at_scratch(&mut world);
    open_wizard_on_file_page(&mut world);
    locations_shown(&mut world);
    browse_folder(&mut world);
    select_documents(&mut world);
    choose_folder_button(&mut world, "Make New Folder");
    world.type_text("Abandoned");
    choose_folder_button(&mut world, "Cancel New Folder");
    let documents = world.scratch_folder().join("Documents");
    assert!(!documents.join("Abandoned").exists());
    folder_field(&mut world);

    choose_folder_button(&mut world, "Make New Folder");
    world.type_text("Workspace");
    choose_folder_button(&mut world, "Create");
    assert!(documents.join("Workspace").is_dir());
    world.expect_text("Folder: Workspace");
    cancel_browser(&mut world);
    browser_closed(&mut world);
    locations_kept(&mut world);

    browse_folder(&mut world);
    select_documents(&mut world);
    let row = world
        .run_in_window("Browse for Folder", "Workspace")
        .center();
    world.click(diagnostics::pointer_pixels(row));
    accept_folder(&mut world);
    assert!(holds_text(&world, "/Documents/Workspace/New Project.opj"));
    cancel_wizard(&mut world);
    projects_shown(&mut world);
    assert!(!documents.join("Workspace/New Project.opj").exists());
    world.close();
}

#[test]
fn finish_creates_a_project_and_preserves_an_existing_file() {
    let mut world = ApplicationWorld::new();
    point_home_at_scratch(&mut world);
    open_wizard_on_file_page(&mut world);
    browse_folder(&mut world);
    select_documents(&mut world);
    accept_folder(&mut world);
    let documents = world.scratch_folder().join("Documents");
    let occupied = documents.join("New Project.opj");
    std::fs::write(&occupied, b"existing project").unwrap();
    choose_button(&mut world, "Next");
    assert!(world.run_is_enabled("Finish"));
    choose_button(&mut world, "Finish");
    world.expect_text("Select Libraries");
    assert!(holds_text(&world, "create project"), "{}", world.text());
    assert_eq!(std::fs::read(&occupied).unwrap(), b"existing project");
    assert_eq!(world.state().projects.len(), 1);

    choose_button(&mut world, "Back");
    set_project_name(&mut world, "Design Office");
    choose_button(&mut world, "Next");
    choose_button(&mut world, "Finish");
    world.expect_no_text("Select Libraries");
    expect_projects_dialog(&mut world);
    assert_project_saved(&mut world, &documents.join("Design Office.opj"));
    world.close();
}

fn set_project_name(world: &mut ApplicationWorld, name: &str) {
    let label = world.run_in_window("New Project", "Name");
    let input = [label.minimum[0] - 45., label.center()[1]];
    world.click(diagnostics::pointer_pixels(input));
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
    world.type_text(name);
}

fn assert_project_saved(world: &mut ApplicationWorld, path: &std::path::Path) {
    let projects = world.state().projects;
    assert_eq!(projects.len(), 2);
    assert!(
        projects
            .iter()
            .any(|project| project.name == super::PROJECT_NAME && project.active)
    );
    assert!(projects.iter().any(|project| {
        project.name == "Design Office"
            && !project.active
            && project.location == path.display().to_string()
    }));
    world.reopen_application();
    world.app().owner.load_desktop_project(path).unwrap();
    let projects = world.state().projects;
    assert_eq!(projects.len(), 1);
    assert_eq!(projects[0].name, "Design Office");
    assert!(projects[0].active);
}
