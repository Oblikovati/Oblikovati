// SPDX-License-Identifier: GPL-2.0-only
//! Native sketch visibility against a solid and an independently drawn profile.
use super::super::workspace_layout::browser_feature::{
    click_branch_arrow, open_extruded_part_for_second_sketch,
};
use super::super::{ApplicationWorld, diagnostics, solid_visibility};
use cucumber::{given, then, when};

#[test]
fn finishing_an_isolated_sketch_restores_the_model_without_a_visibility_edit() {
    let mut world = ApplicationWorld::new();
    active_sketch_with_solid(&mut world);
    let generation = world.state().document_generation;
    hide_available(&mut world);
    hide_others(&mut world);
    assert_eq!(world.state().document_generation, generation);
    assert_solid(&mut world, false);
    world.click_text("Finish Sketch");
    assert_solid(&mut world, true);
    assert!(!world.state().sketch_isolated);
    world.close();
}

#[given("a sketch is active with surrounding model shading visible")]
fn active_sketch_with_solid(world: &mut ApplicationWorld) {
    open_extruded_part_for_second_sketch(world);
    world.start_sketch();
    world.draw_rectangle();
    world.escape();
    world.pump(Vec::new());
    click_branch_arrow(world, "Solid Bodies");
    let run = world.run("Solid1");
    world.observed_row_plain = Some([run.color[0], run.color[1], run.color[2], run.alpha]);
    world.record_initial_sketch();
    assert_solid(world, true);
}
#[given("Hide Others is available in the empty-canvas context menu")]
fn hide_available(world: &mut ApplicationWorld) {
    open_menu(world);
    assert!(world.run_is_enabled("Hide Others"));
    assert!(!world.run_is_enabled("Show All"));
}
#[when("the user chooses Hide Others")]
fn hide_others(world: &mut ApplicationWorld) {
    world.click_text("Hide Others");
    world.pump(Vec::new());
}
#[then("the surrounding model shading disappears")]
fn shading_hidden(world: &mut ApplicationWorld) {
    assert_solid(world, false);
}
#[then("the active sketch outlines remain visible")]
fn sketch_remains(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    let original = &world.observed_sketch.as_ref().unwrap().geometry;
    assert_eq!(geometry.lines(), original.lines());
    assert!(
        visible_rectangle_edge(world),
        "expected the drawn sketch edge on the canvas"
    );
}
#[then("the surrounding model browser entries appear dim")]
#[given("the surrounding model browser entries appear dim")]
fn body_dim(world: &mut ApplicationWorld) {
    let prior = world.observed_row_plain.unwrap();
    let row = world.run("Solid1");
    assert!(
        row.color.iter().map(|value| u32::from(*value)).sum::<u32>() * u32::from(row.alpha)
            < prior[..3]
                .iter()
                .map(|value| u32::from(*value))
                .sum::<u32>()
                * u32::from(prior[3]),
        "Solid1 row {:?} alpha {}: expected dimmer than {prior:?}",
        row.color,
        row.alpha
    );
}
#[when("the user reopens the context menu")]
fn reopen_menu(world: &mut ApplicationWorld) {
    open_menu(world);
}
#[then("Show All is available and Hide Others is disabled")]
fn restore_available(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Show All"));
    assert!(!world.run_is_enabled("Hide Others"));
}
#[given("Hide Others has been applied in the fixture sketch")]
fn isolated_sketch(world: &mut ApplicationWorld) {
    active_sketch_with_solid(world);
    hide_available(world);
    hide_others(world);
    assert_solid(world, false);
}
#[when("the user opens the empty-canvas context menu and choose Show All")]
fn show_all(world: &mut ApplicationWorld) {
    open_menu(world);
    world.click_text("Show All");
    world.pump(Vec::new());
}
#[then("surrounding model shading is visible again")]
fn shading_restored(world: &mut ApplicationWorld) {
    assert_solid(world, true);
}
#[then("the browser entries return to their earlier appearance")]
fn body_restored(world: &mut ApplicationWorld) {
    let row = world.run("Solid1");
    let color = [row.color[0], row.color[1], row.color[2], row.alpha];
    assert_eq!(Some(color), world.observed_row_plain);
}
#[then("Show All is disabled and Hide Others is available")]
fn hide_available_again(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("Show All"));
    assert!(world.run_is_enabled("Hide Others"));
}
#[then("the menu closes and restored visibility remains")]
fn menu_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Hide Others");
    assert_solid(world, true);
}
fn open_menu(world: &mut ApplicationWorld) {
    let [left, _, _, bottom] = world.viewport_bounds();
    world.right_click([left as i32 + 60, bottom as i32 - 60]);
    world.pump(Vec::new());
}
fn assert_solid(world: &mut ApplicationWorld, expected: bool) {
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = diagnostics::window_occlusions();
    assert_eq!(
        solid_visibility::visible_solid_fill(&capture, &windows),
        expected
    );
}
fn visible_rectangle_edge(world: &mut ApplicationWorld) -> bool {
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let width = capture.extent[0] as usize;
    (800..1020)
        .filter(|x| {
            (378..=382).any(|y| {
                let at = (y * width + *x) * 4;
                let background = (370 * width + *x) * 4;
                let pixel = &capture.rgba[at..at + 3];
                let contrast: u32 = pixel
                    .iter()
                    .zip(&capture.rgba[background..background + 3])
                    .map(|(line, background)| u32::from(line.abs_diff(*background)))
                    .sum();
                contrast > 60
            })
        })
        .count()
        > 60
}
