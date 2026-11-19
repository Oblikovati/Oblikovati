// SPDX-License-Identifier: GPL-2.0-only
//! Verify a feature branch and its sketch remain associated during expansion (#3897).
use super::{ApplicationWorld, browser_feature};
use cucumber::{given, then, when};

#[given("a part document with an expanded Extrusion1 branch is open in the MCAD Software")]
fn expanded_feature_part(world: &mut ApplicationWorld) {
    browser_feature::open_extruded_part_for_face_sketch(world);
    world.expect_text("Extrusion1");
    if !world.shows("Sketch1") {
        browser_feature::click_branch_arrow(world, "Extrusion1");
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
    browser_feature::click_branch_arrow(world, "Extrusion1");
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
