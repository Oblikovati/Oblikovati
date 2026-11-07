// SPDX-License-Identifier: GPL-2.0-only
//! `features/application/viewport-orientation`: the cube in the view's corner, and its options.
use super::workspace_layout::browser_feature::open_extruded_part;
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};
use oblikovati_gui::GuiOrientationLayout;
use oblikovati_viewport::ViewportPhysicalKey;

#[path = "appearance.rs"]
mod appearance;
#[path = "behaviour.rs"]
mod behaviour;
#[path = "option_rows.rs"]
mod option_rows;

const CONTROL: &str = "##orientation-control";
const NAVIGATION: &str = "##navigation-bar";
const OPTIONS_TITLE: &str = "Orientation Options";
/// Letters a compass adds around the control; no face's label holds them.
const COMPASS_LETTERS: [&str; 4] = ["N", "E", "S", "W"];
const GREEN_MARGIN: u8 = 40;
const PIXEL_ROUNDING: usize = 2;
const AWAY: [i32; 2] = [420, 300];
/// A press on a face's middle, away from where its rim picks an edge or a corner.
const IN_THE_CORNER: f32 = 0.85;

fn control_geometry() -> ([f32; 2], [f32; 2]) {
    diagnostics::window_geometry(CONTROL).expect("expected the orientation control")
}
fn layout() -> ([f32; 2], GuiOrientationLayout) {
    let (position, size) = control_geometry();
    (position, GuiOrientationLayout::for_side(size[0]))
}
fn press_at(world: &mut ApplicationWorld, position: [f32; 2], local: [f32; 2]) {
    world.click([
        (position[0] + local[0]) as i32,
        (position[1] + local[1]) as i32,
    ]);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn press_part(world: &mut ApplicationWorld, part: fn(&GuiOrientationLayout) -> [f32; 2]) {
    let (position, parts) = layout();
    press_at(world, position, part(&parts));
}
/// Letters of a label drawn on the control, whichever way it is turned.
fn control_runs(world: &ApplicationWorld) -> Vec<diagnostics::RenderedRun> {
    let (position, size) = control_geometry();
    world
        .frame
        .runs
        .iter()
        .filter(|run| !COMPASS_LETTERS.contains(&run.text.as_str()))
        .filter(|run| {
            let [x, y] = run.center();
            x >= position[0]
                && x <= position[0] + size[0]
                && y >= position[1]
                && y <= position[1] + size[1]
        })
        .cloned()
        .collect()
}
fn control_letters(world: &ApplicationWorld) -> String {
    let mut letters: Vec<char> = control_runs(world)
        .iter()
        .flat_map(|run| run.text.chars().collect::<Vec<_>>())
        .collect();
    letters.sort_unstable();
    letters.into_iter().collect()
}
fn sorted(text: &str) -> String {
    let mut letters: Vec<char> = text.chars().collect();
    letters.sort_unstable();
    letters.into_iter().collect()
}
fn expect_control_shows(world: &ApplicationWorld, label: &str) {
    assert_eq!(control_letters(world), sorted(label), "control's label");
}
/// A label reads across or down: its letters spread further one way than the other.
fn label_is_upright(world: &ApplicationWorld) -> bool {
    let runs = control_runs(world);
    let minimum = |axis: usize| {
        runs.iter()
            .map(|run| run.minimum[axis])
            .fold(f32::MAX, f32::min)
    };
    let maximum = |axis: usize| {
        runs.iter()
            .map(|run| run.maximum[axis])
            .fold(f32::MIN, f32::max)
    };
    maximum(0) - minimum(0) > maximum(1) - minimum(1)
}
/// The solid's extent on screen, read from its colour.
fn solid_extent(world: &mut ApplicationWorld) -> [usize; 2] {
    let (low, high) = solid_bounds(world);
    [high[0] - low[0] + 1, high[1] - low[1] + 1]
}
fn solid_bounds(world: &mut ApplicationWorld) -> ([usize; 2], [usize; 2]) {
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let width = capture.extent[0] as usize;
    let (mut low, mut high) = ([usize::MAX; 2], [0; 2]);
    for (index, pixel) in capture.rgba.as_chunks::<4>().0.iter().enumerate() {
        let green = pixel[1] > pixel[0].saturating_add(GREEN_MARGIN)
            && pixel[1] > pixel[2].saturating_add(GREEN_MARGIN);
        if green {
            let at = [index % width, index / width];
            for axis in 0..2 {
                low[axis] = low[axis].min(at[axis]);
                high[axis] = high[axis].max(at[axis]);
            }
        }
    }
    assert!(high[0] >= low[0], "expected the solid in the viewport");
    (low, high)
}
/// Sides in this proportion, give or take the pixels an edge rounds to.
fn expect_proportion(wide: usize, tall: usize, across: usize, down: usize) {
    let difference = (wide * down).abs_diff(tall * across);
    assert!(
        difference <= PIXEL_ROUNDING * (across + down),
        "extent {wide} by {tall}: expected {across} to {down}"
    );
}
fn expect_same_part(world: &mut ApplicationWorld) {
    let active = world.active_id();
    let remembered = *world.parts.last().expect("expected the fixture part");
    assert_eq!(active, remembered, "active part");
    world.expect_text("Origin");
}
fn open_block(world: &mut ApplicationWorld) {
    open_extruded_part(world);
    let id = world.active_id();
    world.parts.push(id);
}
fn press_top_in_iso(world: &mut ApplicationWorld) {
    let (position, parts) = layout();
    press_at(
        world,
        position,
        [
            parts.center[0],
            parts.center[1] - parts.half * IN_THE_CORNER,
        ],
    );
}
/// The middle of the cube picks Front while the view is on Front, as a part opens.
fn press_at_front(world: &mut ApplicationWorld) {
    let (position, parts) = layout();
    press_at(world, position, parts.center);
}
fn open_options(world: &mut ApplicationWorld) {
    world.observed_rows = Some(control_letters(world));
    let (position, parts) = layout();
    world.right_click([
        (position[0] + parts.center[0]) as i32,
        (position[1] + parts.center[1]) as i32,
    ]);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.click_text("Options");
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.expect_text(OPTIONS_TITLE);
}
fn choose_list_row(world: &mut ApplicationWorld, label: &str, row: &str) {
    let field = world.run(label).clone();
    world.click([field.minimum[0] as i32 - 40, field.center()[1] as i32]);
    world.pump(Vec::new());
    world.pump(Vec::new());
    if !row.is_empty() {
        world.click_text(row);
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
}
fn click_row(world: &mut ApplicationWorld, row: &str) {
    world.click_text(row);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn confirm(world: &mut ApplicationWorld) {
    world.click_text("OK");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
// View the top of a part and return to the home view
#[given("a part containing a rectangular block is active in the MCAD Software")]
#[given("a rectangular block is active in the MCAD Software")]
fn given_block(world: &mut ApplicationWorld) {
    open_block(world);
}
#[given("its home view shows the block isometrically")]
fn home_iso(world: &mut ApplicationWorld) {
    press_part(world, |parts| parts.home);
    expect_control_shows(world, "TopFrontRight");
}
#[given("the viewport orientation control is visible")]
#[given("the orientation control is visible in the viewport")]
fn control_visible(_: &mut ApplicationWorld) {
    control_geometry();
}
#[when("I select the Top face of the orientation control")]
fn select_top(world: &mut ApplicationWorld) {
    press_top_in_iso(world);
}
#[then("the viewport shows the top of the block face-on as a rectangle")]
fn top_face_on(world: &mut ApplicationWorld) {
    // 40 mm wide by 20 mm deep from above.
    let [wide, deep] = solid_extent(world);
    expect_proportion(wide, deep, 2, 1);
}
#[then("the orientation control displays Top")]
fn displays_top(world: &mut ApplicationWorld) {
    expect_control_shows(world, "Top");
}
#[then("the same part and model browser remain active")]
fn same_part(world: &mut ApplicationWorld) {
    expect_same_part(world);
}
#[when("I select the orientation control's Home action")]
fn press_home(world: &mut ApplicationWorld) {
    press_part(world, |parts| parts.home);
}
#[then("the viewport returns to the block's isometric home view")]
fn back_home(world: &mut ApplicationWorld) {
    expect_control_shows(world, "TopFrontRight");
    solid_extent(world);
}
// Step between adjacent orthogonal views
#[given("the orientation control displays the Front view")]
#[given("the orientation control displays Front with its label upright")]
#[given("the orientation control displays an upright Front view")]
fn front_displayed(world: &mut ApplicationWorld) {
    press_at_front(world);
    expect_control_shows(world, "Front");
    assert!(label_is_upright(world), "expected an upright label");
}
#[when("I select the right-side navigation arrow beside the orientation control")]
fn step_right(world: &mut ApplicationWorld) {
    press_part(world, |parts| parts.step_right);
}
#[then("the control displays Right")]
fn displays_right(world: &mut ApplicationWorld) {
    expect_control_shows(world, "Right");
}
#[then("the block is displayed face-on from the right")]
fn right_face_on(world: &mut ApplicationWorld) {
    // 20 mm deep by 30 mm tall from the side.
    let [wide, tall] = solid_extent(world);
    expect_proportion(wide, tall, 2, 3);
}
#[when("I select the left-side navigation arrow")]
fn step_left(world: &mut ApplicationWorld) {
    press_part(world, |parts| parts.step_left);
}
#[then("the control displays Front")]
fn displays_front(world: &mut ApplicationWorld) {
    expect_control_shows(world, "Front");
}
#[then("the block returns to the face-on front view")]
fn front_face_on(world: &mut ApplicationWorld) {
    // 40 mm wide by 30 mm tall from the front.
    let [wide, tall] = solid_extent(world);
    expect_proportion(wide, tall, 4, 3);
}
// Rotate and restore a face-on view
#[given("the block's front face is wider than it is tall on screen")]
#[then("the block's front face is wider than it is tall again")]
fn wider(world: &mut ApplicationWorld) {
    let [wide, tall] = solid_extent(world);
    assert!(wide > tall, "extent {wide} by {tall}");
}
#[when("I select the upper curved rotation arrow on the orientation control")]
fn roll_upper(world: &mut ApplicationWorld) {
    press_part(world, |parts| parts.roll_upper);
}
#[then("the control still displays Front with its label rotated by a quarter turn")]
fn label_turned(world: &mut ApplicationWorld) {
    expect_control_shows(world, "Front");
    assert!(!label_is_upright(world), "expected the label turned");
}
#[then("the block's front face is taller than it is wide on screen")]
fn taller(world: &mut ApplicationWorld) {
    let [wide, tall] = solid_extent(world);
    assert!(tall > wide, "extent {wide} by {tall}");
}
#[when("I select the other curved rotation arrow")]
fn roll_lower(world: &mut ApplicationWorld) {
    press_part(world, |parts| parts.roll_lower);
}
#[then("the Front label is upright again")]
fn upright_again(world: &mut ApplicationWorld) {
    expect_control_shows(world, "Front");
    assert!(label_is_upright(world), "expected an upright label");
}
// Choose a corner from a face-on view
#[when("I select the upper-right corner of the orientation control")]
fn select_corner(world: &mut ApplicationWorld) {
    let (position, parts) = layout();
    press_at(
        world,
        position,
        [
            parts.center[0] + parts.half * IN_THE_CORNER,
            parts.center[1] - parts.half * IN_THE_CORNER,
        ],
    );
}
#[then("the block is displayed obliquely with its top, front and right faces visible")]
fn block_oblique(world: &mut ApplicationWorld) {
    solid_extent(world);
    expect_control_shows(world, "TopFrontRight");
}
#[then("the orientation control shows the Top, Front and Right faces")]
fn three_faces(world: &mut ApplicationWorld) {
    expect_control_shows(world, "TopFrontRight");
}
// Options
#[given("its compass preference is off")]
fn compass_off(world: &mut ApplicationWorld) {
    appearance::expect_no_compass(world);
}
#[when("I open the orientation control context menu and choose Options")]
#[when("I open the orientation control options")]
fn context_options(world: &mut ApplicationWorld) {
    open_options(world);
}
#[then("the options dialog displays controls for position, size and inactive opacity")]
fn placement_controls(world: &mut ApplicationWorld) {
    for row in ["On-screen Position", "Size", "Inactive Opacity"] {
        world.expect_text(row);
    }
}
#[then("it displays view transition and orientation settings")]
fn behavior_controls(world: &mut ApplicationWorld) {
    for row in ["Use animated transitions", "Keep model upright"] {
        world.expect_text(row);
    }
}
#[then("the compass option is unchecked and the north angle field is unavailable")]
#[then("the compass option is unchecked")]
fn compass_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Show the compass"), "expected it unchecked");
    expect_north_angle(world, false);
}
fn expect_north_angle(world: &ApplicationWorld, available: bool) {
    assert_eq!(
        world.run_is_enabled("Angle of North"),
        available,
        "Angle of North"
    );
}
#[then("the north angle field is unavailable")]
fn north_off(world: &mut ApplicationWorld) {
    expect_north_angle(world, false);
}
#[when("I enable the compass option")]
fn enable_compass(world: &mut ApplicationWorld) {
    world.click_text("Show the compass");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the north angle field becomes available")]
fn north_on(world: &mut ApplicationWorld) {
    expect_north_angle(world, true);
}
#[when("I cancel the options dialog and reopen it")]
fn cancel_reopen(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    world.pump(Vec::new());
    world.pump(Vec::new());
    open_options(world);
}
// Position
#[given("the orientation control and navigation bar are at the upper right of the viewport")]
fn at_upper_right(world: &mut ApplicationWorld) {
    expect_on_right(world);
}
fn expect_on_right(_: &mut ApplicationWorld) {
    let (control, _) = control_geometry();
    let (bar, _) = diagnostics::window_geometry(NAVIGATION).expect("expected the navigation bar");
    assert!(control[0] > 700. && bar[0] > 700., "{control:?} {bar:?}");
}
#[when("I open the on-screen position list")]
fn open_position_list(world: &mut ApplicationWorld) {
    choose_list_row(world, "On-screen Position", "");
}
#[then("the list offers Top Right, Bottom Right, Top Left and Bottom Left")]
fn position_rows(world: &mut ApplicationWorld) {
    for row in ["Top Right", "Bottom Right", "Top Left", "Bottom Left"] {
        world.expect_text(row);
    }
}
#[when("I choose Top Left and confirm the options")]
fn choose_top_left(world: &mut ApplicationWorld) {
    world.click_text("Top Left");
    world.pump(Vec::new());
    world.pump(Vec::new());
    confirm(world);
}
#[then("the orientation control appears at the upper left of the viewport")]
fn at_upper_left(_: &mut ApplicationWorld) {
    let (control, _) = control_geometry();
    assert!(control[0] < 700. && control[1] < 400., "{control:?}");
}
#[then("the navigation bar appears on the left side of the viewport")]
fn bar_left(_: &mut ApplicationWorld) {
    let (bar, _) = diagnostics::window_geometry(NAVIGATION).expect("expected the navigation bar");
    assert!(bar[0] < 700., "{bar:?}");
}
#[then("the part remains in its existing view")]
fn existing_view(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_rows
        .clone()
        .expect("expected the earlier view");
    assert_eq!(control_letters(world), earlier, "control's label");
}
#[when("I reopen the options, choose Top Right and confirm")]
fn back_top_right(world: &mut ApplicationWorld) {
    open_options(world);
    choose_list_row(world, "On-screen Position", "Top Right");
    confirm(world);
}
#[then("the orientation control and navigation bar return to the right side")]
fn returned_right(world: &mut ApplicationWorld) {
    expect_on_right(world);
}
// Size
#[given("the orientation control is visible with its size set to Automatic")]
fn automatic_size(world: &mut ApplicationWorld) {
    let (_, size) = control_geometry();
    world.observed_counts = vec![size[0] as usize];
}
#[when("I open the orientation control options and expand the size list")]
fn expand_size(world: &mut ApplicationWorld) {
    open_options(world);
    choose_list_row(world, "Size", "");
}
#[then("the list offers Automatic, Tiny, Small, Normal and Large")]
fn size_rows(world: &mut ApplicationWorld) {
    for row in ["Automatic", "Tiny", "Small", "Normal", "Large"] {
        world.expect_text(row);
    }
}
#[when("I choose Large and confirm the options")]
fn choose_large(world: &mut ApplicationWorld) {
    world.click_text("Large");
    world.pump(Vec::new());
    world.pump(Vec::new());
    confirm(world);
}
#[then("the orientation control is visibly larger in the viewport")]
fn is_larger(world: &mut ApplicationWorld) {
    let (_, size) = control_geometry();
    assert!(
        size[0] as usize > world.observed_counts[0],
        "side {}",
        size[0]
    );
}
#[when("I reopen the options, choose Automatic and confirm")]
fn back_automatic(world: &mut ApplicationWorld) {
    open_options(world);
    choose_list_row(world, "Size", "Automatic");
    confirm(world);
}
#[then("the orientation control returns to its earlier size in this viewport")]
fn earlier_size(world: &mut ApplicationWorld) {
    let (_, size) = control_geometry();
    assert_eq!(size[0] as usize, world.observed_counts[0], "side");
}
