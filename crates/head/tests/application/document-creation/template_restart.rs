// SPDX-License-Identifier: GPL-2.0-only
//! Open the chooser with the project's native Part before and after a restart.
use super::*;
use oblikovati_gui::{GuiTemplateFolder, GuiTemplateKind, GuiTemplateMaterial, GuiTemplateUnits};

const PROJECT: &str = "fixtures/cad/document-creation-generic-candidate.opj";
const STANDARD: &str = "Standard (mm)";

#[given(
    "a fresh MCAD Software session loads `fixtures/cad/document-creation-generic-candidate.opj` as its active project"
)]
fn load_generic_project(world: &mut ApplicationWorld) {
    open_fixture_project(world, PROJECT);
}

#[given("its Metric folder contains a standard part template with the material label Generic")]
fn generic_template_is_available(world: &mut ApplicationWorld) {
    let state = world.state();
    assert!(state.project_templates.iter().any(|template| {
        template.title == STANDARD
            && template.kind == GuiTemplateKind::Part
            && template.folder == GuiTemplateFolder::Units(GuiTemplateUnits::Metric)
            && matches!(
                &template.material,
                Some(GuiTemplateMaterial::Assigned { name, .. }) if name == "Generic"
            )
    }));
}

#[given(
    "fixture `fixtures/cad/workspace-layout-native.opd` is open with Sketch1 visible in the MCAD Software"
)]
#[when("the user opens fixture `fixtures/cad/workspace-layout-native.opd`")]
fn open_restart_part(world: &mut ApplicationWorld) {
    open_part_with_visible_sketch(world);
}

#[then("the Create New File chooser opens over the part")]
fn open_over_part(world: &mut ApplicationWorld) {
    chooser_open(world);
    previous_document(world);
}

#[when("the user selects the standard metric part template in the Part group")]
fn select_generic_part(world: &mut ApplicationWorld) {
    world.click_text(STANDARD);
    generic_part_details(world);
}

#[when("the user closes the MCAD Software while the part remains unchanged")]
fn close_unchanged_session(world: &mut ApplicationWorld) {
    let active = world.active_id();
    assert!(!world.state().document_storage[&active].unsaved_edits);
    world.close();
}

#[when("the user restarts the MCAD Software")]
fn restart_with_project(world: &mut ApplicationWorld) {
    world.reopen_application();
    load_generic_project(world);
}

#[then("the template gallery is grouped by document type")]
fn grouped_gallery(world: &mut ApplicationWorld) {
    grouped_by_default(world);
}

#[then("Metric is selected in the template folder tree")]
fn metric_folder_in_tree(world: &mut ApplicationWorld) {
    metric_folder_selected(world);
}

#[then("the standard metric part template is selected in the Part group")]
fn restored_part_template(world: &mut ApplicationWorld) {
    assert!(gallery_has_template(world, STANDARD));
    expect_selected_template(world, STANDARD);
}

#[then("the details identify a Part with millimeter units and Generic material")]
fn generic_part_details(world: &mut ApplicationWorld) {
    expect_part_details(world);
    expect_selected_template(world, "Material: Generic");
}

#[then("Create and Cancel are available")]
fn buttons_available(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Create"));
    assert!(world.run_is_enabled("Cancel"));
}

#[then("the existing part and sketch remain visible behind the chooser")]
#[then("the same part and sketch remain active")]
fn existing_part_and_sketch(world: &mut ApplicationWorld) {
    use super::super::super::workspace_layout::native_part::fixture_sketch;

    previous_document(world);
    let active = world.active_id();
    let name = world.document_name(active);
    world.expect_tab(&name);
    let sketch = fixture_sketch(world);
    assert_eq!(sketch.name, "Sketch1");
    assert!(sketch.visible);
}

#[then("the chooser closes without creating a document")]
fn chooser_closed_without_creation(world: &mut ApplicationWorld) {
    chooser_closed(world);
    assert_eq!(world.state().documents.len(), 1);
}
