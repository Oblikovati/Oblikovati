// SPDX-License-Identifier: GPL-2.0-only
//! Feature Properties, Show Dimensions and a sketch's Dimension Visibility.
use super::find::{expect_control, open_list};
use super::*;

const TITLE: &str = "Feature Properties";
const ALWAYS: &str = "Always";
const IF: &str = "If";
const FROM_TO: &str = "From/To Planes";
const DEPTH: &str = "20";

fn open_properties(world: &mut ApplicationWorld) {
    right_click_row(world, FEATURE);
    choose_menu_row(world, "Properties");
    settle(world);
    world.expect_text(TITLE);
}
fn cancel(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    settle(world);
}

#[test]
fn feature_properties_title_close_discards_pending_suppression() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    open_properties(&mut world);
    toggle(&mut world, ALWAYS);
    expect_marks(&world, true, false);
    let (position, size) = diagnostics::window_geometry(TITLE).unwrap();
    let close = diagnostics::pointer_pixels([position[0] + size[0] - 13., position[1] + 10.]);
    world.click(close);
    assert!(diagnostics::window_geometry(TITLE).is_none());
    open_properties(&mut world);
    expect_marks(&world, false, false);
    cancel(&mut world);
    world.close();
}

#[test]
fn feature_properties_commits_suppression_and_native_undo_restores_the_solid() {
    use oblikovati_viewport::{ViewportKey, ViewportPhysicalKey};
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    open_properties(&mut world);
    toggle(&mut world, ALWAYS);
    world.click_text("OK");
    settle(&mut world);
    world.expect_no_text(TITLE);
    let status = world.app().window.native.scene_status().unwrap();
    assert_eq!(status.visible_bodies, 0);
    world.command_key(ViewportPhysicalKey::Z, ViewportKey::Undo);
    settle(&mut world);
    expect_solid_visible(&mut world);
    world.close();
}

fn toggle(world: &mut ApplicationWorld, label: &str) {
    world.click_text(label);
    settle(world);
}
fn expect_marks(world: &ApplicationWorld, always: bool, conditional: bool) {
    assert_eq!(world.checked(ALWAYS), always, "{ALWAYS}");
    assert_eq!(world.checked(IF), conditional, "{IF}");
}
fn expect_condition_controls(world: &ApplicationWorld, enabled: bool) {
    for label in ["Parameter", "Comparison"] {
        assert_eq!(world.run_is_enabled(label), enabled, "{label}");
    }
}
// Suppression
#[given("an unsuppressed extrusion is selected in the MCAD Software model browser")]
fn extrusion_selected(world: &mut ApplicationWorld) {
    open_block(world);
    world.click_text(FEATURE);
    settle(world);
}
#[when("I choose Properties from the extrusion context menu")]
#[when("the user chooses Properties from the extrusion context menu")]
#[when("I open Feature Properties from its browser context menu")]
#[when("the user opens Feature Properties from its browser context menu")]
fn from_menu(world: &mut ApplicationWorld) {
    open_properties(world);
}
#[then(
    "Feature Properties shows the feature name and Suppress, Adaptive and Feature Appearance controls"
)]
fn dialog_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &["Name", "Suppress", "Adaptive", "Feature Appearance"],
    );
    world.expect_text(FEATURE);
}
#[given("Always and If are unchecked")]
#[then("Always and If are unchecked")]
fn both_clear(world: &mut ApplicationWorld) {
    expect_marks(world, false, false);
}
#[then("the conditional suppression controls are disabled")]
#[then("the conditional controls are disabled")]
fn controls_off(world: &mut ApplicationWorld) {
    expect_condition_controls(world, false);
}
#[when("I check If")]
#[when("the user checks If")]
fn check_if(world: &mut ApplicationWorld) {
    toggle(world, IF);
}
#[then("the parameter selector, comparison operator and value field are enabled")]
#[then("the conditional controls are enabled")]
fn controls_on(world: &mut ApplicationWorld) {
    expect_condition_controls(world, true);
}
#[then("OK is disabled while the comparison value is empty")]
fn ok_off(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
}
#[when("I open the comparison operator list")]
#[when("the user opens the comparison operator list")]
fn open_operators(world: &mut ApplicationWorld) {
    open_list(world, "Comparison");
}
#[then(
    "it offers equals, not equals, less than, less than or equal, greater than and greater than or equal"
)]
fn operator_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &[
            "equals",
            "not equals",
            "less than",
            "less than or equal",
            "greater than",
            "greater than or equal",
        ],
    );
}
#[when("I choose greater than or equal")]
#[when("the user chooses greater than or equal")]
fn choose_operator(world: &mut ApplicationWorld) {
    toggle(world, "greater than or equal");
}
#[then("that operator is displayed")]
fn operator_shown(world: &mut ApplicationWorld) {
    expect_control(world, "Comparison", "greater than or equal");
}
#[when("I click Cancel and reopen Feature Properties")]
#[when("the user clicks Cancel and reopens Feature Properties")]
fn cancel_reopen(world: &mut ApplicationWorld) {
    cancel(world);
    world.expect_no_text(TITLE);
    open_properties(world);
}
#[then("If is unchecked and the conditional controls are disabled")]
fn if_clear(world: &mut ApplicationWorld) {
    assert!(!world.checked(IF), "expected {IF} clear");
    expect_condition_controls(world, false);
}
#[then("the dialog closes and the solid remains visible")]
fn closed_solid(world: &mut ApplicationWorld) {
    world.expect_no_text(TITLE);
    expect_solid_visible(world);
}
#[given("an unsuppressed extrusion in the MCAD Software has Feature Properties open")]
#[given("an extrusion in the MCAD Software has Feature Properties open")]
fn dialog_open(world: &mut ApplicationWorld) {
    open_block(world);
    open_properties(world);
}
#[when("I check Always")]
#[when("the user checks Always")]
#[when("I check Always again")]
#[when("the user checks Always again")]
fn check_always(world: &mut ApplicationWorld) {
    toggle(world, ALWAYS);
}
#[then("Always is checked and If is unchecked")]
fn always_only(world: &mut ApplicationWorld) {
    expect_marks(world, true, false);
}
#[then("If is checked and Always is unchecked")]
fn if_only(world: &mut ApplicationWorld) {
    expect_marks(world, false, true);
}
#[then("the conditional controls are disabled and OK is enabled")]
fn off_and_ok(world: &mut ApplicationWorld) {
    expect_condition_controls(world, false);
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}
// Adaptive
#[given("the Adaptive Sketch and Parameters options are unchecked")]
#[then("Sketch and Parameters are unchecked")]
fn adaptive_clear(world: &mut ApplicationWorld) {
    assert!(!world.checked("Sketch"), "expected Sketch clear");
    assert!(!world.checked("Parameters"), "expected Parameters clear");
}
#[given("From/To Planes is disabled for this extrusion")]
#[then("From/To Planes remains disabled")]
fn from_to_off(world: &mut ApplicationWorld) {
    assert!(
        !world.run_is_enabled(FROM_TO),
        "expected {FROM_TO} disabled"
    );
}
#[when("I check Sketch and Parameters")]
#[when("the user checks Sketch and Parameters")]
fn check_adaptive(world: &mut ApplicationWorld) {
    toggle(world, "Sketch");
    toggle(world, "Parameters");
}
#[then("both options are checked")]
fn adaptive_on(world: &mut ApplicationWorld) {
    assert!(world.checked("Sketch"), "expected Sketch checked");
    assert!(world.checked("Parameters"), "expected Parameters checked");
}
// Appearance
#[given("an extrusion in the MCAD Software has Feature Appearance set to As Body")]
fn as_body(world: &mut ApplicationWorld) {
    open_block(world);
}
#[when("I open the Feature Appearance list")]
#[when("the user opens the Feature Appearance list")]
fn open_appearances(world: &mut ApplicationWorld) {
    open_list(world, "Feature Appearance");
}
#[then("the list offers As Body and named appearances including Black")]
fn appearance_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["As Body", "Black"]);
}
#[then("Feature Appearance shows Black")]
fn shows_black(world: &mut ApplicationWorld) {
    expect_control(world, "Feature Appearance", "Black");
}
#[then("Feature Appearance shows As Body")]
fn shows_as_body(world: &mut ApplicationWorld) {
    expect_control(world, "Feature Appearance", "As Body");
}
// Show Dimensions
fn annotation_texts(world: &mut ApplicationWorld) -> Vec<String> {
    crate::HeadSketchWork::sketch_dimension_annotations(&world.app().owner)
        .unwrap()
        .labels
        .into_iter()
        .map(|label| label.text)
        .collect()
}
#[given("the feature dimensions are not displayed")]
fn no_annotations(world: &mut ApplicationWorld) {
    assert!(
        annotation_texts(world).is_empty(),
        "expected no annotations"
    );
}
#[when("I choose Show Dimensions from the extrusion row context menu")]
#[when("the user chooses Show Dimensions from the extrusion row context menu")]
fn show_dimensions(world: &mut ApplicationWorld) {
    right_click_row(world, FEATURE);
    choose_menu_row(world, "Show Dimensions");
    settle(world);
}
#[then("dimension annotations appear around the block")]
#[then("the selection highlight clears and the dimensions remain displayed")]
fn annotations_up(world: &mut ApplicationWorld) {
    let texts = annotation_texts(world);
    assert!(texts.iter().any(|text| text == DEPTH), "labels {texts:?}");
    assert!(texts.len() > 1, "labels {texts:?}");
    for text in texts {
        world.expect_text(&text);
    }
}
#[then("the dimensions disappear and the extrusion context menu opens")]
fn annotations_gone_menu_up(world: &mut ApplicationWorld) {
    assert!(
        annotation_texts(world).is_empty(),
        "expected no annotations"
    );
    world.expect_text("Show Dimensions");
}
#[then("the context menu closes and the shaded block remains visible")]
#[then("the shaded solid remains visible")]
fn menu_gone_block_stays(world: &mut ApplicationWorld) {
    world.expect_no_text("Show Dimensions");
    expect_solid_visible(world);
}
// Dimension Visibility
fn sketch_menu(world: &mut ApplicationWorld) {
    right_click_row(world, SKETCH);
}
#[given("the MCAD Software has a rectangular dimensioned sketch consumed by an extrusion")]
fn consumed_sketch(world: &mut ApplicationWorld) {
    open_block(world);
}
#[given("the sketch is hidden and its Dimension Visibility setting is enabled")]
fn sketch_hidden(world: &mut ApplicationWorld) {
    assert!(
        annotation_texts(world).is_empty(),
        "expected no annotations"
    );
    sketch_menu(world);
    assert!(
        world.checked("Dimension Visibility"),
        "expected the setting on"
    );
    world.escape();
    settle(world);
    click_empty_viewport(world);
}
#[when("I choose Visibility from the sketch row context menu")]
#[when("the user chooses Visibility from the sketch row context menu")]
fn visibility_from_row(world: &mut ApplicationWorld) {
    sketch_menu(world);
    choose_menu_row(world, "Visibility");
    settle(world);
}
#[when("I choose Visibility from the sketch context menu")]
#[when("the user chooses Visibility from the sketch context menu")]
fn visibility_from_menu(world: &mut ApplicationWorld) {
    if !world.shows("Dimension Visibility") {
        sketch_menu(world);
    }
    choose_menu_row(world, "Visibility");
    settle(world);
    click_empty_viewport(world);
}
#[then("the sketch is visible and its dimension annotations appear")]
#[then("the dimension annotations remain visible")]
#[then("the dimension annotations reappear")]
fn dimensions_shown(world: &mut ApplicationWorld) {
    settle(world);
    assert!(
        !annotation_texts(world).is_empty(),
        "expected the annotations"
    );
}
#[when("I reopen the sketch context menu")]
#[when("the user reopens the sketch context menu")]
fn reopen_menu(world: &mut ApplicationWorld) {
    sketch_menu(world);
}
#[then("Visibility and Dimension Visibility are checked")]
fn both_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Visibility"), "expected Visibility checked");
    assert!(
        world.checked("Dimension Visibility"),
        "expected Dimension Visibility checked"
    );
}
#[when("I choose Dimension Visibility")]
#[when("the user chooses Dimension Visibility")]
#[when("I choose Dimension Visibility again")]
#[when("the user chooses Dimension Visibility again")]
fn choose_dimension_visibility(world: &mut ApplicationWorld) {
    if !world.shows("Dimension Visibility") {
        sketch_menu(world);
    }
    choose_menu_row(world, "Dimension Visibility");
    settle(world);
}
#[then("the dimension annotations disappear while the sketch remains visible")]
fn dimensions_gone(world: &mut ApplicationWorld) {
    assert!(
        annotation_texts(world).is_empty(),
        "expected no annotations"
    );
    let browser = world.state().browser.expect("expected a browser");
    assert!(browser.sketches[0].visible, "expected the sketch visible");
}
#[then("Visibility is checked and Dimension Visibility is unchecked")]
fn one_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Visibility"), "expected Visibility checked");
    assert!(
        !world.checked("Dimension Visibility"),
        "expected Dimension Visibility clear"
    );
}
#[then("the sketch and its annotations are hidden again")]
fn all_hidden(world: &mut ApplicationWorld) {
    assert!(
        annotation_texts(world).is_empty(),
        "expected no annotations"
    );
    let browser = world.state().browser.expect("expected a browser");
    assert!(!browser.sketches[0].visible, "expected the sketch hidden");
}
