// SPDX-License-Identifier: GPL-2.0-only
//! Feature rows in the part browser: extended names, the row's menu and the solid's outline.
use super::super::ApplicationWorld;
use super::browser_tree::expand_display_preferences;
use crate::scene_fixture as fixture;
use cucumber::{given, then, when};
use oblikovati_api::{client::Client, wire::OpenDocumentArgs};
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, sketch_prism::SketchPrismDefinition, work_geometry::DatumLength,
    },
    units::UnitsOfMeasure,
};
use oblikovati_persistence::package::DocumentPackage;

pub(in super::super) const FEATURE: &str = "Extrude1";
/// 20 mm in working units.
const DEPTH: f64 = 2.;
const EMPTY_VIEWPORT: [i32; 2] = [1300, 800];
/// The body's flat display colour and the outline's accent, as drawn.
pub(in super::super) const SOLID_COLOR: [u8; 3] = [0, 255, 0];
const EDGE_COLOR: [u8; 3] = [64, 153, 255];
const COLOR_REACH: i32 = 40;
const DIMENSION_LABEL_STEP: f64 = 1.5;
/// How far from a row its menu's first command opens.
const MENU_REACH: f32 = 40.;
pub(in super::super) const SKETCH: &str = "Profile";
pub(in super::super) const BODY: &str = "Solid1";

/// The fixture stores its dimensions without label positions; a shown dimension needs one.
fn place_dimension_labels(snapshot: &mut oblikovati_model::history::DocumentSnapshot) {
    let mut geometry = snapshot
        .sketches
        .get(fixture::SKETCH)
        .unwrap()
        .geometry()
        .clone();
    let dimensions: Vec<_> = geometry
        .dimensional_constraints()
        .map(|(constraint, _)| constraint.id)
        .collect();
    for (index, dimension) in dimensions.into_iter().enumerate() {
        let position = [index as f64 * DIMENSION_LABEL_STEP, -1.];
        geometry
            .set_dimension_text_point(dimension, position)
            .unwrap();
    }
    snapshot
        .sketches
        .replace_geometry(fixture::SKETCH, geometry)
        .unwrap();
}
/// The fixture profile, stored with one 20 mm extrusion; this build has no command that makes one.
fn extruded_part_bytes(name: &str) -> Vec<u8> {
    let mut recipe = fixture::selected_prism_definition().recipe();
    recipe.displacement[2] = DatumLength::Constant(DEPTH);
    let definition = SketchPrismDefinition::from_recipe(&recipe).unwrap();
    let mut snapshot = fixture::selected_sketch_document();
    place_dimension_labels(&mut snapshot);
    let feature = AuthoredFeature::sketch_prism(FeatureId(1), name, definition, &snapshot);
    snapshot.work_geometry.append(feature.unwrap()).unwrap();
    // Making the feature takes its sketch out of view; only its browser row calls it back.
    snapshot
        .sketches
        .set_visible(fixture::SKETCH, false)
        .unwrap();
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")
        .unwrap()
        .with_parameters(&snapshot.parameters, UnitsOfMeasure::default())
        .unwrap()
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap()
        .with_work_geometry(&snapshot.work_geometry, 1)
        .unwrap()
        .encode()
        .unwrap()
}
pub(in super::super) fn open_extruded_part(world: &mut ApplicationWorld) {
    open_extruded_part_named(world, FEATURE);
}
pub(in super::super) fn open_extruded_part_named(world: &mut ApplicationWorld, name: &str) {
    open_extruded_part_with_visibility(world, name, false);
}
pub(in super::super) fn open_visible_extruded_part(world: &mut ApplicationWorld) {
    open_extruded_part_with_visibility(world, FEATURE, true);
}
fn open_extruded_part_with_visibility(world: &mut ApplicationWorld, name: &str, visible: bool) {
    let path = world.scratch_folder().join("extruded.opd");
    std::fs::write(&path, extruded_part_bytes(name)).unwrap();
    Client::new(world.app().owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            visible,
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    world.pump(Vec::new());
}
pub(in super::super) fn colored_pixels(world: &mut ApplicationWorld, wanted: [u8; 3]) -> usize {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .filter(|pixel| {
            (0..3)
                .all(|channel| (pixel[channel] as i32 - wanted[channel] as i32).abs() < COLOR_REACH)
        })
        .count()
}
pub(in super::super) fn expect_solid_visible(world: &mut ApplicationWorld) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(
        super::super::solid_visibility::visible_solid_fill(&capture, &windows),
        "expected a shaded solid outside the GUI windows, not only an outline or a selected row"
    );
}
fn expect_row_and_edges_highlighted(world: &mut ApplicationWorld) {
    assert!(
        world.row_highlighted(FEATURE),
        "expected the {FEATURE} row's accent"
    );
    assert!(
        colored_pixels(world, EDGE_COLOR) > 0,
        "expected the solid's edges outlined"
    );
}
pub(in super::super) fn right_click_row(world: &mut ApplicationWorld, label: &str) {
    let at = world.run(label).center().map(|value| value as i32);
    world.right_click(at);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn right_click_feature(world: &mut ApplicationWorld) {
    right_click_row(world, FEATURE);
}
/// A branch's arrow sits just left of its text, and folds it without choosing the row.
pub(in super::super) fn click_branch_arrow(world: &mut ApplicationWorld, label: &str) {
    let run = world.run(label).clone();
    world.click([run.minimum[0] as i32 - 10, run.center()[1] as i32]);
    world.pump(Vec::new());
}
pub(in super::super) fn choose_menu_row(world: &mut ApplicationWorld, label: &str) {
    world.click_text(label);
    world.pump(Vec::new());
}
pub(super) fn expect_menu_beside(world: &ApplicationWorld, row_label: &str, first_row: &str) {
    let row = world.run(row_label).clone();
    let first = nearest_matching_run(&world.frame.runs, first_row, row.center())
        .unwrap_or_else(|| panic!("expected menu entry {first_row:?} beside row {row:?}"));
    assert!(
        first.minimum[0] >= row.minimum[0],
        "menu {first:?} beside row {row:?}"
    );
    assert!(
        (first.center()[1] - row.center()[1]).abs() < MENU_REACH,
        "menu {first:?} beside row {row:?}"
    );
}
pub(in super::super) fn expect_rows(world: &ApplicationWorld, rows: &[&str]) {
    for row in rows {
        world.expect_text(row);
    }
}
pub(in super::super) fn expect_outline(world: &mut ApplicationWorld, shown: bool) {
    let outlined = colored_pixels(world, EDGE_COLOR);
    assert_eq!(outlined > 0, shown, "outline pixels {outlined}");
}
pub(in super::super) fn click_empty_viewport(world: &mut ApplicationWorld) {
    world.click(EMPTY_VIEWPORT);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
// Show and hide extended feature names in the browser
#[given("a part with a 20 mm new-solid extrusion is active in the MCAD Software")]
#[given("a part with a solid extrusion is active in the MCAD Software")]
fn extruded_part(world: &mut ApplicationWorld) {
    open_extruded_part(world);
    expect_solid_visible(world);
}
#[given("Show Extended Names is unchecked in browser Display Preferences")]
fn unchecked(world: &mut ApplicationWorld) {
    expand_display_preferences(world);
    assert!(
        !world.checked("Show Extended Names"),
        "expected it unchecked"
    );
}
#[given("the extrusion is displayed with its short feature name")]
#[then("the extrusion is displayed with its short feature name")]
fn short_name(world: &mut ApplicationWorld) {
    world.expect_text(FEATURE);
    world.expect_no_text("NewSolid");
}
#[when("I select Show Extended Names")]
#[when("I select Show Extended Names again")]
fn select_extended_names(world: &mut ApplicationWorld) {
    world.click_text("Show Extended Names");
    world.pump(Vec::new());
}
#[then("the extrusion's browser label includes its new-solid operation and 20 mm depth")]
fn extended_label(world: &mut ApplicationWorld) {
    world.expect_text("Extrude1(NewSolid,20mm)");
}
#[then("the solid remains visible in the viewport")]
fn solid_visible(world: &mut ApplicationWorld) {
    expect_solid_visible(world);
}
#[then("Show Extended Names is checked")]
fn checked(world: &mut ApplicationWorld) {
    assert!(
        world.menu_row_checked("Show Extended Names"),
        "expected it checked"
    );
}
// Inspect the context menu for a browser feature
#[given("its extrusion feature is visible in the model browser")]
fn feature_visible(world: &mut ApplicationWorld) {
    world.expect_text(FEATURE);
}
#[when("I right-click the extrusion feature in the browser")]
fn right_click_extrusion_row(world: &mut ApplicationWorld) {
    right_click_feature(world);
}
#[then("a context menu opens beside the feature row")]
fn menu_beside(world: &mut ApplicationWorld) {
    expect_menu_beside(world, FEATURE, "Edit Sketch");
}
#[then("the menu offers Edit Sketch, Edit Feature and Show Dimensions")]
fn first_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["Edit Sketch", "Edit Feature", "Show Dimensions"]);
}
#[then("it offers Copy, Delete, Measure and Properties")]
fn second_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["Copy", "Delete", "Measure", "Properties"]);
}
#[then("it offers Expand All Children and Collapse All Children")]
#[then("its context menu offers Expand All Children and Collapse All Children")]
fn third_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["Expand All Children", "Collapse All Children"]);
}
#[then("the extrusion row and the solid edges are highlighted")]
#[then("the extrusion row and solid edges remain highlighted")]
fn highlighted(world: &mut ApplicationWorld) {
    expect_row_and_edges_highlighted(world);
}
#[when("I click the empty viewport area again")]
fn click_viewport(world: &mut ApplicationWorld) {
    click_empty_viewport(world);
}
// Expand and collapse an extrusion's browser children
#[given("a part with an extrusion and its underlying sketch is active in the MCAD Software")]
#[given(
    "a part with an extrusion and its underlying dimensioned sketch is active in the MCAD Software"
)]
fn part_with_sketch(world: &mut ApplicationWorld) {
    open_extruded_part(world);
    expect_solid_visible(world);
}
#[given("the extrusion branch is collapsed in the model browser")]
fn extrusion_collapsed(world: &mut ApplicationWorld) {
    click_branch_arrow(world, FEATURE);
    world.expect_no_text(SKETCH);
}
#[when("I right-click the extrusion and choose Expand All Children")]
fn expand_extrusion(world: &mut ApplicationWorld) {
    right_click_feature(world);
    choose_menu_row(world, "Expand All Children");
}
#[then("the extrusion branch displays its sketch child")]
#[then("the extrusion branch still displays its direct sketch child")]
fn sketch_child(world: &mut ApplicationWorld) {
    world.expect_text(SKETCH);
}
#[when("I right-click the extrusion and choose Collapse All Children")]
fn collapse_extrusion(world: &mut ApplicationWorld) {
    right_click_feature(world);
    choose_menu_row(world, "Collapse All Children");
}
// Inspect the context menu for a sketch under an extrusion
#[given("the extrusion branch is expanded in the model browser")]
fn extrusion_expanded(world: &mut ApplicationWorld) {
    world.expect_text(SKETCH);
}
#[when("I right-click the sketch child in the browser")]
fn right_click_sketch(world: &mut ApplicationWorld) {
    right_click_row(world, SKETCH);
}
#[then("a context menu opens beside the sketch row")]
fn sketch_menu_beside(world: &mut ApplicationWorld) {
    expect_menu_beside(world, SKETCH, "Edit Sketch");
}
#[then("the menu offers Edit Sketch, Redefine, Share Sketch and Properties")]
fn sketch_first_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &["Edit Sketch", "Redefine", "Share Sketch", "Properties"],
    );
}
#[then("it offers Measure, Visibility and Export Sketch As")]
fn sketch_more_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["Measure", "Visibility", "Export Sketch As"]);
}
#[then("the sketch row is highlighted")]
fn sketch_row_highlighted(world: &mut ApplicationWorld) {
    assert!(
        world.row_highlighted(SKETCH),
        "expected the sketch row's accent"
    );
}
fn sketch_dimension_texts(world: &mut ApplicationWorld) -> Vec<String> {
    let annotations = crate::HeadWork::sketch_dimension_annotations(&world.app().owner).unwrap();
    annotations
        .labels
        .into_iter()
        .map(|label| label.text)
        .collect()
}
fn expect_sketch_highlight(world: &mut ApplicationWorld) {
    expect_outline(world, true);
    let texts = sketch_dimension_texts(world);
    assert!(
        !texts.is_empty(),
        "expected the sketch's dimensions over the solid"
    );
    for text in texts {
        world.expect_text(&text);
    }
}
#[then("the sketch geometry and dimensions are highlighted over the solid")]
fn sketch_highlighted(world: &mut ApplicationWorld) {
    expect_sketch_highlight(world);
}
#[then("the sketch row, geometry and dimensions remain highlighted")]
fn sketch_remains(world: &mut ApplicationWorld) {
    sketch_row_highlighted(world);
    expect_sketch_highlight(world);
}
#[then("the sketch row is no longer highlighted")]
fn sketch_row_plain(world: &mut ApplicationWorld) {
    assert!(
        !world.row_highlighted(SKETCH),
        "expected the sketch row plain"
    );
}
#[then("the highlighted sketch geometry and dimensions disappear from the viewport")]
fn sketch_gone(world: &mut ApplicationWorld) {
    expect_outline(world, false);
    let texts = sketch_dimension_texts(world);
    assert!(
        texts.is_empty(),
        "expected the sketch's dimensions gone, read {texts:?}"
    );
}
#[then("the solid remains visible")]
fn solid_remains(world: &mut ApplicationWorld) {
    expect_solid_visible(world);
}
// Expand and collapse nested entries in the solid-bodies folder
fn runs_reading(world: &ApplicationWorld, text: &str) -> usize {
    world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == text)
        .count()
}
#[given(
    "a part with one solid body made from an extrusion and sketch is active in the MCAD Software"
)]
fn part_with_body(world: &mut ApplicationWorld) {
    open_extruded_part(world);
    expect_solid_visible(world);
}
#[given("the Solid Bodies folder is collapsed in the model browser")]
fn bodies_collapsed(world: &mut ApplicationWorld) {
    world.expect_text("Solid Bodies");
    world.expect_no_text(BODY);
}
#[when("I right-click Solid Bodies")]
fn right_click_bodies(world: &mut ApplicationWorld) {
    right_click_row(world, "Solid Bodies");
}
#[then("its context menu offers Visibility, Expand All Children and Collapse All Children")]
fn bodies_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &["Visibility", "Expand All Children", "Collapse All Children"],
    );
}
#[then("the folder displays the body row")]
#[then("the body row remains visible beneath the folder")]
fn body_shown(world: &mut ApplicationWorld) {
    world.expect_text(BODY);
}
#[then("the body displays its extrusion and underlying sketch as nested rows")]
fn body_nests(world: &mut ApplicationWorld) {
    // The feature and its sketch also sit in the Features branch: these are the second rows.
    assert_eq!(runs_reading(world, FEATURE), 2, "{FEATURE} rows");
    assert_eq!(runs_reading(world, SKETCH), 2, "{SKETCH} rows");
}
#[when("I right-click Solid Bodies and choose Collapse All Children")]
fn collapse_bodies(world: &mut ApplicationWorld) {
    right_click_bodies(world);
    choose_menu_row(world, "Collapse All Children");
}
#[then("the extrusion and sketch rows beneath that body are hidden")]
fn body_hides(world: &mut ApplicationWorld) {
    assert_eq!(runs_reading(world, FEATURE), 1, "{FEATURE} rows");
    assert_eq!(runs_reading(world, SKETCH), 1, "{SKETCH} rows");
}

// The selection toolbar repeats menu labels; draw order does not identify the menu.
fn nearest_matching_run<'a>(
    runs: &'a [super::super::diagnostics::RenderedRun],
    label: &str,
    anchor: [f32; 2],
) -> Option<&'a super::super::diagnostics::RenderedRun> {
    let wanted = label.replace(' ', "");
    let distance = |run: &super::super::diagnostics::RenderedRun| {
        let center = run.center();
        (center[0] - anchor[0]).hypot(center[1] - anchor[1])
    };
    runs.iter()
        .filter(|run| run.text == wanted)
        .min_by(|left, right| distance(left).total_cmp(&distance(right)))
}

#[test]
fn context_entry_is_not_the_duplicate_selection_toolbar_button() {
    use super::super::diagnostics::RenderedRun;
    let entry = RenderedRun {
        text: "EditSketch".into(),
        minimum: [95., 480.],
        maximum: [170., 490.],
        ..Default::default()
    };
    let toolbar = RenderedRun {
        minimum: [733., 287.],
        maximum: [808., 296.],
        ..entry.clone()
    };
    let runs = [entry.clone(), toolbar];
    assert_eq!(
        nearest_matching_run(&runs, "Edit Sketch", [86., 468.]),
        Some(&entry)
    );
    assert!(nearest_matching_run(&runs, "Properties", [86., 468.]).is_none());
}
