// SPDX-License-Identifier: GPL-2.0-only
//! Native pointer checks for viewport orientation.
use super::super::{diagnostics, viewport_pan, x11_input};
use super::*;

fn native_press(world: &mut ApplicationWorld, local: [f32; 2], position: [f32; 2]) {
    let point = [position[0] + local[0], position[1] + local[1]];
    x11_input::click(diagnostics::pointer_pixels(point));
    for _ in 0..4 {
        world.pump(Vec::new());
    }
}

#[test]
fn native_rotation_arrow_turns_and_restores_front_view() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    front_displayed(&mut world);
    let part = world.active_id();
    world.observed_view_camera = Some(viewport_pan::camera(&mut world));
    let (position, parts) = layout();
    native_press(&mut world, parts.roll_upper, position);
    label_turned(&mut world);
    taller(&mut world);
    world.save_capture("orientation-native-rotated-3961");
    native_press(&mut world, parts.roll_lower, position);
    expect_control_shows(&world, "Front");
    assert!(label_is_upright(&world));
    assert_eq!(world.active_id(), part);
    world.close();
}

#[test]
fn native_cube_reveals_chrome_marks_the_corner_and_opens_its_menu() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    front_displayed(&mut world);
    let (position, parts) = layout();
    let (bar, bar_size) = diagnostics::window_geometry(NAVIGATION).unwrap();
    assert!(
        (bar[0] + bar_size[0] / 2. - position[0] - parts.center[0]).abs() <= 1.,
        "navigation bar {bar:?} / {bar_size:?}, cube {position:?} / {parts:?}"
    );
    let pixel = |local: [f32; 2]| {
        diagnostics::pointer_pixels([position[0] + local[0], position[1] + local[1]])
    };
    let corner = pixel([
        parts.center[0] + 0.83 * parts.half,
        parts.center[1] - 0.83 * parts.half,
    ]);
    let roof = pixel([parts.home[0], parts.home[1] - 5.]).map(|value| value as usize);
    let away = diagnostics::pointer_pixels([position[0] - 50., position[1] + 45.]);

    hover_cube(&mut world, away);
    let idle_roof = world.pixel_at(roof);
    world.save_capture("viewcube-idle");

    hover_cube(&mut world, pixel(parts.home));
    assert_ne!(world.pixel_at(roof), idle_roof);
    let unmarked = world.pixel_at(corner.map(|value| value as usize));

    hover_cube(&mut world, corner);
    assert_ne!(world.pixel_at(corner.map(|value| value as usize)), unmarked);
    world.save_capture("viewcube-corner-hover");

    native_press(&mut world, parts.menu, position);
    world.expect_text("Perspective with Ortho Faces");
    world.save_capture("viewcube-chevron-menu");
    world.close();
}

fn hover_cube(world: &mut ApplicationWorld, point: [i32; 2]) {
    x11_input::move_pointer(point);
    world.pump(Vec::new());
    std::thread::sleep(std::time::Duration::from_millis(90));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[test]
fn native_blank_assembly_shows_orientation_navigation_and_world_axes() {
    let mut world = ApplicationWorld::new();
    let assembly = world.add_template(oblikovati_gui::GuiTemplateKind::Assembly);
    crate::HeadWindow::set_view_camera(&mut world.app().window, Default::default()).unwrap();
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(NAVIGATION).is_some());
    assert!(diagnostics::window_geometry("##viewport-axis-triad").is_some());
    let (position, parts) = layout();
    world.save_capture("blank-assembly-hud");
    native_press(&mut world, parts.step_right, position);
    navigation::expect_camera_direction(&mut world, [1., 0., 0.]);
    expect_control_shows(&world, "Right");
    assert_eq!(world.active_id(), assembly);
    world.close();
}

#[test]
fn native_origin_picker_keeps_edge_on_planes_reachable_from_the_browser() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    front_displayed(&mut world);
    world.click_text("Start 2D Sketch");
    world.pump(Vec::new());
    for (name, expected) in [("XYPlane", 2), ("YZPlane", 1), ("XZPlane", 1)] {
        let rendered = world
            .frame
            .runs
            .iter()
            .filter(|run| run.text == name)
            .count();
        assert_eq!(rendered, expected, "rendered copies of {name}");
    }
    world.save_capture("front-origin-picker");
    let viewport_label = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "XYPlane")
        .max_by(|a, b| a.minimum[0].total_cmp(&b.minimum[0]))
        .unwrap()
        .center();
    world.click(diagnostics::pointer_pixels(viewport_label));
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    world.finish_sketch();
    front_displayed(&mut world);
    world.click_text("Start 2D Sketch");
    let row = world.run_in_window("Model", "YZPlane").center();
    world.click(diagnostics::pointer_pixels(row));
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    world.close();
}

#[test]
fn native_origin_planes_composite_through_the_viewcube_margin() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    front_displayed(&mut world);
    let mut camera = viewport_pan::camera(&mut world);
    camera.scale *= 0.06;
    crate::HeadWindow::set_view_camera(&mut world.app().window, camera).unwrap();
    world.pump(Vec::new());
    let (position, _) = layout();
    // This point is inside the overlay window and outside the rendered cube and arrows.
    let pixel = diagnostics::pointer_pixels([position[0] + 12., position[1] + 32.])
        .map(|axis| axis as usize);
    let before = world.pixel_at(pixel);
    world.click_text("Start 2D Sketch");
    assert!(
        world.pixel_at(pixel)[0] > before[0],
        "the plane fill should continue through the transparent cube margin"
    );
    world.save_capture("origin-through-viewcube-margin");
    world.close();
}
