// SPDX-License-Identifier: GPL-2.0-only
//! `features/application/model-browser`: the part's tree, its menus and its dialogs.
use super::viewport_pan::camera;
use super::workspace_layout::browser_feature::{
    BODY, FEATURE, SKETCH, choose_menu_row, click_branch_arrow, click_empty_viewport,
    expect_outline, expect_rows, expect_solid_visible, open_extruded_part,
    open_extruded_part_named, right_click_row,
};
use super::workspace_layout::browser_origin::{plane_shows_at, row_accent};
use super::workspace_layout::browser_tree::{
    expand_display_preferences, open_browser_menu, type_in_search,
};
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};

#[cfg(target_os = "linux")]
#[path = "feature_outline_native.rs"]
mod feature_outline_native;
#[path = "feature_properties.rs"]
mod feature_properties;
#[path = "feature_rows.rs"]
mod feature_rows;
#[path = "find.rs"]
mod find;
#[path = "find_lists.rs"]
mod find_lists;
#[path = "line_properties.rs"]
mod line_properties;
#[path = "model_state_copy.rs"]
mod model_state_copy;
#[path = "model_state_scope.rs"]
mod model_state_scope;
#[path = "model_state_suppression.rs"]
mod model_state_suppression;
#[path = "model_state_table.rs"]
mod model_state_table;
#[cfg(target_os = "linux")]
#[path = "model_state_table_extra.rs"]
mod model_state_table_extra;
#[cfg(target_os = "linux")]
#[path = "model_state_table_parameter_steps.rs"]
mod model_state_table_parameter_steps;
#[cfg(target_os = "linux")]
#[path = "model_state_table_row_steps.rs"]
mod model_state_table_row_steps;
#[cfg(target_os = "linux")]
#[path = "model_state_table_verify_steps.rs"]
mod model_state_table_verify_steps;
#[path = "model_states.rs"]
mod model_states;
#[path = "origin_rows.rs"]
mod origin_rows;
#[path = "panes.rs"]
mod panes;
#[path = "relationships.rs"]
mod relationships;

const ORIGIN_RESULT_ROW: &str = "XY Plane";
const ORIGIN_ROW_NAMES: [&str; 7] = [
    "YZ Plane",
    "XZ Plane",
    "XY Plane",
    "X Axis",
    "Y Axis",
    "Z Axis",
    "Center Point",
];

pub(super) fn settle(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    world.pump(Vec::new());
}
pub(super) fn open_block(world: &mut ApplicationWorld) {
    open_extruded_part(world);
    world.observed_view_camera = Some(camera(world));
}
pub(super) fn expect_block_kept(world: &mut ApplicationWorld) {
    expect_solid_visible(world);
    let earlier = world
        .observed_view_camera
        .expect("expected the earlier view");
    assert_eq!(camera(world), earlier, "view");
}
/// Rows in one run are read together: adjacent text on a baseline renders as one.
pub(super) fn all_text(world: &ApplicationWorld) -> String {
    world
        .frame
        .runs
        .iter()
        .map(|run| run.text.as_str())
        .collect()
}
fn expect_origin_rows(world: &ApplicationWorld) {
    expect_rows(world, &ORIGIN_ROW_NAMES);
}
fn expect_no_origin_rows(world: &ApplicationWorld) {
    for row in ORIGIN_ROW_NAMES {
        world.expect_no_text(row);
    }
}
#[given("an active part in the MCAD Software has a visible model browser")]
#[given(
    "an active part in the MCAD Software contains a visible solid and has a visible model browser"
)]
#[given(
    "an active part in the MCAD Software contains a visible block made by an extrusion with a nested sketch"
)]
#[given("an active part in the MCAD Software contains a visible block made by an extrusion")]
#[given("an active saved part in the MCAD Software contains one visible solid block")]
#[given("The MCAD Software shows a visible block made by an extrusion")]
#[given("an active part in the MCAD Software contains a single block created by an extrusion")]
#[given("an active part in the MCAD Software contains an extrusion with a nested sketch")]
#[given("an active part in the MCAD Software contains a block created by an extrusion")]
#[given(
    "an active part in the MCAD Software has expanded representation, origin and extrusion branches"
)]
#[given("an active part in the MCAD Software contains a 20 mm new-solid extrusion")]
#[given("an active part is open in the MCAD Software")]
#[given("an active part is visible in the MCAD Software")]
#[given("an active part with a visible block is open in the MCAD Software")]
#[given(
    "an active part in the MCAD Software contains a dimensioned rectangular sketch used by an extrusion"
)]
#[given("a dimensioned sketch is nested under an extrusion in the MCAD Software")]
#[given(
    "an active part in the MCAD Software contains a single extrusion forming a rectangular block"
)]
#[given("an active part in the MCAD Software contains one visible solid block")]
#[given(
    "an active part in the MCAD Software has one solid body made by an extrusion from a sketch"
)]
#[given("the MCAD Software displays the single-block part fixture")]
#[given("a part with a visible block is open in the MCAD Software")]
#[given("a part is open in the MCAD Software with its model browser visible")]
#[given("an active part in the MCAD Software contains one extrusion forming a block")]
#[given("a part with one extrusion and no fillet is open in the MCAD Software")]
#[given("a part with one unsuppressed extrusion and no fillet is open in the MCAD Software")]
#[given("a nested sketch in the MCAD Software has Default line color")]
#[given("a nested sketch in the MCAD Software has Default line weight")]
#[given("a nested sketch in the MCAD Software has Default line type")]
#[given(
    "a part in the MCAD Software contains a rectangular sketch on the XY plane used by one extrusion"
)]
#[given(
    "a part in the MCAD Software contains a sketch on the XY plane and an extrusion using that sketch"
)]
#[given("a part in the MCAD Software contains a sketch on the XY plane and an extrusion using it")]
#[given("a part is open in the MCAD Software with the model browser visible")]
#[given("a part in the MCAD Software contains a sketch on the XY plane used by an extrusion")]
#[given("a rectangular block in the MCAD Software is made by an extrusion of a dimensioned sketch")]
fn block_part(world: &mut ApplicationWorld) {
    open_block(world);
}
#[given("the Origin branch is expanded")]
#[given("the Origin folder is expanded with its planes, axes and center point visible as rows")]
#[then("the branch shows three origin planes, three origin axes and a center point")]
#[then("the origin planes, axes and center point are visible again")]
#[then("the reference rows are visible again")]
#[then("the Origin folder remains expanded with its direct reference rows visible")]
fn origin_rows_shown(world: &mut ApplicationWorld) {
    expect_origin_rows(world);
}
#[given("its browser has expanded representation, origin and extrusion branches")]
fn expanded_browser_branches(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &["Representations", "Model State", "Origin", FEATURE, SKETCH],
    );
    expect_origin_rows(world);
}
#[when("I click the Origin branch expansion control")]
#[when("the user clicks the Origin branch expansion control")]
#[when("I click the Origin branch expansion control again")]
#[when("the user clicks the Origin branch expansion control again")]
#[when("I click the Origin expansion control")]
#[when("the user clicks the Origin expansion control")]
#[when("I click the Origin expansion control again")]
#[when("the user clicks the Origin expansion control again")]
fn fold_origin(world: &mut ApplicationWorld) {
    click_branch_arrow(world, "Origin");
}
#[then("the origin children are hidden")]
fn origin_folded(world: &mut ApplicationWorld) {
    expect_no_origin_rows(world);
}
#[then("the Origin branch remains visible")]
fn origin_row_stays(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
}
#[then("the reference rows are hidden and the Origin folder remains visible")]
fn origin_folded_folder_stays(world: &mut ApplicationWorld) {
    expect_no_origin_rows(world);
    world.expect_text("Origin");
}
#[then("the solid remains in the same apparent view")]
#[then("the block remains visible in the same apparent view")]
#[then("the solid remains visible in the same apparent view")]
#[then("the block remains visible in the same view")]
#[then("the part remains visible in the same apparent view")]
#[then("the shaded block returns in the same apparent view")]
fn same_view(world: &mut ApplicationWorld) {
    settle(world);
    expect_block_kept(world);
}
// Search
#[given("an active part in the MCAD Software has an extrusion named Extrusion1")]
#[given(
    "an active part in the MCAD Software contains a visible block made by an extrusion named Extrusion1"
)]
fn named_extrusion(world: &mut ApplicationWorld) {
    open_extruded_part_named(world, "Extrusion1");
    world.observed_view_camera = Some(camera(world));
}
#[given("the model browser shows its normal hierarchy")]
#[then("the normal model hierarchy returns")]
#[then("the normal hierarchy remains visible")]
fn normal_tree(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &["Origin", "Sketches", "Features", ORIGIN_RESULT_ROW],
    );
}
#[when("I open browser search and enter \"Extrusion\"")]
#[when("the user opens browser search and enters \"Extrusion\"")]
fn search_extrusion(world: &mut ApplicationWorld) {
    world.click_text("Search");
    world.pump(Vec::new());
    type_in_search(world, "Extrusion");
}
#[then("the browser shows matching extrusion rows and their ancestor hierarchy")]
fn matches_and_ancestors(world: &mut ApplicationWorld) {
    expect_rows(world, &["Extrusion1", "Solid Bodies", BODY, "Features"]);
    let rows = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Extrusion1")
        .count();
    assert_eq!(
        rows, 2,
        "expected the feature under its solid and under Features"
    );
}
#[then("unrelated origin rows are absent from the filtered tree")]
fn no_origin_results(world: &mut ApplicationWorld) {
    world.expect_no_text(ORIGIN_RESULT_ROW);
    world.expect_no_text("Center Point");
}
#[when("the user clears the search text")]
#[when("I clear the search text")]
#[when("I clear the search query")]
fn clear_search_text(world: &mut ApplicationWorld) {
    use oblikovati_viewport::{
        ViewportEvent, ViewportInput, ViewportModifiers, ViewportPhysicalKey,
    };
    let modifiers = ViewportModifiers {
        control: !cfg!(target_os = "macos"),
        super_key: cfg!(target_os = "macos"),
        ..Default::default()
    };
    let key = |pressed, modifiers| {
        ViewportEvent::Input(ViewportInput::PhysicalKey {
            key: ViewportPhysicalKey::A,
            layout_key: ViewportPhysicalKey::A,
            pressed,
            command: None,
            modifiers,
        })
    };
    world.pump(vec![key(true, modifiers), key(false, Default::default())]);
    world.key(ViewportPhysicalKey::Backspace, false);
    settle(world);
}
#[when("I close browser search")]
#[when("the user closes browser search")]
fn press_search_close(world: &mut ApplicationWorld) {
    let close = world.run("x").center().map(|value| value as i32);
    world.click(close);
    settle(world);
}
#[then("the search field disappears")]
fn search_field_gone(world: &mut ApplicationWorld) {
    let field = world.frame.runs.iter().any(|run| run.text == "x");
    assert!(!field, "expected no search field: {}", world.text());
}
// Extended names
/// Escape alone cannot do it: the pointer rests on a row that reopens its submenu at once.
fn close_menus(world: &mut ApplicationWorld) {
    click_empty_viewport(world);
    world.expect_no_text("Display Preferences");
}
#[given("Show Extended Names is disabled in the model browser display preferences")]
fn names_short(world: &mut ApplicationWorld) {
    expand_display_preferences(world);
    assert!(!world.checked("Show Extended Names"), "expected it clear");
    close_menus(world);
}
#[when("I enable Show Extended Names from the browser display preferences")]
#[when("the user enables Show Extended Names from the browser display preferences")]
fn enable_names(world: &mut ApplicationWorld) {
    expand_display_preferences(world);
    choose_menu_row(world, "Show Extended Names");
    settle(world);
}
#[when("I reopen the browser display preferences")]
#[when("the user reopens the browser display preferences")]
fn reopen_preferences(world: &mut ApplicationWorld) {
    expand_display_preferences(world);
}
#[when("I choose Show Extended Names again")]
#[when("the user chooses Show Extended Names again")]
fn names_again(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Show Extended Names");
    settle(world);
}
#[then("the extrusion row includes its new-solid operation and 20 mm extent after its name")]
fn names_long(world: &mut ApplicationWorld) {
    world.expect_text("Extrude1(NewSolid,20mm)");
}
#[then("the extrusion row returns to its shorter name")]
fn names_shorter(world: &mut ApplicationWorld) {
    world.expect_no_text("NewSolid");
    world.expect_text(FEATURE);
}
// The whole tree
#[when("I open the model browser menu and choose Collapse All")]
#[when("the user opens the model browser menu and chooses Collapse All")]
fn collapse_all(world: &mut ApplicationWorld) {
    open_browser_menu(world);
    choose_menu_row(world, "Collapse All");
    settle(world);
}
#[when("I open the model browser menu and choose Expand All")]
#[when("the user opens the model browser menu and chooses Expand All")]
fn expand_all(world: &mut ApplicationWorld) {
    open_browser_menu(world);
    choose_menu_row(world, "Expand All");
    settle(world);
}
#[then("the top-level branches remain visible with their children hidden")]
fn branches_folded(world: &mut ApplicationWorld) {
    expect_rows(world, &["Origin", "Sketches", "Features"]);
    expect_no_origin_rows(world);
    world.expect_no_text(SKETCH);
}
#[then("the origin planes and the nested extrusion sketch are visible")]
fn branches_open(world: &mut ApplicationWorld) {
    expect_origin_rows(world);
    world.expect_text(SKETCH);
}
// Display preferences
#[given("the Hide Work Features display preference is disabled")]
fn work_features_shown(world: &mut ApplicationWorld) {
    expand_display_preferences(world);
    assert!(!world.checked("Hide Work Features"), "expected it clear");
    close_menus(world);
}
#[when("the user enables Hide Work Features in the browser display preferences")]
#[when("the user disables Hide Work Features")]
fn toggle_work_features(world: &mut ApplicationWorld) {
    if !world.shows("Hide Work Features") {
        expand_display_preferences(world);
    }
    choose_menu_row(world, "Hide Work Features");
    settle(world);
}
#[then("the Origin branch is absent from the model browser")]
fn origin_absent(world: &mut ApplicationWorld) {
    world.expect_no_text("Origin");
}
#[then("the solid body and extrusion entries remain visible")]
fn bodies_remain(world: &mut ApplicationWorld) {
    if !world.shows(BODY) {
        click_branch_arrow(world, "Solid Bodies");
        settle(world);
    }
    expect_rows(world, &[BODY, FEATURE]);
}
#[when("the user opens the browser display preferences")]
fn open_preferences(world: &mut ApplicationWorld) {
    expand_display_preferences(world);
}
#[then("the Origin branch returns to the model browser collapsed")]
fn origin_back_folded(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    expect_no_origin_rows(world);
}
#[given("Design View Expansion is unchecked in the browser Display Preferences menu")]
fn expansion_off(world: &mut ApplicationWorld) {
    expand_display_preferences(world);
    assert!(!world.checked("Design View Expansion"), "expected it clear");
}
#[when("I choose Design View Expansion")]
#[when("the user chooses Design View Expansion")]
#[when("I choose Design View Expansion again")]
#[when("the user chooses Design View Expansion again")]
fn choose_expansion(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Design View Expansion");
    settle(world);
}
#[when("I reopen the Display Preferences menu")]
#[when("the user reopens the Display Preferences menu")]
fn reopen_display_menu(world: &mut ApplicationWorld) {
    expand_display_preferences(world);
}
#[then("Design View Expansion is checked")]
fn expansion_on(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Design View Expansion"),
        "expected it checked"
    );
}
#[then("Design View Expansion is unchecked")]
fn expansion_clear(world: &mut ApplicationWorld) {
    assert!(!world.checked("Design View Expansion"), "expected it clear");
}
