// SPDX-License-Identifier: GPL-2.0-only
//! Verify a feature branch and its sketch remain associated during expansion (#3897).
use super::{ApplicationWorld, browser_feature};
use cucumber::{given, then, when};

#[given("a part document with an expanded Extrusion1 branch is open in the MCAD Software")]
fn expanded_feature_part(world: &mut ApplicationWorld) {
    browser_feature::open_extruded_part_for_face_sketch(world);
    expand_extrusion_row(world);
}

#[given(
    "`fixtures/cad/workspace-layout-native.opd` is open in the MCAD Software with an expanded Extrusion1 branch"
)]
fn expanded_named_extrusion(world: &mut ApplicationWorld) {
    super::native_part::open_named_extrusion(world);
    expand_extrusion_row(world);
}

fn expand_extrusion_row(world: &mut ApplicationWorld) {
    world.expect_text("Extrusion1");
    if !world.shows("Sketch1") {
        toggle_feature_branch(world);
    }
    world.expect_text("Sketch1");
}

#[given("the model browser shows its Sketch1 child")]
#[then("the Sketch1 child is shown beneath Extrusion1")]
fn sketch_child_visible(world: &mut ApplicationWorld) {
    let feature = world.run("Extrusion1").center();
    let sketch = world.run("Sketch1").center();
    assert!(
        sketch[1] > feature[1],
        "sketch {sketch:?}: expected below {feature:?}"
    );
}

#[when("the user collapses the Extrusion1 branch")]
#[when("the user expands the Extrusion1 branch")]
fn toggle_feature_branch(world: &mut ApplicationWorld) {
    click_feature_chevron(world, "Extrusion1");
}

pub(super) fn click_feature_chevron(world: &mut ApplicationWorld, label: &str) {
    let row = world.run_in_window("Model", label);
    let [left, _] = row.minimum;
    let y = row.center()[1];
    let arrow = world
        .frame
        .marks
        .iter()
        .find(|mark| {
            let [x, at_y] = mark.position;
            mark.list == row.list
                && mark.bright
                && (left - 52. ..left - 12.).contains(&x)
                && (at_y - y).abs() < 6.
        })
        .unwrap_or_else(|| panic!("expected the rendered chevron left of {label:?}"));
    world.click(super::super::diagnostics::pointer_pixels([
        arrow.position[0],
        y,
    ]));
    world.pump(Vec::new());
}

#[then("the Sketch1 child is hidden from the browser")]
fn sketch_child_hidden(world: &mut ApplicationWorld) {
    world.expect_text("Extrusion1");
    world.expect_no_text("Sketch1");
}

#[then("the solid remains displayed in the viewport")]
fn solid_remains(world: &mut ApplicationWorld) {
    browser_feature::expect_solid_visible(world);
}
