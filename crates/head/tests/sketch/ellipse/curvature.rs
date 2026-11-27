// SPDX-License-Identifier: GPL-2.0-only
//! Native curvature commands and rendered comb observations.
use super::super::super::{capture_framebuffer, diagnostics};
use super::{
    ApplicationWorld, dimension_lifecycle::assert_rendered_ellipse,
    dimension_major::ellipse_axis_end_pixel,
};
use cucumber::{given, then, when};
use oblikovati_viewport::{ViewportCapture, ViewportSketchCurvature};

#[given("the rotated ellipse has no curvature comb visible")]
fn no_initial_comb(world: &mut ApplicationWorld) {
    assert!(current_comb(world).is_none());
    assert_rendered_ellipse(world);
    world.last_capture = Some(capture_framebuffer(&mut world.app().window.native));
}

#[given("the user enables Display Curvature on the ellipse")]
#[when("the user right-click its curve and choose Display Curvature")]
pub(super) fn enable_curvature(world: &mut ApplicationWorld) {
    open_curve_menu(world);
    assert!(!world.checked("Display Curvature"));
    native_click_text(world, "Display Curvature");
}

#[then("a green comb appears around the ellipse")]
fn green_comb_is_drawn(world: &mut ApplicationWorld) {
    assert!(current_comb(world).is_some());
    let current = capture_framebuffer(&mut world.app().window.native);
    let baseline = world.last_capture.as_ref().unwrap();
    let added = added_green_pixels(baseline, &current);
    assert!(
        added > 50,
        "green comb pixels {added}: expected more than 50"
    );
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("ellipse-curvature-visible");
    }
}

fn added_green_pixels(before: &ViewportCapture, after: &ViewportCapture) -> usize {
    assert_eq!(before.extent, after.extent);
    before
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .zip(after.rgba.as_chunks::<4>().0)
        .filter(|(previous, current)| {
            current != previous && current[1] > current[0].max(current[2]).saturating_add(25)
        })
        .count()
}

#[when("the user right-click its curve again")]
pub(super) fn open_curve_menu(world: &mut ApplicationWorld) {
    let position = ellipse_axis_end_pixel(world);
    #[cfg(target_os = "linux")]
    {
        super::super::super::x11_input::right_click(position);
        world.pump(Vec::new());
    }
    #[cfg(not(target_os = "linux"))]
    world.right_click(position);
    world.run("Display Curvature");
}

#[then("Display Curvature is checked and Setup Curvature Display is offered")]
fn checked_with_settings(world: &mut ApplicationWorld) {
    assert!(world.checked("Display Curvature"));
    world.run("Setup Curvature Display");
}

#[when("the user click the checked Display Curvature option")]
fn toggle_curvature_off(world: &mut ApplicationWorld) {
    assert!(world.checked("Display Curvature"));
    native_click_text(world, "Display Curvature");
}

#[when("the user right-click the ellipse and select the checked Display Curvature option")]
fn open_menu_and_toggle_off(world: &mut ApplicationWorld) {
    open_curve_menu(world);
    toggle_curvature_off(world);
}

#[then("the comb disappears and the ellipse remains visible")]
fn only_ellipse_remains(world: &mut ApplicationWorld) {
    assert!(current_comb(world).is_none());
    assert_rendered_ellipse(world);
    if world.last_capture.is_some() {
        let current = capture_framebuffer(&mut world.app().window.native);
        let pixels = added_green_pixels(world.last_capture.as_ref().unwrap(), &current);
        assert!(
            pixels <= 50,
            "green comb pixels {pixels}: expected the comb to be hidden"
        );
    }
}

pub(super) fn current_comb(world: &mut ApplicationWorld) -> Option<ViewportSketchCurvature> {
    let scene = world.app().window.native.sketch_scene_status().unwrap();
    let mut combs = scene.sketches.iter().flat_map(|sketch| &sketch.curvature);
    let first = combs.next().copied();
    assert!(
        combs.next().is_none(),
        "expected at most one active curvature comb"
    );
    first
}

pub(super) fn native_click_text(world: &mut ApplicationWorld, label: &str) {
    let position = diagnostics::pointer_pixels(world.run(label).center());
    native_click(world, position);
}

pub(super) fn native_click(world: &mut ApplicationWorld, position: [i32; 2]) {
    #[cfg(target_os = "linux")]
    {
        super::super::super::x11_input::click(position);
        world.pump(Vec::new());
    }
    #[cfg(not(target_os = "linux"))]
    world.click(position);
}
