// SPDX-License-Identifier: GPL-2.0-only
//! Exercise Create from the named, geometry-bearing native template.
use super::*;
use oblikovati_gui::{GuiTemplateFolder, GuiTemplateKind, GuiTemplateUnits};

const TEMPLATE_TITLE: &str = "Geometry Template";
const TEMPLATE: &str =
    "fixtures/cad/document-creation-template-content/Metric/Geometry Template.opd";

#[given(
    "a fresh MCAD Software session loads `fixtures/cad/document-creation-template-content.opj` as its active project"
)]
fn load_geometry_project(world: &mut ApplicationWorld) {
    open_fixture_project(world, "fixtures/cad/document-creation-template-content.opj");
}

#[given("no model document is open")]
fn no_open_model(world: &mut ApplicationWorld) {
    let state = world.state();
    assert!(state.documents.is_empty() && state.document.is_none());
}

fn geometry_template_path() -> std::path::PathBuf {
    super::super::super::scenario_root().join(TEMPLATE)
}

#[given(
    "the Metric catalog contains `fixtures/cad/document-creation-template-content/Metric/Geometry Template.opd`"
)]
fn contains_geometry_template(world: &mut ApplicationWorld) {
    let path = geometry_template_path().canonicalize().unwrap();
    let state = world.state();
    assert!(
        state.project_templates.iter().any(|entry| {
            entry.path == path
                && entry.title == TEMPLATE_TITLE
                && entry.kind == GuiTemplateKind::Part
                && entry.folder == GuiTemplateFolder::Units(GuiTemplateUnits::Metric)
        }),
        "catalog issues: {:?}",
        state.project_template_issue
    );
    world.bytes_before_save = Some(std::fs::read(path).unwrap());
}

#[when("the user opens the new-document template chooser")]
#[given("the user opens the new-document template chooser")]
fn open_named_template_chooser(world: &mut ApplicationWorld) {
    open_chooser(world);
    chooser_open(world);
}

#[when("the user selects the Metric folder and the Geometry Template part")]
fn select_geometry_template(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    world.click_text(TEMPLATE_TITLE);
    expect_selected_template(world, TEMPLATE_TITLE);
}

#[then("a new unsaved part document is active")]
fn unsaved_part_is_active(world: &mut ApplicationWorld) {
    new_part(world);
    let state = world.state();
    let document = state.document.unwrap();
    assert_eq!(state.documents.len(), 1);
    assert!(state.document_storage[&document].path.is_none());
    chooser_closed(world);
}

#[then("its viewport displays the template's rectangular solid")]
fn displays_rectangular_solid(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected a Part browser");
    let [body] = browser.bodies.as_slice() else {
        panic!("expected one solid from the template")
    };
    assert!(body.visible);
    assert_eq!(
        body.faces, 6,
        "expected the six faces of the rectangular prism"
    );
    super::super::super::workspace_layout::browser_feature::expect_solid_visible(world);
}

#[then("its model browser contains the template's extrusion and source sketch")]
fn browser_contains_template_recipe(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected a Part browser");
    let feature = browser
        .features
        .iter()
        .find(|row| row.name == "Extrusion1")
        .expect("expected the template's Extrusion1");
    assert_eq!(feature.kind, "Extrusion");
    assert!(
        browser
            .sketches
            .iter()
            .any(|sketch| sketch.name == "Sketch1" && sketch.consumed_by == Some(feature.id))
    );
    world.expect_text("Extrusion1");
    world.expect_text("Sketch1");
}

#[then("the original template file remains unchanged")]
fn source_template_unchanged(world: &mut ApplicationWorld) {
    let before = world
        .bytes_before_save
        .as_deref()
        .expect("expected the original template bytes");
    assert_eq!(std::fs::read(geometry_template_path()).unwrap(), before);
}
