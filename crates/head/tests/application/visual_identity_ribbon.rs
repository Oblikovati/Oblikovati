// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn native_normal_modify_rows_fill_the_flyout_and_share_one_pitch() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    ribbon_panel_expansion::opened_unpinned(&mut world);
    let name = "##ribbon-additional-PartModify";
    let (position, size) = diagnostics::window_geometry(name).unwrap();
    let rows = ["MoveBodies", "BendPart", "CopyObject"]
        .map(|label| world.run_in_window(name, label).center());
    assert!(
        rows.windows(2)
            .all(|pair| (pair[1][1] - pair[0][1] - 34.).abs() < 2.)
    );
    world.save_capture("normal-modify-flyout");
    let right_side = diagnostics::pointer_pixels([position[0] + size[0] - 18., rows[0][1]]);
    world.click(right_side);
    assert!(diagnostics::window_geometry("Move Bodies").is_some());
    check_move_bodies_at_minimum_size(&mut world);
    world.close();
}

fn check_move_bodies_at_minimum_size(world: &mut ApplicationWorld) {
    let features = world.state().browser.unwrap().features.len();
    x11_input::resize_application_window([640, 480]);
    for _ in 0..3 {
        world.pump(Vec::new());
    }
    assert_eq!(world.extent(), [640, 480]);
    world.save_capture("move-bodies-at-minimum-size");
    assert_move_body_properties_fit(world);
    let cancel = world.run_in_window("Move Bodies", "Cancel");
    x11_input::click(diagnostics::pointer_pixels(cancel.center()));
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Move Bodies").is_none());
    assert_eq!(world.state().browser.unwrap().features.len(), features);
}

fn assert_move_body_properties_fit(world: &mut ApplicationWorld) {
    let title = "Move Bodies";
    let viewport = world.viewport_bounds().map(|edge| edge as f32);
    let panel = diagnostics::window_pixel_bounds(title).expect("visible Move Bodies properties");
    assert!(panel[0] >= viewport[0] && panel[1] >= viewport[1]);
    assert!(panel[2] <= viewport[2] && panel[3] <= viewport[3]);
    let label = world.run_in_window(title, "Body");
    let body = world.run_in_window(title, "Solid1");
    assert!(
        label.maximum[0] < body.minimum[0],
        "Body label must lead the selector"
    );
    for axis in ["X", "Y", "Z"] {
        let axis_label = world.run_in_window(title, axis);
        assert!((axis_label.minimum[0] - label.minimum[0]).abs() < 2.);
    }
    let cancel = world.run_in_window(title, "Cancel");
    assert!(cancel.maximum[1] < panel[3] && panel[3] - cancel.maximum[1] < 40.);
}

#[test]
fn native_tab_strip_aligns_controls_and_separates_the_brand_mark() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    let (_, band) = diagnostics::window_geometry("##application-ribbon").unwrap();
    assert_eq!(band[1], 32. + 1. + 28. + 88.);
    let icons = ribbon_appearance::icon_quads::icon_quads_inside(([8., 0.], [95., 32.]));
    assert!(icons.len() >= 2);
    assert_eq!(icons[0].1[0] - icons[0].0[0], 16.);
    assert!(icons[1].0[0] - icons[0].1[0] >= 8.);
    let x = (band[0] * 0.9) as usize;
    let border = [x, band[1] as usize - 1];
    let initial = world.pixel_at(border);
    world.click([x as i32, border[1] as i32 - 5]);
    assert_eq!(world.pixel_at(border), initial);
    world.save_capture("ribbon-chrome");
    world.close();
}

#[test]
fn native_normal_command_rows_share_the_same_band_across_tabs() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    assert_row_pitch(&world, "Extrude", "Revolve", 34.);
    world.save_capture("normal-model-command-band");

    world.click_text("Tools");
    world.pump(Vec::new());
    assert_row_pitch(&world, "Application Options", "Document Settings", 34.);
    assert_row_pitch(&world, "Material", "Appearance", 34.);
    world.expect_text("Find");
    world.save_capture("normal-tools-command-band");

    world.click_text("3D Model");
    world.pump(Vec::new());
    world.start_sketch();
    world.pump(Vec::new());
    assert_row_pitch(&world, "Circle", "Center Arc", 22.);
    assert_row_pitch(&world, "Center Arc", "Center Rect.", 22.);
    world.save_capture("normal-sketch-command-band");
    world.close();
}

fn assert_row_pitch(world: &ApplicationWorld, upper: &str, lower: &str, pitch: f32) {
    let (_, band) = diagnostics::window_geometry("##application-ribbon").unwrap();
    assert_eq!(band[1], 32. + 1. + 28. + 88.);
    let upper = world.run_in_window("##application-ribbon", &upper.replace(' ', ""));
    let lower = world.run_in_window("##application-ribbon", &lower.replace(' ', ""));
    let actual = lower.center()[1] - upper.center()[1];
    // The text reader reports ink bounds, which differ with ascenders and descenders.
    assert!(
        (actual - pitch).abs() <= 2.,
        "{upper:?}, {lower:?}: {actual}"
    );
}

#[test]
fn native_light_ribbon_shows_work_and_tools_artwork() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    world.app().owner.select_theme_from_gui("Light").unwrap();
    world.pump(Vec::new());
    world.save_capture("light-model-commands");

    world.click_text("Tools");
    world.pump(Vec::new());
    world.expect_text("Measure");
    world.expect_text("Open Team Web");
    world.save_capture("light-tools-commands");

    world.click_text("View");
    world.pump(Vec::new());
    world.click_text("Edit Theme");
    assert!(diagnostics::window_geometry("Theme editor").is_some());
    world.close();
}

pub(super) fn assert_stacked_command_art(world: &ApplicationWorld, label: &str) {
    let text = world.run(label);
    let center = text.center()[0];
    let top = text.minimum[1];
    let icons = ribbon_appearance::icon_quads::icon_quads_inside((
        [center - 24., top - 56.],
        [center + 24., top],
    ));
    assert!(
        icons.iter().any(|(min, max)| max[0] - min[0] >= 39.),
        "expected command artwork above {label}"
    );
}

#[test]
fn native_document_list_is_centered_and_active_caption_clears_underline() {
    let mut world = ApplicationWorld::new();
    workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    let strip = diagnostics::window_pixel_bounds("##open-document-tabs").unwrap();
    let control = diagnostics::window_pixel_bounds("##document-list-control").unwrap();
    assert_eq!([control[1], control[3]], [strip[1], strip[3]]);

    let button = world
        .run_in_window("##document-list-control", "Documents")
        .clone();
    assert!((button.center()[1] - (strip[1] + strip[3]) * 0.5).abs() <= 2.);

    let caption = world.expect_tab("extruded");
    let [x, ink_bottom] = diagnostics::pointer_pixels(caption.maximum).map(|point| point as usize);
    let underline = (ink_bottom..strip[3] as usize)
        .find(|&y| world.pixel_at([x, y])[..3] == [129, 114, 250])
        .expect("expected the active document underline");
    assert!(
        underline >= ink_bottom + 5,
        "ink ends at {ink_bottom}, underline starts at {underline}"
    );
    world.save_capture("document-list-and-caption");

    world.click(diagnostics::pointer_pixels(button.center()));
    world.expect_text("Close Tabs");
    world.close();
}
