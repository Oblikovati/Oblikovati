// SPDX-License-Identifier: GPL-2.0-only
//! Save and reopen authored and resolved Sketch Text in the same desktop session.
use super::super::super::sketch::dimension_label_overlap::place_midpoint_dimensions;
use super::*;
use oblikovati_api::{
    client::Client,
    types::{
        Color, OVERRIDE_COLOR_SOURCE, SKETCH_TEXT_FIELD_FILE_NAME,
        SKETCH_TEXT_FIELD_MODEL_PARAMETER, SKETCH_TEXT_FIELD_PLACEHOLDER, SketchTextField,
        SketchTextFraction, SketchTextRun,
    },
    wire::AddTextArgs,
};
use std::path::{Path, PathBuf};
use text_linked_journey::capture_when_requested;

const FILE_NAME: &str = "Part6.opd";
const NUMBERED_TEXT: &str = "  1. a\n  2. b";
const EXPECTED_FREEDOMS: usize = 47;
const SKETCH_TWO_INDEX: usize = 1;

#[given(
    "Sketch2 contains both stacked fractions, numbered paragraphs, and mixed property/model Text"
)]
fn build_expanded_text_part(world: &mut ApplicationWorld) {
    world.add_part();
    world.add_finished_sketch();
    world.start_sketch();
    expect_sketch_two_active(world);
    place_midpoint_dimensions(world, "10");
    seed_expanded_text(world);
    assert_eq!(world.edited_geometry().texts().len(), 5);
    world.seed_dimensions_needed(EXPECTED_FREEDOMS);
    world.text_creation_baseline = Some(world.edited_geometry());
}

fn seed_expanded_text(world: &mut ApplicationWorld) {
    let numbered = AddTextArgs {
        text: NUMBERED_TEXT.into(),
        ..Default::default()
    };
    let placements = [
        ([810, 500], fraction_text(100.)),
        ([975, 500], fraction_text(50.)),
        ([810, 600], numbered),
        ([1110, 690], numeric_text()),
        ([975, 600], mixed_text()),
    ];
    for (pixels, mut request) in placements {
        request.sketch_index = SKETCH_TWO_INDEX as i64;
        request.anchor = Some(world.sketch_point_of(pixels).to_vec());
        request.height = "6.10 mm".into();
        request.font = "DejaVu Sans".into();
        Client::new(world.app().owner.document_host())
            .sketch()
            .add_text(&request)
            .unwrap();
    }
    world.pump(Vec::new());
}

fn fraction_text(width: f64) -> AddTextArgs {
    AddTextArgs {
        text: "1/2".into(),
        stacked_fractions: Some(vec![SketchTextFraction { start: 0, end: 3 }]),
        formatting_runs: Some(vec![SketchTextRun {
            end: 3,
            width_percent: Some(Box::new(width)),
            color: Some(Box::new(Color {
                r: 0,
                g: 128,
                b: 255,
                opacity: 1.,
                source: OVERRIDE_COLOR_SOURCE,
            })),
            ..Default::default()
        }]),
        ..Default::default()
    }
}

fn mixed_text() -> AddTextArgs {
    AddTextArgs {
        text: format!("{SKETCH_TEXT_FIELD_PLACEHOLDER}\n{SKETCH_TEXT_FIELD_PLACEHOLDER}"),
        linked_fields: Some(vec![
            SketchTextField {
                kind: SKETCH_TEXT_FIELD_FILE_NAME,
                ..Default::default()
            },
            model_field(2, Some(2)),
        ]),
        ..Default::default()
    }
}

fn numeric_text() -> AddTextArgs {
    AddTextArgs {
        text: SKETCH_TEXT_FIELD_PLACEHOLDER.into(),
        linked_fields: Some(vec![model_field(0, None)]),
        ..Default::default()
    }
}

fn model_field(at: u32, precision: Option<u8>) -> SketchTextField {
    SketchTextField {
        at,
        kind: SKETCH_TEXT_FIELD_MODEL_PARAMETER,
        parameter: "d0".into(),
        precision: precision.map(Box::new),
    }
}

#[when("the user finishes the sketch and save Part6")]
fn save_expanded_part6(world: &mut ApplicationWorld) {
    world.finish_sketch();
    let folder = world.scratch_folder();
    let document = world.active_id();
    let proposal = format!("{}.opd", world.document_name(document));
    x11_input::press_control_letter(b's');
    world.pump(Vec::new());
    world.enter_in_field("Save in", folder.to_str().unwrap());
    enter_part6_name(world, &proposal);
    let path = folder.join(FILE_NAME);
    assert!(path.is_file(), "expected saved Part6 at {path:?}");
    capture_portable_part(&path);
    world.saved_path = Some(path);
}

fn capture_portable_part(saved: &Path) {
    if let Some(reports) = std::env::var_os("OBK_BDD_REPORTS") {
        std::fs::copy(saved, PathBuf::from(reports).join(FILE_NAME)).unwrap();
    }
}

fn enter_part6_name(world: &mut ApplicationWorld, proposal: &str) {
    let input = world.run(proposal).center();
    x11_input::click(diagnostics::pointer_pixels(input));
    world.pump(Vec::new());
    x11_input::press_control_a();
    world.pump(Vec::new());
    for character in FILE_NAME.chars() {
        x11_input::press_path_character(character);
        world.pump(Vec::new());
    }
    world.expect_text(FILE_NAME);
    let save = world.run("Save").center();
    x11_input::click(diagnostics::pointer_pixels(save));
    world.pump(Vec::new());
    world.expect_no_text("Save in");
}

#[when("the user closes and reopen the saved document in the same application session")]
fn reopen_expanded_part(world: &mut ApplicationWorld) {
    let document = world.active_id();
    world.close_active_tab();
    assert!(!world.document_is_open(document));
    world.reopen_through_dialog(&world.saved_path.clone().unwrap());
    world.expect_text("Sketch2");
}

#[when("the user chooses Edit Sketch for Sketch2")]
fn edit_reopened_sketch_two(world: &mut ApplicationWorld) {
    let row = world.run("Sketch2").center();
    x11_input::right_click(diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    let edit = world.run("Edit Sketch").center();
    x11_input::click(diagnostics::pointer_pixels(edit));
    world.pump(Vec::new());
    expect_sketch_two_active(world);
    assert_eq!(
        &world.edited_geometry(),
        world.text_creation_baseline.as_ref().unwrap()
    );
}

fn expect_sketch_two_active(world: &mut ApplicationWorld) {
    let state = world.state();
    let sketch = &state.browser.unwrap().sketches[SKETCH_TWO_INDEX];
    assert_eq!(state.active_sketch, Some(sketch.id));
    world.expect_text("Sketch2");
}

#[then("both stacked fractions remain")]
fn both_fractions_remain(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    assert_eq!(
        geometry
            .texts()
            .iter()
            .filter(|text| text.style.stacked_fractions.is_some())
            .count(),
        2
    );
    for (text, width) in geometry.texts().iter().take(2).zip([100., 50.]) {
        text_fraction::assert_stacked(&text.style);
        assert_eq!(
            oblikovati_model::sketch::uniform_sketch_text_width(
                &text.style.content,
                text.style.formatting_runs.as_deref().unwrap()
            ),
            Some(width)
        );
    }
}

#[then("the two numbered paragraphs retain a and b with prefixes 1. and 2.")]
fn numbered_paragraphs_remain(world: &mut ApplicationWorld) {
    assert_eq!(
        world.edited_geometry().texts()[2].style.content,
        NUMBERED_TEXT
    );
}

#[then("the mixed field resolves to the current document filename and 10.00")]
fn mixed_field_resolves_after_reopen(world: &mut ApplicationWorld) {
    let shown = reopened_scene_text(world);
    assert_eq!(
        shown,
        ["1/2", "1/2", NUMBERED_TEXT, "10.000", "Part6.opd\n10.00"]
    );
    capture_when_requested(world, "expanded-sketch2-after-reopen");
}

#[then("the earlier linked field remains 10.000")]
fn earlier_field_resolves_after_reopen(world: &mut ApplicationWorld) {
    assert_eq!(reopened_scene_text(world)[3], "10.000");
}

fn reopened_scene_text(world: &mut ApplicationWorld) -> Vec<String> {
    let active = world.app().owner.active_sketch_edit().unwrap().sketch;
    let view = &world.app().window.native;
    assert!(view.sketch_display_declines().unwrap().is_empty());
    let scene = view
        .sketch_scene_status()
        .unwrap()
        .sketches
        .iter()
        .find(|scene| scene.sketch == active)
        .expect("expected visible Sketch2");
    scene
        .geometry
        .texts()
        .iter()
        .map(|text| text.style.content.clone())
        .collect()
}
