// SPDX-License-Identifier: GPL-2.0-only
//! `features/application/project-dialog`: what the chooser's Projects action shows.
use super::ApplicationWorld;
use cucumber::{given, then, when};

const PROJECT_NAME: &str = "Gearbox";
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
