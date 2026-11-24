// SPDX-License-Identifier: GPL-2.0-only
//! Bind the basic creation cases to the native project and its supplied templates.
use super::*;
use oblikovati_gui::{GuiTemplateFolder, GuiTemplateKind, GuiTemplateUnits};

#[given(
    "a fresh MCAD Software session loads `fixtures/cad/document-creation-native-candidate.opj` as its active project"
)]
fn load_basic_project(world: &mut ApplicationWorld) {
    open_fixture_project(world, "fixtures/cad/document-creation-native-candidate.opj");
}

#[given(
    "its Metric catalog contains the native standard part, assembly, drawing, sheet-metal and presentation templates"
)]
fn verify_basic_catalog(world: &mut ApplicationWorld) {
    let state = world.state();
    for (title, kind) in [
        ("Standard (mm)", GuiTemplateKind::Part),
        ("Standard Assembly", GuiTemplateKind::Assembly),
        ("Standard Drawing", GuiTemplateKind::Drawing),
        ("Sheet Metal (mm)", GuiTemplateKind::SheetMetal),
        ("Standard Presentation", GuiTemplateKind::Presentation),
    ] {
        assert!(
            state
                .project_templates
                .iter()
                .any(|entry| entry.title == title
                    && entry.folder == GuiTemplateFolder::Units(GuiTemplateUnits::Metric)
                    && entry.kind == kind
                    && entry.length_unit == "mm"),
            "expected {title} in the Metric catalog: {:?}",
            state.project_template_issue
        );
    }
}

#[given("`fixtures/cad/workspace-layout-native.opd` is open and active in the MCAD Software")]
fn open_starting_part(world: &mut ApplicationWorld) {
    super::super::workspace_layout::native_part::open_named_extrusion(world);
}

pub(super) fn metric_title<'a>(
    world: &mut ApplicationWorld,
    built_in: &'a str,
    native: &'a str,
) -> &'a str {
    if world.state().project_templates.is_empty() {
        built_in
    } else {
        native
    }
}

pub(super) fn select_metric_template(world: &mut ApplicationWorld, built_in: &str, native: &str) {
    let title = metric_title(world, built_in, native);
    world.click_text(title);
}
