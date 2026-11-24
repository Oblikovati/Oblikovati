// SPDX-License-Identifier: GPL-2.0-only
//! Native project configurations shared with the scenario authors.
use super::{ApplicationWorld, PROJECT_NAME, choose_button, open_projects};
use cucumber::given;
use oblikovati_viewport::ViewportPhysicalKey;
use std::path::PathBuf;

const ACTIVE: &str = "ActiveProject";
const ALTERNATE: &str = "AlternateProject";
const WIZARD: &str = "WizardProject";
const VISIBLE_PART: &str = "fixtures/cad/workspace-layout-native.opd";

pub(super) fn active_file(project: &str) -> &'static str {
    match project {
        ACTIVE => "ActiveProject.opj",
        WIZARD => "WizardProject.opj",
        PROJECT_NAME => "gearbox.opj",
        other => panic!("project {other:?}: expected a registered fixture role"),
    }
}

pub(super) fn original_project(world: &ApplicationWorld) -> &'static str {
    [ACTIVE, WIZARD, PROJECT_NAME]
        .into_iter()
        .find(|name| {
            world.shows(&format!("(active) {name}")) || world.shows(&format!("Project: {name}"))
        })
        .expect("expected the original project row or chooser field")
}

pub(super) fn alternate_file(world: &ApplicationWorld) -> &'static str {
    if alternate_name(world) == ALTERNATE {
        "AlternateProject.opj"
    } else {
        "turbine.opj"
    }
}

pub(super) fn alternate_name(world: &ApplicationWorld) -> &'static str {
    if world.shows(ALTERNATE) {
        ALTERNATE
    } else {
        super::OTHER_PROJECT
    }
}

pub(super) fn fixture_file(name: &str) -> PathBuf {
    super::super::scenario_root()
        .join("fixtures/cad/project-dialog-native")
        .join(name)
}

fn register(world: &mut ApplicationWorld, file: &str) {
    world
        .app()
        .owner
        .load_desktop_project(&fixture_file(file))
        .unwrap();
    world.pump(Vec::new());
}

#[given(
    "`fixtures/cad/workspace-layout-native.opd` is open with `Sketch1` visible and the new-document template chooser open in the MCAD Software"
)]
fn open_part_and_chooser(world: &mut ApplicationWorld) {
    start_active_project(world);
    world.open_fixture(&super::super::scenario_root().join(VISIBLE_PART));
    visible_sketch_and_chooser(world);
}

#[given(
    "the MCAD Software starts with `fixtures/cad/project-dialog-native/ActiveProject.opj` as its first project and `fixtures/cad/workspace-layout-native.opd` as its part"
)]
fn start_project_with_part(world: &mut ApplicationWorld) {
    start_active_project(world);
    let part = super::super::scenario_root().join(VISIBLE_PART);
    crate::startup::open_startup_documents(&mut world.app().owner, &[part]).unwrap();
    world.pump(Vec::new());
}

#[given("`Sketch1` is visible and the new-document template chooser is open")]
fn visible_sketch_and_chooser(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected an open part");
    assert!(
        browser
            .sketches
            .iter()
            .any(|sketch| sketch.name == "Sketch1" && sketch.visible),
        "expected Sketch1 to be visible in the fixture"
    );
    world.key(ViewportPhysicalKey::N, true);
    world.expect_text("Template folders");
}

#[given(
    "the MCAD Software starts with `fixtures/cad/project-dialog-native/ActiveProject.opj` as its first project"
)]
fn start_active_project(world: &mut ApplicationWorld) {
    register(world, active_file(ACTIVE));
}

#[given(
    "the MCAD Software starts with `fixtures/cad/project-dialog-native/ActiveProject.opj` first and `fixtures/cad/project-dialog-native/AlternateProject.opj` second"
)]
fn start_two_projects(world: &mut ApplicationWorld) {
    start_active_project(world);
    register(world, "AlternateProject.opj");
}

#[given(
    "the MCAD Software starts with `fixtures/cad/project-dialog-native/WizardProject.opj` as its first project"
)]
fn start_wizard_project(world: &mut ApplicationWorld) {
    register(world, active_file(WIZARD));
}

#[given("`ActiveProject` is the active project")]
fn active_project(world: &mut ApplicationWorld) {
    world.expect_text("Project: ActiveProject");
}

#[given("the Projects dialog is open with `ActiveProject` and `AlternateProject` listed")]
fn two_projects(world: &mut ApplicationWorld) {
    open_registered_dialog(world, ACTIVE, &["AlternateProject.opj"]);
    project_rows_listed(world);
}

#[given("`ActiveProject` and `AlternateProject` are listed")]
fn project_rows_listed(world: &mut ApplicationWorld) {
    world.expect_text("(active) ActiveProject");
    world.expect_text(ALTERNATE);
    super::expect_list_location(world, "AlternateProject.opj");
}

#[given("`ActiveProject` is marked active and `AlternateProject` remains unselected")]
fn original_marked_and_selected(world: &mut ApplicationWorld) {
    super::expect_project_marked(world);
    assert!(
        !world.row_highlighted(ALTERNATE),
        "expected AlternateProject unselected"
    );
    world.expect_text("Workspace: workspace-active");
    world.observed_rows = Some(ACTIVE.to_owned());
}

#[given("the Projects dialog is open with `ActiveProject` active")]
#[given("the Projects dialog is open with `ActiveProject` selected")]
fn active_dialog(world: &mut ApplicationWorld) {
    open_registered_dialog(world, ACTIVE, &[]);
    world.expect_text("Workspace: workspace-active");
}

#[given("the Projects dialog is open with `WizardProject` active")]
fn wizard_project(world: &mut ApplicationWorld) {
    open_registered_dialog(world, WIZARD, &[]);
}

fn open_registered_dialog(world: &mut ApplicationWorld, active: &str, others: &[&str]) {
    register(world, active_file(active));
    for file in others {
        register(world, file);
    }
    open_projects_from_chooser(world);
}

#[given("the user opens the Projects dialog from the template chooser")]
fn open_projects_from_chooser(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::N, true);
    open_projects(world);
    super::expect_project_marked(world);
}

#[given("the user opens the Projects dialog from the template chooser with `ActiveProject` active")]
fn open_active_projects(world: &mut ApplicationWorld) {
    open_projects_from_chooser(world);
    world.expect_text("(active) ActiveProject");
}

#[given("the user opens the Projects dialog from the template chooser and selects `ActiveProject`")]
fn select_active_project(world: &mut ApplicationWorld) {
    open_active_projects(world);
    choose_button(world, "(active) ActiveProject");
    world.expect_text("Workspace: workspace-active");
}

#[given("the user opens the Projects dialog from the template chooser with `WizardProject` active")]
fn open_wizard_projects(world: &mut ApplicationWorld) {
    open_projects_from_chooser(world);
    world.expect_text("(active) WizardProject");
}

#[given(
    "`WizardProject` is active and the new single-user project wizard is on its project-type page in the MCAD Software"
)]
fn wizard_type_page(world: &mut ApplicationWorld) {
    wizard_project(world);
    choose_button(world, "New");
    project_type_page(world);
}

#[given("the user opens Projects from the template chooser and chooses New")]
fn choose_new_project(world: &mut ApplicationWorld) {
    open_wizard_projects(world);
    choose_button(world, "New");
}

#[given("the new single-user project wizard is on its project-type page")]
fn project_type_page(world: &mut ApplicationWorld) {
    world.expect_text(super::NEW_PROJECT_ROW);
}

#[given(
    "the user opens Projects from the template chooser, chooses New and advances to the project-file page"
)]
fn choose_project_file_page(world: &mut ApplicationWorld) {
    super::point_home_at_scratch(world);
    choose_new_project(world);
    project_type_page(world);
    choose_button(world, "Next");
    world.expect_text("Project File to be Created");
}

#[given(
    "`WizardProject` is active and the new single-user project wizard is on its project-file page in the MCAD Software"
)]
fn wizard_file_page(world: &mut ApplicationWorld) {
    super::point_home_at_scratch(world);
    wizard_type_page(world);
    choose_button(world, "Next");
    world.expect_text("Project File to be Created");
}
