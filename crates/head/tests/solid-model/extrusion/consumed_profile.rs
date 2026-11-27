// SPDX-License-Identifier: GPL-2.0-only
use super::super::{diagnostics, workspace_layout::browser_feature::click_branch_arrow};
use super::acceptance::feature_count;
use super::*;

#[given(
    "a fresh copy of `fixtures/cad/extrusion/extrusion-rect-20mm.opd` is open with the source sketch for Extrusion1 made visible"
)]
fn show_consumed_profile(world: &mut ApplicationWorld) {
    let source =
        super::super::scenario_root().join("fixtures/cad/extrusion/extrusion-rect-20mm.opd");
    let copy = world.scratch_folder().join("shared-extrusion-profile.opd");
    std::fs::copy(source, &copy).unwrap();
    world.open_fixture(&copy);
    assert_eq!(feature_count(world), 1);
    let sketch = consumed_sketch(world);
    if !world.frame.runs.iter().any(|run| run.text == sketch.name) {
        click_branch_arrow(world, "Extrusion1");
    }
    if sketch.visible {
        toggle_sketch_visibility(world, &sketch.name);
        assert!(!consumed_sketch(world).visible);
    }
    toggle_sketch_visibility(world, &sketch.name);
    assert!(consumed_sketch(world).visible);
}

fn toggle_sketch_visibility(world: &mut ApplicationWorld, name: &str) {
    world.right_click(diagnostics::pointer_pixels(world.run(name).center()));
    world.click_text("Visibility");
}

#[when("the user starts Extrude and selects its closed region")]
fn select_consumed_profile(world: &mut ApplicationWorld) {
    let sketch = consumed_sketch(world);
    world.click_text(&sketch.name);
    assert_eq!(
        world.state().browser.unwrap().selected_sketches,
        [sketch.id]
    );
    world.click_text("Extrude");
    world.pump(Vec::new());
    let draft = world
        .state()
        .extrude_draft_preview
        .expect("closed profile preview");
    assert_eq!(draft.sketch, sketch.id);
    assert_eq!(draft.profile, 0);
    world.run_in_window("Extrude", "1Profile");
}

#[when("the user confirms a valid second extrusion from the region")]
fn accept_shared_profile(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("OK"));
    world.click_text("OK");
    world.pump(Vec::new());
    assert_eq!(feature_count(world), 2);
}

#[then("the source sketch is shared by both features")]
fn both_features_reference_one_sketch(world: &mut ApplicationWorld) {
    let state = world.state();
    let browser = state.browser.unwrap();
    assert_eq!(browser.bodies.len(), 2);
    let [sketch] = browser.sketches.as_slice() else {
        panic!("expected one shared sketch, got {:?}", browser.sketches);
    };
    assert_eq!(sketch.consumed_by, None);
    world.expect_text(&sketch.name);
    let document = world.active_id();
    let program = &world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_geometry;
    let consumers: Vec<_> = program
        .sketch_consumers(sketch.id)
        .map(|feature| feature.id())
        .collect();
    assert_eq!(
        consumers,
        browser
            .bodies
            .iter()
            .map(|body| body.feature)
            .collect::<Vec<_>>()
    );
}

fn consumed_sketch(world: &mut ApplicationWorld) -> oblikovati_gui::GuiBrowserSketch {
    let browser = world.state().browser.unwrap();
    let first = browser
        .features
        .iter()
        .find(|row| row.name == "Extrusion1")
        .unwrap();
    browser
        .sketches
        .into_iter()
        .find(|row| row.consumed_by == Some(first.id))
        .unwrap()
}
