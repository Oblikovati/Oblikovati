// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard navigation of visible feature rows while ribbon tips are present.
use super::super::workspace_layout::browser_feature::{
    FEATURE, SKETCH, expect_outline, open_visible_extruded_part,
};
use super::*;

#[given("a part is active in the MCAD Software with the full 3D Model ribbon visible")]
fn part_with_feature(world: &mut ApplicationWorld) {
    open_visible_extruded_part(world);
    world.expect_text("Extrude");
}
#[given("an extrusion is expanded in the model browser to show its child sketch")]
fn expanded_feature(world: &mut ApplicationWorld) {
    world.expect_text(FEATURE);
    world.expect_text(SKETCH);
}
#[given("the extrusion is selected in the browser")]
fn select_feature(world: &mut ApplicationWorld) {
    world.click_text(FEATURE);
    assert!(world.row_highlighted(FEATURE));
}
#[when("the user presses Down Arrow")]
fn down(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::DownArrow, false);
}
#[then("the child sketch is selected in the browser")]
fn sketch_selected(world: &mut ApplicationWorld) {
    assert!(world.row_highlighted(SKETCH));
    let browser = world.state().browser.unwrap();
    let sketch = browser
        .sketches
        .iter()
        .find(|row| row.name == SKETCH)
        .unwrap();
    assert_eq!(browser.selected_sketches, [sketch.id]);
    assert_eq!(
        super::super::viewport_pan::camera(world),
        world.observed_view_camera.unwrap()
    );
}
#[then("the corresponding sketch geometry is highlighted in the viewport")]
fn sketch_highlighted(world: &mut ApplicationWorld) {
    expect_outline(world, true);
    assert!(
        !crate::HeadWork::sketch_dimension_annotations(&world.app().owner)
            .unwrap()
            .labels
            .is_empty()
    );
}
#[when("the user presses Up Arrow")]
fn up(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::UpArrow, false);
}
#[then("the extrusion is selected again")]
fn feature_selected(world: &mut ApplicationWorld) {
    assert!(world.row_highlighted(FEATURE));
    let browser = world.state().browser.unwrap();
    let feature = browser
        .features
        .iter()
        .find(|row| row.name == FEATURE)
        .unwrap();
    assert_eq!(browser.selected_feature, Some(feature.id));
    assert_eq!(
        super::super::viewport_pan::camera(world),
        world.observed_view_camera.unwrap()
    );
}
#[test]
fn a_collapsed_feature_does_not_navigate_to_its_hidden_sketch() {
    let mut world = ApplicationWorld::new();
    part_with_feature(&mut world);
    super::super::workspace_layout::browser_feature::click_branch_arrow(&mut world, FEATURE);
    select_feature(&mut world);
    let before = super::super::viewport_pan::camera(&mut world);
    down(&mut world);
    let browser = world.state().browser.unwrap();
    assert!(browser.selected_feature.is_some());
    assert!(browser.selected_sketches.is_empty());
    assert_eq!(before, super::super::viewport_pan::camera(&mut world));
    world.close();
}
fn walk_feature_branch(with_tips: bool) {
    let mut world = ApplicationWorld::new();
    part_with_feature(&mut world);
    select_feature(&mut world);
    if with_tips {
        super::alt(&mut world);
    }
    let before = super::super::viewport_pan::camera(&mut world);
    world.key(ViewportPhysicalKey::LeftArrow, false);
    world.expect_no_text(SKETCH);
    assert_eq!(before, super::super::viewport_pan::camera(&mut world));
    world.key(ViewportPhysicalKey::RightArrow, false);
    world.expect_text(SKETCH);
    assert_eq!(
        before,
        super::super::viewport_pan::camera(&mut world),
        "after opening"
    );
    world.key(ViewportPhysicalKey::RightArrow, false);
    assert!(world.row_highlighted(SKETCH));
    assert_eq!(
        before,
        super::super::viewport_pan::camera(&mut world),
        "after entering child"
    );
    world.key(ViewportPhysicalKey::LeftArrow, false);
    assert!(world.row_highlighted(FEATURE));
    world.expect_text(SKETCH);
    let browser = world.state().browser.unwrap();
    assert!(browser.selected_feature.is_some());
    assert!(browser.selected_sketches.is_empty());
    assert_eq!(before, super::super::viewport_pan::camera(&mut world));
    if with_tips {
        assert!(super::exact_tip(&world, "M"));
    }
    world.close();
}

#[test]
fn left_and_right_walk_feature_branch_without_rotating_the_view() {
    walk_feature_branch(false);
}
#[test]
fn left_and_right_walk_feature_branch_with_key_tips_open() {
    walk_feature_branch(true);
}
