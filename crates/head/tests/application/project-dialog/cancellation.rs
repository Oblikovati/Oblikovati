// SPDX-License-Identifier: GPL-2.0-only
//! Checks for canceling a new-project wizard.
use super::ApplicationWorld;
use cucumber::{given, then};

// The scenario uses a project without configured libraries.
#[given("the wizard fixture has no entries in either library list")]
fn empty_library_fixture(world: &mut ApplicationWorld) {
    let path = world.scratch_folder().join("gearbox.opj");
    let project = std::fs::read_to_string(path).expect("expected fixture project");
    assert!(
        !project.contains("libraries:"),
        "fixture must omit project libraries"
    );
}
#[then("both library lists are empty")]
fn empty_library_lists(world: &mut ApplicationWorld) {
    let empty = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "(none)")
        .count();
    assert_eq!(empty, 2, "expected both library lists to be empty");
}
#[then("a provisional project entry is visible in the project list behind the wizard")]
fn provisional_visible(world: &mut ApplicationWorld) {
    let wizard = world.run("Select Libraries").list;
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.text.replace(' ', "") == "NewProject" && run.list < wizard),
        "expected a provisional entry behind the wizard"
    );
}

#[then("the chooser still displays the original active project")]
fn chooser_keeps_project(world: &mut ApplicationWorld) {
    world.expect_text(&format!("Project: {}", super::PROJECT_NAME));
}
