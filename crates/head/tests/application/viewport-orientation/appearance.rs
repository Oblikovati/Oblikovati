// SPDX-License-Identifier: GPL-2.0-only
//! Orientation scenarios on how the control and the view look: opacity, projection, compass.
use super::*;
// Inactive opacity
fn press_face_pixel(world: &mut ApplicationWorld) -> [u8; 4] {
    let (position, parts) = layout();
    world.hover(AWAY);
    world.pump(Vec::new());
    // Inside the face, left of its label.
    let at = [
        position[0] + parts.center[0] - parts.half * 0.8,
        position[1] + parts.center[1] + parts.half * 0.8,
    ];
    world.pixel_at([at[0] as usize, at[1] as usize])
}
#[given("the orientation control has 50 percent inactive opacity")]
fn half_opacity(world: &mut ApplicationWorld) {
    let pixel = press_face_pixel(world);
    world.observed_row_plain = Some(pixel);
}
#[when("I open the orientation control options and expand the inactive opacity list")]
fn expand_opacity(world: &mut ApplicationWorld) {
    open_options(world);
    choose_list_row(world, "Inactive Opacity", "");
}
#[then("the list offers 0, 25, 50, 75 and 100 percent")]
fn opacity_rows(world: &mut ApplicationWorld) {
    for row in ["0%", "25%", "75%", "100%"] {
        world.expect_text(row);
    }
}
#[when("I choose 100 percent and confirm the options")]
fn choose_opaque(world: &mut ApplicationWorld) {
    world.click_text("100%");
    world.pump(Vec::new());
    world.pump(Vec::new());
    confirm(world);
}
#[when("I move the pointer away from the orientation control")]
#[when("I move the pointer away from the control")]
fn pointer_away(world: &mut ApplicationWorld) {
    world.hover(AWAY);
    world.pump(Vec::new());
}
#[then("the inactive control appears more opaque than at 50 percent")]
fn more_opaque(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_row_plain
        .expect("expected the earlier colour");
    let now = press_face_pixel(world);
    assert!(now[0] > earlier[0], "face {now:?} from {earlier:?}");
}
#[when("I reopen the options, choose 50 percent and confirm")]
fn back_half(world: &mut ApplicationWorld) {
    open_options(world);
    choose_list_row(world, "Inactive Opacity", "50%");
    confirm(world);
}
#[then("the control returns to its earlier translucent appearance")]
fn translucent_again(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_row_plain
        .expect("expected the earlier colour");
    assert_eq!(press_face_pixel(world), earlier, "face colour");
}
// Projection
pub(super) fn open_menu(world: &mut ApplicationWorld) {
    let (position, parts) = layout();
    world.right_click([
        (position[0] + parts.center[0]) as i32,
        (position[1] + parts.center[1]) as i32,
    ]);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
pub(super) fn choose_mode(world: &mut ApplicationWorld, mode: &str) {
    world.click_text(mode);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn solid_pixels(world: &mut ApplicationWorld) -> usize {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .filter(|pixel| {
            pixel[1] > pixel[0].saturating_add(GREEN_MARGIN)
                && pixel[1] > pixel[2].saturating_add(GREEN_MARGIN)
        })
        .count()
}
#[given("the block is displayed obliquely in Orthographic projection")]
#[given("it is displayed obliquely in Orthographic projection")]
fn oblique_orthographic(world: &mut ApplicationWorld) {
    press_part(world, |parts| parts.home);
    expect_control_shows(world, "TopFrontRight");
    world.observed_counts = vec![solid_pixels(world)];
}
#[when("I open the orientation control context menu")]
#[when("I reopen the orientation control context menu")]
#[when("I reopen the orientation control menu")]
fn open_control_menu(world: &mut ApplicationWorld) {
    open_menu(world);
}
#[then("Orthographic is marked as the current projection")]
fn marks_orthographic(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Orthographic"),
        "expected Orthographic marked"
    );
}
#[then("Perspective and Perspective with Ortho Faces are offered")]
fn offers_perspective(world: &mut ApplicationWorld) {
    world.expect_text("Perspective");
    world.expect_text("Perspective with Ortho Faces");
}
#[when("I choose Perspective")]
fn choose_perspective(world: &mut ApplicationWorld) {
    choose_mode(world, "Perspective");
}
#[then("the block remains visible in an oblique view with a changed projection")]
fn projection_changed(world: &mut ApplicationWorld) {
    expect_control_shows(world, "TopFrontRight");
    let earlier = world.observed_counts[0];
    let now = solid_pixels(world);
    assert!(
        now > 0 && now != earlier,
        "solid pixels {now} from {earlier}"
    );
}
#[then("Perspective is marked as the current projection")]
fn marks_perspective(world: &mut ApplicationWorld) {
    assert!(world.checked("Perspective"), "expected Perspective marked");
}
#[when("I choose Orthographic")]
fn choose_orthographic(world: &mut ApplicationWorld) {
    choose_mode(world, "Orthographic");
}
#[then("the block returns to its orthographic appearance")]
fn orthographic_again(world: &mut ApplicationWorld) {
    assert_eq!(
        solid_pixels(world),
        world.observed_counts[0],
        "solid pixels"
    );
}
#[when("I choose Perspective with Ortho Faces from the orientation control menu")]
fn choose_hybrid(world: &mut ApplicationWorld) {
    open_menu(world);
    choose_mode(world, "Perspective with Ortho Faces");
}
#[when("I select Top on the orientation control")]
fn select_top_face(world: &mut ApplicationWorld) {
    press_top_in_iso(world);
}
#[then("the block is displayed face-on from the top")]
fn top_rectangle(world: &mut ApplicationWorld) {
    top_face_on(world);
    assert_eq!(control_letters(world), sorted("Top"), "control's label");
}
#[then("Perspective with Ortho Faces is marked as the current mode")]
fn marks_hybrid(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Perspective with Ortho Faces"),
        "expected the hybrid marked"
    );
}
#[then("the same part remains active in the top view")]
fn same_part_top(world: &mut ApplicationWorld) {
    expect_same_part(world);
    expect_control_shows(world, "Top");
}
// Compass
#[given("a part is active in the MCAD Software with its Top view displayed")]
pub(super) fn part_top_view(world: &mut ApplicationWorld) {
    open_block(world);
    press_top_in_iso(world);
    expect_control_shows(world, "Top");
}
#[given("the orientation compass is disabled")]
fn compass_disabled(world: &mut ApplicationWorld) {
    expect_no_compass(world);
}
pub(super) fn expect_no_compass(world: &ApplicationWorld) {
    for letter in COMPASS_LETTERS {
        assert!(
            !world.frame.runs.iter().any(|run| run.text == letter),
            "expected no compass letter {letter}"
        );
    }
}
#[when("I enable the compass and confirm the options")]
#[when("I disable the compass and confirm the options")]
pub(super) fn toggle_compass(world: &mut ApplicationWorld) {
    world.click_text("Show the compass");
    world.pump(Vec::new());
    world.pump(Vec::new());
    confirm(world);
}
#[then(
    "a compass with north, east, south and west indicators appears around the orientation control"
)]
pub(super) fn compass_shown(world: &mut ApplicationWorld) {
    for letter in COMPASS_LETTERS {
        assert!(
            world.frame.runs.iter().any(|run| run.text == letter),
            "expected the compass letter {letter}"
        );
    }
}
#[then("the part remains in the Top view")]
fn stays_top(world: &mut ApplicationWorld) {
    expect_control_shows(world, "Top");
}
#[when("I reopen the orientation control options")]
fn reopen_options(world: &mut ApplicationWorld) {
    open_options(world);
}
#[then("the compass option is checked and its north angle field is available")]
fn compass_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Show the compass"), "expected it checked");
    expect_north_angle(world, true);
}
#[then("the compass disappears")]
fn compass_gone(world: &mut ApplicationWorld) {
    expect_no_compass(world);
}
#[then("the orientation control and the same part remain visible")]
fn control_and_part(world: &mut ApplicationWorld) {
    control_geometry();
    expect_same_part(world);
}
