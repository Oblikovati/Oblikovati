// SPDX-License-Identifier: GPL-2.0-only
//! Verify compositing and hover chrome in both native 3D workspaces.
use super::super::desktop_input;
use super::hover::settled_pixels;
use super::wireframe::{image_pixel, triad_lower_left};
use super::*;
use oblikovati_viewport::ViewportCapture;

#[path = "hud_labels.rs"]
mod hud_labels;

const CONTROLS: [&str; 8] = [
    "Home",
    "left triangle",
    "right triangle",
    "up triangle",
    "down triangle",
    "upper arc",
    "lower arc",
    "menu chevron",
];

pub(in super::super) struct HudSample {
    document: u64,
    idle: Option<ViewportCapture>,
    visible: Option<ViewportCapture>,
}

#[given("a Part or Assembly 3D document is active in the MCAD Software")]
fn part_and_assembly_ready(world: &mut ApplicationWorld) {
    let part_path =
        super::super::scenario_root().join("fixtures/cad/docking-visible-solid-current.opd");
    fixture::open_saved_navigation_block(world, &part_path);
    let part = world.active_id();
    let assembly = world.add_template(oblikovati_gui::GuiTemplateKind::Assembly);
    assert_eq!(
        world.state().document_type,
        Some(oblikovati_api::types::DOCUMENT_ASSEMBLY)
    );
    world.orientation_hud = [part, assembly]
        .map(|document| HudSample {
            document,
            idle: None,
            visible: None,
        })
        .into();
}

fn activate(world: &mut ApplicationWorld, document: u64) {
    if world.active_id() != document {
        let name = world.document_name(document);
        let tab = world.expect_tab(&name);
        world.click(diagnostics::pointer_pixels(tab.center()));
        world.pump(Vec::new());
    }
    world.expect_active(document);
}

fn look_from(world: &mut ApplicationWorld, direction: [f64; 3]) {
    let mut camera = super::super::viewport_pan::camera(world);
    camera.eye = std::array::from_fn(|axis| camera.target[axis] + direction[axis] * 20.);
    camera.up = [0., 1., 0.];
    crate::HeadWindow::set_view_camera(&mut world.app().window, camera).unwrap();
    world.pump(Vec::new());
}

fn hover_control(world: &mut ApplicationWorld, point: [f32; 2]) -> ViewportCapture {
    desktop_input::hover(world, diagnostics::pointer_pixels(point));
    settled_pixels(world)
}

fn at(origin: [f32; 2], point: [f32; 2]) -> [f32; 2] {
    [origin[0] + point[0], origin[1] + point[1]]
}

#[when("the pointer is away from the orientation control")]
fn leave_cube(world: &mut ApplicationWorld) {
    for index in 0..world.orientation_hud.len() {
        activate(world, world.orientation_hud[index].document);
        look_from(world, [0., 0., 1.]);
        let (origin, _) = layout();
        let idle = hover_control(world, at(origin, [-60., 40.]));
        expect_control_shows(world, "FRONT");
        world.orientation_hud[index].idle = Some(idle);
    }
}

#[then(
    "the ViewCube composites transparently over the 3D canvas without an opaque rectangular cutout"
)]
fn transparent_cube_margin(world: &mut ApplicationWorld) {
    for index in 0..world.orientation_hud.len() {
        let sample = &world.orientation_hud[index];
        activate(world, sample.document);
        let (origin, size) = control_geometry();
        let idle = world.orientation_hud[index].idle.as_ref().unwrap();
        for y in [15., 30., 50., 70., 90.] {
            let canvas = image_pixel(idle, at(origin, [-8., y]));
            for x in [3., 6., size[0] - 6., size[0] - 3.] {
                assert_eq!(
                    image_pixel(idle, at(origin, [x, y])),
                    canvas,
                    "viewport row at [{x}, {y}]"
                );
            }
        }
    }
}

#[then(
    "surrounding controls fade out while the Navigation Bar and lower-left XYZ axis triad remain visible"
)]
fn chrome_fades(world: &mut ApplicationWorld) {
    for index in 0..world.orientation_hud.len() {
        activate(world, world.orientation_hud[index].document);
        assert!(diagnostics::window_geometry(NAVIGATION).is_some());
        triad_lower_left(world);
        let (origin, parts) = layout();
        let idle = world.orientation_hud[index].idle.as_ref().unwrap();
        for (name, center) in CONTROLS.into_iter().zip(control_centers(parts)) {
            let canvas = image_pixel(idle, at(origin, [-8., center[1]]));
            let pixel = image_pixel(idle, at(origin, center));
            assert_eq!(pixel, canvas, "{name} should fade out");
        }
        world.save_capture(&format!("viewcube-{}-idle", workspace(index)));
    }
}

fn control_centers(parts: GuiOrientationLayout) -> [[f32; 2]; 8] {
    [
        parts.home,
        parts.step_left,
        parts.step_right,
        parts.step_up,
        parts.step_down,
        parts.roll_upper,
        parts.roll_lower,
        parts.menu,
    ]
}

#[when("the pointer hovers over the ViewCube")]
fn hover_cube(world: &mut ApplicationWorld) {
    for index in 0..world.orientation_hud.len() {
        activate(world, world.orientation_hud[index].document);
        let (origin, parts) = layout();
        let visible = hover_control(world, at(origin, parts.center));
        world.orientation_hud[index].visible = Some(visible);
        world.save_capture(&format!("viewcube-{}-hover", workspace(index)));
    }
}

#[then(
    "the Home control, cardinal triangles, curved 90-degree roll arcs, and context-menu chevron appear"
)]
fn reveal_controls(world: &mut ApplicationWorld) {
    for index in 0..world.orientation_hud.len() {
        activate(world, world.orientation_hud[index].document);
        let (origin, parts) = layout();
        let sample = &world.orientation_hud[index];
        let idle = sample.idle.as_ref().unwrap();
        let visible = sample.visible.as_ref().unwrap();
        for (name, center) in CONTROLS.into_iter().zip(control_centers(parts)) {
            assert!(
                changed_pixels(idle, visible, at(origin, center)) >= 7,
                "missing {name}"
            );
        }
        for center in [parts.roll_upper, parts.roll_lower] {
            let point = at(origin, center);
            assert_eq!(
                image_pixel(idle, point),
                image_pixel(visible, point),
                "roll arc should have an open center"
            );
        }
    }
}

fn changed_pixels(idle: &ViewportCapture, hover: &ViewportCapture, center: [f32; 2]) -> usize {
    (-5..=5)
        .flat_map(|y| (-5..=5).map(move |x| [x as f32, y as f32]))
        .filter(|offset| {
            let point = at(center, *offset);
            image_pixel(idle, point) != image_pixel(hover, point)
        })
        .count()
}

fn workspace(index: usize) -> &'static str {
    ["part", "assembly"][index]
}

#[then(
    "face labels remain crisp, perspective-aligned to their 3D cube faces, and legible in both Dark and Light themes"
)]
fn labels_follow_faces_in_both_themes(world: &mut ApplicationWorld) {
    use crate::HeadWork;
    for (theme, light) in [("Dark", false), ("Light", true)] {
        world.app().owner.select_theme_from_gui(theme).unwrap();
        world.pump(Vec::new());
        for index in 0..world.orientation_hud.len() {
            activate(world, world.orientation_hud[index].document);
            look_from(world, [1., 0.8, 1.2]);
            let (origin, parts) = layout();
            let capture = hover_control(world, at(origin, parts.home));
            hud_labels::verify(world, &capture, origin, parts, light);
            world.save_capture(&format!(
                "viewcube-{}-{}-labels",
                workspace(index),
                theme.to_lowercase()
            ));
        }
    }
}
