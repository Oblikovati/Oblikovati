// SPDX-License-Identifier: GPL-2.0-only
//! Native pointer checks for viewport orientation.
use super::super::{diagnostics, viewport_pan, x11_input};
use super::*;
use crate::HeadWork;

fn native_press(world: &mut ApplicationWorld, local: [f32; 2], position: [f32; 2]) {
    let point = [position[0] + local[0], position[1] + local[1]];
    x11_input::click(diagnostics::pointer_pixels(point));
    for _ in 0..4 {
        world.pump(Vec::new());
    }
}

#[test]
fn native_view_controls_fit_at_the_minimum_window_size() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    front_displayed(&mut world);
    x11_input::resize_application_window([640, 480]);
    for _ in 0..3 {
        world.pump(Vec::new());
    }
    assert_eq!(world.extent(), [640, 480]);
    let (position, parts) = layout();
    world.hover(diagnostics::pointer_pixels([
        position[0] + parts.center[0],
        position[1] + parts.center[1],
    ]));
    world.save_capture("view-controls-at-minimum-size");
    assert_controls_inside_viewport(&mut world);
    native_press(&mut world, parts.menu, position);
    world.expect_text("Orthographic");
    x11_input::press_escape();
    world.pump(Vec::new());
    let orbit = viewport_pan::bar_icon_center(viewport_pan::NavigationControl::Orbit);
    x11_input::click(diagnostics::pointer_pixels(orbit));
    world.pump(Vec::new());
    world.expect_status("Free Rotation");
    world.close();
}

fn assert_controls_inside_viewport(world: &mut ApplicationWorld) {
    let viewport = world.viewport_bounds().map(|edge| edge as f32);
    for name in ["##orientation-control", "##navigation-bar"] {
        let panel = diagnostics::window_pixel_bounds(name).unwrap();
        assert!(
            panel[0] >= viewport[0]
                && panel[1] >= viewport[1]
                && panel[2] <= viewport[2]
                && panel[3] <= viewport[3],
            "{name} {panel:?} outside viewport {viewport:?}"
        );
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
fn native_origin_picker_keeps_a_face_label_visible_after_zoom() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    front_displayed(&mut world);
    zoomed_front_view(&mut world);
    world.click_text("Start 2D Sketch");
    world.pump(Vec::new());
    world.save_capture("zoomed-origin-plane-picker");

    assert_plane_label_contrast(&mut world);
    world.app().owner.select_theme_from_gui("Light").unwrap();
    world.pump(Vec::new());
    world.save_capture("zoomed-origin-plane-picker-light");
    let label = assert_plane_label_contrast(&mut world);
    assert!(diagnostics::plane_choice_center(&world.frame, "YZ Plane").is_none());
    assert!(diagnostics::plane_choice_center(&world.frame, "XZ Plane").is_none());
    x11_input::click(diagnostics::pointer_pixels(label));
    world.pump(Vec::new());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    world.close();
}

fn zoomed_front_view(world: &mut ApplicationWorld) {
    let mut camera = viewport_pan::camera(world);
    let [minimum, maximum] = crate::HeadWindow::presentation_bounds(&world.app().window).unwrap();
    for axis in 0..2 {
        let center = (minimum[axis] + maximum[axis]) * 0.5;
        camera.eye[axis] += center - camera.target[axis];
        camera.target[axis] = center;
    }
    camera.scale *= 0.1;
    crate::HeadWindow::set_view_camera(&mut world.app().window, camera).unwrap();
    world.pump(Vec::new());
}

fn assert_plane_label_contrast(world: &mut ApplicationWorld) -> [f32; 2] {
    let center = diagnostics::plane_choice_center(&world.frame, "XY Plane")
        .expect("expected the face-on label inside the cropped plane");
    let label = world
        .frame
        .runs
        .iter()
        .find(|run| run.center() == center)
        .unwrap();
    let ink = relative_luminance(label.color);
    let position = diagnostics::pointer_pixels([label.minimum[0] - 2., center[1]]);
    let [r, g, b, _] = world.pixel_at(position.map(|axis| axis as usize));
    let backing = relative_luminance([r, g, b]);
    let contrast = (ink.max(backing) + 0.05) / (ink.min(backing) + 0.05);
    assert!(
        contrast >= 4.5,
        "plane label contrast {contrast:.2} on {:?}",
        [r, g, b]
    );
    center
}

fn relative_luminance(rgb: [u8; 3]) -> f32 {
    rgb.into_iter()
        .zip([0.2126, 0.7152, 0.0722])
        .map(|(channel, weight)| {
            let value = f32::from(channel) / 255.;
            let linear = if value <= 0.04045 {
                value / 12.92
            } else {
                ((value + 0.055) / 1.055).powf(2.4)
            };
            linear * weight
        })
        .sum()
}

#[test]
fn native_origin_plane_fill_matches_studio_opacity_on_the_viewport() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    front_displayed(&mut world);
    let mut camera = viewport_pan::camera(&mut world);
    camera.scale *= 0.15;
    crate::HeadWindow::set_view_camera(&mut world.app().window, camera).unwrap();
    let probe = [600, 300];
    for (theme, channel, amber, opacity) in [("Dark", 0, 245., 0.14), ("Light", 2, 6., 0.12)] {
        world.app().owner.select_theme_from_gui(theme).unwrap();
        world.pump(Vec::new());
        let clear = f32::from(world.pixel_at(probe)[channel]);
        world.click_text("Start 2D Sketch");
        world.pump(Vec::new());
        let tinted = f32::from(world.pixel_at(probe)[channel]);
        world.save_capture(&format!("plane-opacity-{theme}"));
        let apparent = (tinted - clear) / (amber - clear);
        assert!(
            (apparent - opacity).abs() < 0.025,
            "{theme} fill opacity {apparent:.3}, expected {opacity:.3} ({clear} to {tinted})"
        );
        world.escape();
        world.pump(Vec::new());
    }
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
