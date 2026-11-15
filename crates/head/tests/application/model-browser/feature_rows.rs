// SPDX-License-Identifier: GPL-2.0-only
//! Choosing feature, body and root rows, and what they show on the block.
use super::*;

const TOOLBAR: &str = "##selection-toolbar";
/// Well inside the block's outline, in the home view.
const SMALL_VIEW_FACTOR: f64 = 4.;
// A shaded face in the accent colour fills many times what an outline does.

fn expect_toolbar(shown: bool) {
    assert_eq!(
        diagnostics::window_geometry(TOOLBAR).is_some(),
        shown,
        "selection toolbar"
    );
}
fn expect_row_and_edges(world: &mut ApplicationWorld, row: &str) {
    assert!(row_accent(world, row), "expected the {row} row's accent");
    expect_outline(world, true);
}
fn expect_nothing_chosen(world: &mut ApplicationWorld) {
    expect_outline(world, false);
    expect_solid_visible(world);
}
// One extrusion
#[given("the model browser shows the extrusion and its nested sketch")]
#[given("the extrusion branch is expanded and shows its direct sketch child")]
#[given("the extrusion row is visible in the model browser")]
fn nested_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &[FEATURE, SKETCH]);
}
#[when("I select the extrusion row in the model browser")]
#[when("the user selects the extrusion row in the model browser")]
fn select_extrusion(world: &mut ApplicationWorld) {
    world.click_text(FEATURE);
    settle(world);
}
#[then("the extrusion row is highlighted")]
fn extrusion_lit(world: &mut ApplicationWorld) {
    assert!(row_accent(world, FEATURE), "expected the row's accent");
}
#[then("the visible block edges are highlighted in blue")]
#[then("the visible edges of the block are highlighted in blue")]
#[then("the visible block edges remain highlighted")]
fn edges_blue(world: &mut ApplicationWorld) {
    expect_outline(world, true);
}
#[then("a contextual toolbar appears over the model")]
fn toolbar_up(_: &mut ApplicationWorld) {
    expect_toolbar(true);
}
#[then("the blue edge highlighting and contextual toolbar disappear")]
fn toolbar_gone(world: &mut ApplicationWorld) {
    expect_outline(world, false);
    expect_toolbar(false);
}
#[then("the block remains visible")]
fn block_stays(world: &mut ApplicationWorld) {
    expect_solid_visible(world);
}
#[when("I click the extrusion branch expansion control")]
#[when("the user clicks the extrusion branch expansion control")]
#[when("I click the extrusion branch expansion control again")]
#[when("the user clicks the extrusion branch expansion control again")]
fn fold_extrusion(world: &mut ApplicationWorld) {
    click_branch_arrow(world, FEATURE);
    settle(world);
}
#[then("the sketch row is hidden and the extrusion row remains visible")]
fn sketch_folded(world: &mut ApplicationWorld) {
    world.expect_text(FEATURE);
    world.expect_no_text(SKETCH);
}
#[then("the sketch row appears beneath the extrusion")]
fn sketch_back(world: &mut ApplicationWorld) {
    let feature = world.run(FEATURE).center()[1];
    let sketch = world.run(SKETCH).center()[1];
    assert!(
        sketch > feature,
        "sketch {sketch} beneath feature {feature}"
    );
}
#[when("I right-click the extrusion row")]
#[when("the user right-clicks the extrusion row")]
#[when("I right-click the extrusion row in the model browser")]
#[when("the user right-clicks the extrusion row in the model browser")]
fn right_click_extrusion(world: &mut ApplicationWorld) {
    right_click_row(world, FEATURE);
}

#[when("chooses Collapse All Children")]
fn collapse_extrusion_children(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Collapse All Children");
    settle(world);
}

#[then("the direct sketch row remains visible under the extrusion")]
fn direct_sketch_remains(world: &mut ApplicationWorld) {
    sketch_back(world);
    expect_rows(world, &[FEATURE, SKETCH]);
}
#[then("the row is highlighted and the block edges are highlighted in blue")]
fn row_and_edges(world: &mut ApplicationWorld) {
    expect_row_and_edges(world, FEATURE);
}
#[then("the context menu offers Edit Sketch and Edit Feature")]
fn edit_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["Edit Sketch", "Edit Feature"]);
}
#[then("the context menu offers Show Dimensions, Measure and Properties")]
fn show_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["Show Dimensions", "Measure", "Properties"]);
}
#[then("the context menu offers Expand All Children and Collapse All Children")]
#[then("the context menu offers Expand All Children and collapses All Children")]
fn expand_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["Expand All Children", "Collapse All Children"]);
}
#[when("I dismiss the menu and click empty viewport space")]
#[when("the user dismisses the menu and clicks empty viewport space")]
fn dismiss_and_click(world: &mut ApplicationWorld) {
    world.escape();
    settle(world);
    click_empty_viewport(world);
}
#[then("the context menu and blue edge highlighting disappear")]
fn menu_and_edges_gone(world: &mut ApplicationWorld) {
    world.expect_no_text("Edit Feature");
    expect_outline(world, false);
}
#[when("I click empty viewport space")]
fn click_empty(world: &mut ApplicationWorld) {
    click_empty_viewport(world);
}
// Find in Window
#[given("the block appears small in the viewport")]
fn block_small(world: &mut ApplicationWorld) {
    let mut view = camera(world);
    view.scale *= SMALL_VIEW_FACTOR;
    crate::HeadWindow::set_view_camera(&mut world.app().window, view).unwrap();
    world.observed_view_camera = Some(view);
    settle(world);
}
#[when("I choose Find in Window")]
#[when("the user chooses Find in Window")]
fn choose_find_in_window(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Find in Window");
    settle(world);
}
#[then("the block appears larger and fits within the viewport")]
fn block_larger(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_view_camera
        .expect("expected the earlier view");
    let now = camera(world);
    assert!(
        now.scale < earlier.scale,
        "view height {} from {}",
        now.scale,
        earlier.scale
    );
    expect_solid_visible(world);
}
#[then("the extrusion row remains selected")]
fn row_stays(world: &mut ApplicationWorld) {
    assert!(row_accent(world, FEATURE), "expected the row's accent");
}
#[then("the apparent orientation is unchanged")]
fn orientation_same(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_view_camera
        .expect("expected the earlier view");
    let now = camera(world);
    let directions = |view: oblikovati_viewport::ViewportCamera| {
        let toward: [f64; 3] = std::array::from_fn(|axis| view.eye[axis] - view.target[axis]);
        let length = toward.iter().map(|part| part * part).sum::<f64>().sqrt();
        toward.map(|part| part / length)
    };
    for (first, second) in directions(earlier).iter().zip(directions(now)) {
        assert!(
            (first - second).abs() < 1e-9,
            "view direction {first} and {second}"
        );
    }
}
// The body
/// The folder starts closed.
fn open_bodies_folder(world: &mut ApplicationWorld) {
    if !world.shows(BODY) {
        click_branch_arrow(world, "Solid Bodies");
        settle(world);
    }
}
#[given("the solid body row is visible in the model browser")]
fn body_row_shown(world: &mut ApplicationWorld) {
    open_bodies_folder(world);
    world.expect_text(BODY);
}
#[then("the body and its feature remain represented in the browser")]
fn body_listed(world: &mut ApplicationWorld) {
    expect_rows(world, &[BODY, FEATURE]);
}
#[when("I right-click the solid body row")]
#[when("the user right-clicks the solid body row")]
#[when("I right-click the hidden body row")]
#[when("the user right-clicks the hidden body row")]
fn right_click_body(world: &mut ApplicationWorld) {
    right_click_row(world, BODY);
}
#[then("the block faces are highlighted in blue")]
fn faces_blue(world: &mut ApplicationWorld) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(
        super::super::solid_visibility::visible_selected_fill(&capture, &windows),
        "expected a blue face outside the GUI windows"
    );
}
#[then("Visibility is checked in the context menu")]
#[then("Visibility is checked")]
fn visibility_on(world: &mut ApplicationWorld) {
    assert!(world.checked("Visibility"), "expected Visibility checked");
}
#[then("Visibility is not checked")]
#[then("Visibility is unchecked")]
fn visibility_off(world: &mut ApplicationWorld) {
    assert!(!world.checked("Visibility"), "expected Visibility clear");
}
#[when("I choose Visibility")]
#[when("the user chooses Visibility")]
#[when("I choose Visibility again")]
#[when("the user chooses Visibility again")]
fn choose_visibility(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Visibility");
    settle(world);
}
#[then("the block disappears from the viewport")]
fn block_gone(world: &mut ApplicationWorld) {
    let [width, height] = super::super::viewport_orientation::solid_extent_or_zero(world);
    assert_eq!((width, height), (0, 0), "expected no shaded block");
}
#[then("the hidden body is indicated by a blue wireframe while selected")]
fn hidden_wireframe(world: &mut ApplicationWorld) {
    expect_outline(world, true);
    let [width, height] = super::super::viewport_orientation::solid_extent_or_zero(world);
    assert_eq!((width, height), (0, 0), "expected only the wireframe");
}
#[then("the shaded block is hidden and its body remains listed in the browser")]
fn folder_hides(world: &mut ApplicationWorld) {
    block_gone(world);
    world.expect_text(BODY);
}
// The Solid Bodies folder
#[given("the Solid Bodies folder shows the body, its extrusion and nested sketch")]
fn folder_rows(world: &mut ApplicationWorld) {
    click_branch_arrow(world, "Solid Bodies");
    settle(world);
    expand_folder(world);
    expect_rows(world, &[BODY, FEATURE, SKETCH]);
}
fn folder_menu(world: &mut ApplicationWorld, row: &str) {
    right_click_row(world, "Solid Bodies");
    choose_menu_row(world, row);
    settle(world);
}
fn expand_folder(world: &mut ApplicationWorld) {
    folder_menu(world, "Expand All Children");
}
#[when("I right-click Solid Bodies and choose Expand All Children")]
#[when("the user right-clicks Solid Bodies and chooses Expand All Children")]
fn expand_bodies(world: &mut ApplicationWorld) {
    expand_folder(world);
}
#[then("the body row remains visible under the folder")]
fn body_stays(world: &mut ApplicationWorld) {
    world.expect_text(BODY);
}
#[then("the extrusion and sketch rows under the body are hidden")]
fn body_folded(world: &mut ApplicationWorld) {
    let feature_rows = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == FEATURE)
        .count();
    assert_eq!(feature_rows, 1, "expected only the Features branch's row");
    assert_eq!(
        world
            .frame
            .runs
            .iter()
            .filter(|run| run.text == SKETCH)
            .count(),
        1,
        "expected only the Features branch's sketch row"
    );
}
#[then("the body's extrusion and nested sketch rows are visible again")]
fn body_open(world: &mut ApplicationWorld) {
    let feature_rows = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == FEATURE)
        .count();
    assert_eq!(feature_rows, 2, "expected the row under the body as well");
    assert_eq!(
        world
            .frame
            .runs
            .iter()
            .filter(|run| run.text == SKETCH)
            .count(),
        2,
        "expected the sketch row under the body as well"
    );
}
#[given("the Solid Bodies folder is visible in the model browser")]
fn folder_visible(world: &mut ApplicationWorld) {
    open_bodies_folder(world);
}
#[when("I right-click the Solid Bodies folder")]
#[when("the user right-clicks the Solid Bodies folder")]
#[when("I right-click the Solid Bodies folder again")]
#[when("the user right-clicks the Solid Bodies folder again")]
fn right_click_folder(world: &mut ApplicationWorld) {
    right_click_row(world, "Solid Bodies");
}
#[then("its context menu offers Visibility, Expand All Children and collapses All Children")]
fn body_folder_menu(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &["Visibility", "Expand All Children", "Collapse All Children"],
    );
}
// The document's own row
#[given("the document root row is visible in the model browser")]
fn root_listed(world: &mut ApplicationWorld) {
    let name = world.state().browser.expect("expected a browser").name;
    world.expect_text(&name);
}
fn root_name(world: &mut ApplicationWorld) -> String {
    world.state().browser.expect("expected a browser").name
}
#[when("I right-click the document root row")]
#[when("the user right-clicks the document root row")]
fn right_click_root(world: &mut ApplicationWorld) {
    let name = root_name(world);
    right_click_row(world, &name);
}
#[then("the root row is highlighted and the visible block edges are blue")]
fn root_lit(world: &mut ApplicationWorld) {
    let name = root_name(world);
    expect_row_and_edges(world, &name);
}
#[then("the context menu offers Open File Location and Copy File Path")]
fn file_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["Open File Location", "Copy File Path"]);
}
#[then("the context menu offers Create Drawing View and Open Drawing")]
fn drawing_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["Create Drawing View", "Open Drawing"]);
}
#[then("the context menu offers Expand All Children, Collapse All Children and Find in Window")]
fn root_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &[
            "Expand All Children",
            "Collapse All Children",
            "Find in Window",
        ],
    );
}
#[then("the context menu closes and the block edges remain highlighted")]
fn menu_closed_edges_stay(world: &mut ApplicationWorld) {
    world.expect_no_text("Find in Window");
    expect_outline(world, true);
}
#[then("the blue edge highlighting disappears and the block remains visible")]
#[then("the selection highlight disappears and the block remains visible")]
fn edges_gone_block_stays(world: &mut ApplicationWorld) {
    expect_nothing_chosen(world);
}
// The nested sketch
#[when("I right-click the nested sketch row")]
#[when("the user right-clicks the nested sketch row")]
fn right_click_sketch(world: &mut ApplicationWorld) {
    right_click_row(world, SKETCH);
}
#[then("its rectangular geometry and dimensions appear highlighted in the viewport")]
fn sketch_shown(world: &mut ApplicationWorld) {
    expect_outline(world, true);
    let state = world.state();
    assert!(state.browser.is_some(), "expected a browser");
    let labels = crate::HeadSketchWork::sketch_dimension_annotations(&world.app().owner)
        .unwrap()
        .labels;
    assert!(!labels.is_empty(), "expected the sketch's dimensions");
    for label in labels {
        world.expect_text(&label.text);
    }
}
#[then("the sketch context menu offers Edit Sketch, Redefine, Share Sketch and Properties")]
fn sketch_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &["Edit Sketch", "Redefine", "Share Sketch", "Properties"],
    );
}
#[then("the menu offers Visibility and Export Sketch As")]
fn sketch_more(world: &mut ApplicationWorld) {
    expect_rows(world, &["Visibility", "Export Sketch As"]);
}
#[when("I dismiss the menu and clear the selection")]
#[when("the user dismisses the menu and clears the selection")]
fn dismiss_and_clear(world: &mut ApplicationWorld) {
    world.escape();
    settle(world);
    click_empty_viewport(world);
}
