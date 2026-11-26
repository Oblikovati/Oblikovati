// SPDX-License-Identifier: GPL-2.0-only
//! Read the rendered icons, captions and divider in the native User Commands panel.
use super::super::ribbon_appearance::icon_quads::icon_quads_inside;
use super::transaction_steps::{add_command, all_commands};
use super::*;

type Bounds = ([f32; 2], [f32; 2]);

#[when("the user selects \"Text\" and leaves \"Large\" clear for the command")]
fn choose_small_with_text(world: &mut ApplicationWorld) {
    assert!(!row_option_checked(world, "2DSketches", "Large"));
    toggle_row_option(world, "2DSketches", "Text");
}

#[when("the user selects \"Large\" and leaves \"Text\" clear for the command")]
fn choose_large_without_text(world: &mut ApplicationWorld) {
    assert!(!row_option_checked(world, "2DSketches", "Text"));
    choose_large(world);
}

#[when("the user selects \"Large\" for the same command")]
fn choose_large(world: &mut ApplicationWorld) {
    assert!(!row_option_checked(world, "2DSketches", "Large"));
    toggle_row_option(world, "2DSketches", "Large");
}

#[then("the \"User Commands\" panel shows a small command icon with \"2D Sketches\" beside it")]
fn small_icon_beside_caption(world: &mut ApplicationWorld) {
    let [(lo, hi)] = panel_icons::<1>(world, false);
    let caption = sketch_caption(world).expect("expected the small command caption");
    assert!(caption.minimum[0] >= hi[0]);
    assert!((lo[1]..hi[1]).contains(&caption.center()[1]));
}

#[then("the panel shows a larger command icon with \"2D Sketches\" beneath it")]
fn large_icon_above_caption(world: &mut ApplicationWorld) {
    let [(lo, hi)] = panel_icons::<1>(world, true);
    let caption = sketch_caption(world).expect("expected the large command caption");
    assert!(caption.minimum[1] >= hi[1]);
    assert!((lo[0]..hi[0]).contains(&caption.center()[0]));
}

#[then("the \"User Commands\" panel shows a large command icon without a visible command label")]
fn large_icon_without_caption(world: &mut ApplicationWorld) {
    panel_icons::<1>(world, true);
    assert!(sketch_caption(world).is_none());
}

#[then("the User Commands panel appears with a small command icon and no visible label")]
fn small_icon_without_caption(world: &mut ApplicationWorld) {
    panel_icons::<1>(world, false);
    assert!(sketch_caption(world).is_none());
}

#[when("the user adds \"2D Sketches\", a separator, and \"3D Annotations\" in that order")]
fn stage_two_small_commands_and_separator(world: &mut ApplicationWorld) {
    add_command(world, "2DSketches");
    click_in_customize(world, "<Separator>");
    transfer_to_target(world);
    add_command(world, "3DAnnotations");
    assert_eq!(
        transaction_steps::target_rows(world),
        ["2DSketches", "<Separator>", "3DAnnotations"]
    );
}

#[when("both commands have \"Large\" and \"Text\" clear")]
fn verify_default_options(world: &mut ApplicationWorld) {
    both_options_clear(world);
}

#[then("the \"User Commands\" panel displays the two small command icons")]
fn two_small_icons(world: &mut ApplicationWorld) {
    panel_icons::<2>(world, false);
}

#[then("a vertical divider appears between the icons")]
fn divider_between_icons(world: &mut ApplicationWorld) {
    let [(_, first_end), (second_start, _)] = panel_icons::<2>(world, false);
    let [left, bottom] = diagnostics::pointer_pixels(first_end).map(|value| value as usize);
    let [right, top] = diagnostics::pointer_pixels(second_start).map(|value| value as usize);
    assert!(
        top < bottom && left + 6 < right,
        "expected a gap between the icons"
    );
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let pixel = |x: usize, y: usize| {
        let at = (y * capture.extent[0] as usize + x) * 4;
        &capture.rgba[at..at + 4]
    };
    assert!(
        (left + 3..right - 3).any(|x| {
            (top..bottom + 12)
                .all(|y| pixel(x - 2, y) == pixel(x + 2, y) && pixel(x, y) != pixel(x - 2, y))
        }),
        "expected a continuous vertical rule between the two command icons"
    );
}

#[given("ribbon customization is open with \"All Commands\" selected")]
fn customize_tools_with_all_commands(world: &mut ApplicationWorld) {
    open_tools_customization(world);
    all_commands(world);
}

fn panel_icons<const N: usize>(world: &ApplicationWorld, large: bool) -> [Bounds; N] {
    let mut icons = icon_quads_inside(user_commands_bounds(world));
    icons.sort_by(|a, b| a.0[0].total_cmp(&b.0[0]));
    let icons: [Bounds; N] = icons
        .try_into()
        .unwrap_or_else(|icons| panic!("expected {N} User Commands icons, found {icons:?}"));
    let width_range = if large { 36.0..45.0 } else { 16.0..22.0 };
    assert!(
        icons
            .iter()
            .all(|(lo, hi)| width_range.contains(&(hi[0] - lo[0]))),
        "{icons:?}"
    );
    icons
}

fn sketch_caption(world: &ApplicationWorld) -> Option<&diagnostics::RenderedRun> {
    let (min, max) = user_commands_bounds(world);
    world.frame.runs.iter().find(|row| {
        let [x, y] = row.center();
        row.text == "2DSketches" && (min[0]..max[0]).contains(&x) && (min[1]..max[1]).contains(&y)
    })
}

fn user_commands_bounds(world: &ApplicationWorld) -> Bounds {
    const RIBBON: &str = "##application-ribbon";
    let title = world.run("User Commands");
    let top = world.run_in_window(RIBBON, "3DModel").maximum[1];
    let (position, size) = diagnostics::window_geometry(RIBBON).unwrap();
    let [left, bottom] = title.minimum;
    // The centered caption has padding on both sides of its text.
    ([left - 16., top], [position[0] + size[0], bottom])
}
