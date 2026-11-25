// SPDX-License-Identifier: GPL-2.0-only
//! Open the real populated drawing template through its project catalog.
use super::*;
use oblikovati_api::{
    client::Client,
    wire::{ActivateDocumentArgs, ParameterSetArgs, UpdateDocumentArgs},
};
use oblikovati_gui::{GuiTemplateFolder, GuiTemplateKind, GuiTemplateUnits};
use oblikovati_model::drawing::DrawingAnnotationDefinition;
use oblikovati_persistence::package::DocumentPackage;

const PROJECT: &str = "fixtures/cad/document-creation-layout-candidate.opj";
const PART: &str = "fixtures/cad/document-opening/OpenPart.opd";
const LAYOUT: &str = "fixtures/cad/document-creation-layout-candidate/English/Populated Layout.odd";
const UPDATED_PART: &str = "fixtures/cad/document-creation-layout-candidate/English/layout-data/extrusion/extrusion-rect-8mm.opd";
const ALTERNATE_PART: &str =
    "fixtures/cad/document-creation-layout-candidate/English/layout-data/alternate-component.opd";
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
    open_layout_part(world, &path);
}

fn open_layout_part(world: &mut ApplicationWorld, path: &std::path::Path) {
    world.open_fixture(path);
    let active = world.active_id();
    assert_eq!(
        world.state().document_type,
        Some(oblikovati_api::types::DOCUMENT_PART)
    );
    world.previous_document = Some(active);
    world.parts.push(active);
}

#[given(
    "a disposable copy of `fixtures/cad/document-creation-layout-candidate/English/layout-data/extrusion/extrusion-rect-8mm.opd` is active"
)]
fn open_disposable_layout_part(world: &mut ApplicationWorld) {
    open_disposable_part(world, UPDATED_PART);
}

fn open_disposable_part(world: &mut ApplicationWorld, relative: &str) {
    let source = super::super::super::scenario_root().join(relative);
    let copy = world.scratch_folder().join(source.file_name().unwrap());
    std::fs::copy(source, &copy).unwrap();
    open_layout_part(world, &copy);
}

#[given(
    "disposable copies of `fixtures/cad/document-creation-layout-candidate/English/layout-data/extrusion/extrusion-rect-8mm.opd` and `fixtures/cad/document-creation-layout-candidate/English/layout-data/alternate-component.opd` are open in the same session"
)]
fn open_both_layout_parts(world: &mut ApplicationWorld) {
    open_disposable_part(world, UPDATED_PART);
    open_disposable_part(world, ALTERNATE_PART);
    assert_ne!(world.parts[0], world.parts[1]);
    world.previous_document = Some(world.parts[0]);
}

#[given("each part has been updated and then edited so that it needs updating again")]
fn mark_both_layout_parts(world: &mut ApplicationWorld) {
    mark_part_update_needed(world);
    let first = world.parts[0];
    Client::new(world.app().owner.document_host())
        .documents()
        .activate(&ActivateDocumentArgs { id: first })
        .unwrap();
    mark_part_update_needed(world);
}

#[given("the file-selection dialog can access the alternate part")]
fn alternate_layout_part_exists(world: &mut ApplicationWorld) {
    let path = world.scratch_folder().join("alternate-component.opd");
    assert!(path.is_file(), "expected the alternate Part at {path:?}");
    world.saved_path = Some(path);
}

#[given("the active part has been updated and then edited so that it needs updating again")]
fn mark_part_update_needed(world: &mut ApplicationWorld) {
    let client = Client::new(world.app().owner.document_host());
    let update = client
        .documents()
        .update(&UpdateDocumentArgs::default())
        .unwrap();
    assert!(!update.requires_update && update.errors.is_none());
    client
        .parameters()
        .add(&ParameterSetArgs {
            name: "layout_probe".into(),
            expression: "10 mm".into(),
            value_type: "numeric".into(),
            ..Default::default()
        })
        .unwrap();
    // The next native frame can evaluate the edit, so check the update status here.
    assert!(
        client
            .documents()
            .requires_update()
            .unwrap()
            .requires_update
    );
}

#[given(
    "`fixtures/cad/document-creation-layout-candidate/English/Populated Layout.odd` is available as the populated inch assembly layout"
)]
fn populated_layout_is_available(world: &mut ApplicationWorld) {
    saved_layout_is_available(world);
    world.click_text("English");
    assert!(gallery_has_template(world, TITLE));
}

#[given(
    "`fixtures/cad/document-creation-layout-candidate/English/Populated Layout.odd` is available as a drawing layout with a view and parts list"
)]
fn saved_layout_is_available(world: &mut ApplicationWorld) {
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
}

#[given("the user selects the assembly layout and opens its component-selection dialog")]
fn open_layout_component_dialog(world: &mut ApplicationWorld) {
    select_populated_layout(world);
    create_part(world);
    world.pump(Vec::new());
    world.expect_text("Select Component");
}

#[when("the user selects the drawing template and populated layout")]
fn select_populated_layout(world: &mut ApplicationWorld) {
    open_chooser(world);
    world.click_text("English");
    world.click_text(TITLE);
    world.click_text("Assembly Layout");
}

#[given("the dialog displays the active part")]
#[then("the component-selection dialog displays the proposed component")]
pub(super) fn expect_active_component_name(world: &mut ApplicationWorld) {
    let id = world.active_id();
    let expected = world.document_name(id).replace(' ', "");
    let label = world.run("Document name");
    // Adjacent glyphs can be grouped into one diagnostic run.
    let joined = format!("Documentname{expected}");
    assert!(
        label.text == joined
            || world.frame.runs.iter().any(|run| {
                run.list == label.list
                    && run.minimum[0] > label.maximum[0]
                    && (run.center()[1] - label.center()[1]).abs() < 4.
                    && run.text == expected
            }),
        "expected the component field to display {expected:?}: {}",
        world.text()
    );
}

#[when("the user confirms that component in the component-selection dialog")]
#[when("the user confirms the selected component")]
fn confirm_component(world: &mut ApplicationWorld) {
    let button = world.run_in_window("Select Component", "OK");
    let point = super::super::super::diagnostics::pointer_pixels(button.center());
    world.click(point);
    world.pump(Vec::new());
}

#[then("a message identifies the component as needing an update")]
fn component_needs_update(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected the proposed Part");
    let name = world.document_name(part).replace(' ', "");
    if !world.shows("needs updating") {
        let freshness = Client::new(world.app().owner.document_host())
            .documents()
            .requires_update();
        panic!(
            "expected an update warning for {name:?}; active document update status: {freshness:?}"
        );
    }
    expect_warning_component(world, &name);
}

#[then("a message identifies the selected component as needing an update")]
fn selected_component_needs_update(world: &mut ApplicationWorld) {
    expect_warning_component(world, "alternate-component");
}

fn expect_warning_component(world: &ApplicationWorld, name: &str) {
    let warning = world.run("needs updating");
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.list == warning.list && run.text.contains(name)),
        "expected the update warning to identify {name:?}"
    );
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
                    .filter(|pixel| {
                        let rgb = &pixel[..3];
                        // Both palettes use light neutral paper, with dark ink inside it.
                        rgb.iter().all(|&channel| channel >= 220)
                    })
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

#[test]
fn a_part_backed_drawing_displays_its_authored_parts_list() {
    let mut world = ApplicationWorld::new();
    super::super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    let part = world.active_id();
    open_part_backed_layout(&mut world);
    world.expect_text("PART NUMBER");
    world.expect_text("Parts List1");
    world.expect_text("Front");
    expect_part_row(&mut world, "extruded");

    set_part_list_metadata(&mut world, part);
    world.pump(Vec::new());
    expect_part_row(&mut world, "LAYOUT-08");
    world.expect_text("LAYOUT-08");
    world.expect_text("Layout bracket");
    world.save_capture("part-layout-parts-list");
    world.close();
}

fn open_part_backed_layout(world: &mut ApplicationWorld) {
    use oblikovati_model::drawing::{DrawingBaseViewDefinition, DrawingDocument};
    let mut drawing = DrawingDocument::new();
    drawing.set_model_reference("extruded.opd").unwrap();
    let front = DrawingBaseViewDefinition::new(
        "Front",
        oblikovati_api::types::BASE_VIEW_FRONT,
        1.,
        [100., 100.],
    )
    .unwrap();
    drawing.add_base_view_definition(0, front).unwrap();
    let list = DrawingAnnotationDefinition::parts_list("Parts List1", [210., 170.]).unwrap();
    drawing.add_annotation_definition(0, list).unwrap();
    let bytes = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 3\n")
        .unwrap()
        .with_drawing(Some(&drawing))
        .unwrap()
        .encode()
        .unwrap();
    let copy = world.scratch_folder().join("part-layout.odd");
    std::fs::write(&copy, bytes).unwrap();
    world.open_fixture(&copy);
}

fn expect_part_row(world: &mut ApplicationWorld, part_number: &str) {
    let drawing = world.active_id();
    let state = world.state();
    let projection = &state.drawing_projections[&drawing];
    assert_eq!(projection.error, None);
    assert!(!projection.lines.is_empty());
    let [table] = projection.parts_lists.as_slice() else {
        panic!(
            "expected one authored parts list: {:?}",
            projection.parts_lists
        );
    };
    assert_eq!(table.row_count, 1);
    let row: Vec<_> = table.labels[4..]
        .iter()
        .map(|label| label.text.as_str())
        .collect();
    assert_eq!([row[0], row[1], row[3]], ["1", part_number, "1"]);
}

fn set_part_list_metadata(world: &mut ApplicationWorld, part: u64) {
    use oblikovati_api::{
        types::{Variant, VariantValue},
        wire::SetPropertyArgs,
    };
    let client = Client::new(world.app().owner.document_host());
    for (name, text) in [
        ("Part Number", "LAYOUT-08"),
        ("Description", "Layout bracket"),
    ] {
        client
            .documents()
            .set_property(&SetPropertyArgs {
                document: part,
                set: oblikovati_model::property::DESIGN_TRACKING.into(),
                name: name.into(),
                value: Variant {
                    value: VariantValue::String(text.into()),
                    ..Default::default()
                },
            })
            .unwrap();
    }
}
