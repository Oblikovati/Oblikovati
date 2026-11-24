// SPDX-License-Identifier: GPL-2.0-only
//! Open the real populated drawing template through its project catalog.
use super::*;
use oblikovati_gui::{GuiTemplateFolder, GuiTemplateKind, GuiTemplateUnits};
use oblikovati_model::drawing::DrawingAnnotationDefinition;
use oblikovati_persistence::package::DocumentPackage;

const PROJECT: &str = "fixtures/cad/document-creation-layout-candidate.opj";
const PART: &str = "fixtures/cad/document-opening/OpenPart.opd";
const LAYOUT: &str = "fixtures/cad/document-creation-layout-candidate/English/Populated Layout.odd";
pub(super) const TITLE: &str = "Populated Layout";

#[given(
    "a fresh MCAD Software session loads `fixtures/cad/document-creation-layout-candidate.opj` as its active project"
)]
fn load_layout_project(world: &mut ApplicationWorld) {
    open_fixture_project(world, PROJECT);
}

#[given("the active document is `fixtures/cad/document-opening/OpenPart.opd` in the MCAD Software")]
fn open_part_for_layout(world: &mut ApplicationWorld) {
    let path = super::super::super::scenario_root().join(PART);
    world.open_fixture(&path);
    let active = world.active_id();
    assert_eq!(
        world.state().document_type,
        Some(oblikovati_api::types::DOCUMENT_PART)
    );
    world.previous_document = Some(active);
    world.parts.push(active);
}

#[given(
    "`fixtures/cad/document-creation-layout-candidate/English/Populated Layout.odd` is available as the populated inch assembly layout"
)]
fn populated_layout_is_available(world: &mut ApplicationWorld) {
    let path = super::super::super::scenario_root()
        .join(LAYOUT)
        .canonicalize()
        .unwrap();
    assert!(world.state().project_templates.iter().any(|entry| {
        entry.title == TITLE
            && entry.path == path
            && entry.kind == GuiTemplateKind::Drawing
            && entry.folder == GuiTemplateFolder::Units(GuiTemplateUnits::Inch)
            && entry.length_unit == "in"
    }));
    let package = DocumentPackage::decode(&std::fs::read(path).unwrap()).unwrap();
    let drawing = package.restore_drawing().unwrap().unwrap();
    let sheet = &drawing.sheets()[drawing.active_sheet()];
    assert!(!sheet.views.is_empty());
    assert!(
        sheet
            .annotations
            .iter()
            .any(|annotation| matches!(annotation, DrawingAnnotationDefinition::PartsList { .. }))
    );
    world.click_text("English");
    assert!(gallery_has_template(world, TITLE));
}

pub(super) fn expect_sheet_previews(world: &mut ApplicationWorld) {
    let ranges = ["Use Template", "Assembly Layout"].map(|label| {
        let run = world.run(label);
        let y = run.minimum[1] as usize;
        let x = run.maximum[0] as usize + 8;
        (label, y, x)
    });
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let width = capture.extent[0] as usize;
    let height = capture.extent[1] as usize;
    for (label, y, x) in ranges {
        let paper_pixels = (y..(y + 32).min(height))
            .map(|row| {
                capture.rgba[(row * width + x) * 4..(row * width + (x + 185).min(width)) * 4]
                    .as_chunks::<4>()
                    .0
                    .iter()
                    .filter(|pixel| pixel[..3] == [237, 237, 232])
                    .count()
            })
            .sum::<usize>();
        assert!(
            paper_pixels > 100,
            "{label}: expected a visible sheet thumbnail"
        );
    }
    world.save_capture("drawing-template-previews");
}
