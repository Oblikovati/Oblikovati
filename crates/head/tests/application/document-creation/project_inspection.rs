// SPDX-License-Identifier: GPL-2.0-only
//! Inspect the project while a root-level native assembly template stays selected.
use super::*;
use oblikovati_gui::{GuiTemplateFolder, GuiTemplateKind};

const PROJECT: &str = "AssemblyTemplateFixture";
const PROJECT_FILE: &str = "assembly-template-fixture.opj";
const TEMPLATE: &str = "Standard (mm)";

#[given(
    "a fresh MCAD Software session loads `fixtures/cad/assembly-template-fixture.opj` as its active project"
)]
fn load_assembly_template_project(world: &mut ApplicationWorld) {
    open_fixture_project(world, "fixtures/cad/assembly-template-fixture.opj");
}

#[given("the user selects the `Standard (mm)` assembly template and displays its details")]
fn select_root_assembly_template(world: &mut ApplicationWorld) {
    let state = world.state();
    assert!(
        state
            .project_templates
            .iter()
            .any(|entry| entry.title == TEMPLATE
                && entry.folder == GuiTemplateFolder::Root
                && entry.kind == GuiTemplateKind::Assembly
                && entry
                    .path
                    .ends_with("assembly-template-native/Standard (mm).oad")),
        "catalog issues: {:?}",
        state.project_template_issue
    );
    world.click_text(TEMPLATE);
    expect_selected_template(world, TEMPLATE);
    expect_selected_template(world, "Document type: Assembly");
    expect_selected_template(world, "Length: millimeters (mm)");
    world.proposed_name = Some(super::super::super::refinements::selected_template_details(
        world,
    ));
}

#[then("the fixture project name and location are listed")]
fn fixture_project_is_listed(world: &mut ApplicationWorld) {
    super::super::super::project_dialog::expect_list_name(world, PROJECT);
    super::super::super::project_dialog::expect_list_location(world, PROJECT_FILE);
}

#[then("the fixture project is marked as active")]
fn fixture_project_is_active(world: &mut ApplicationWorld) {
    super::super::super::project_dialog::expect_list_name(world, &format!("(active) {PROJECT}"));
    assert!(
        world
            .state()
            .projects
            .iter()
            .any(|entry| entry.name == PROJECT && entry.active)
    );
}
