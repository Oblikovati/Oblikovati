// SPDX-License-Identifier: GPL-2.0-only
//! Verify that the fixture's analytic corners are not hidden by docked controls.
use super::ApplicationWorld;

#[test]
fn native_camera_projection_follows_the_resized_document_area() {
    let mut world = ApplicationWorld::new();
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    let original = world.app().window.native.view.camera_state().unwrap();
    let camera = record_panned_and_zoomed_camera(&mut world, original);
    let wide = world.viewport_bounds();
    super::super::workspace_layout::dockspace_native::resize_boundary(&mut world, wide, -80);
    assert_camera_target_centered(&mut world, camera.target);
    resize_narrow(&mut world);
    assert_camera_target_centered(&mut world, camera.target);
    let view = &mut world.app().window.native.view;
    assert!(!view.set_document_projection([1440, 900], wide).unwrap());
    assert_eq!(view.camera_state().unwrap(), camera);
    assert!(view.previous_view().unwrap());
    assert_eq!(view.camera_state().unwrap(), original);
    assert_camera_target_centered(&mut world, original.target);
    world.pump(Vec::new());
    world.save_capture("camera-projection-after-dock-resize");
    world.close();
}

fn record_panned_and_zoomed_camera(
    world: &mut ApplicationWorld,
    mut camera: oblikovati_viewport::ViewportCamera,
) -> oblikovati_viewport::ViewportCamera {
    camera.eye[0] += 0.5;
    camera.target[0] += 0.5;
    camera.scale *= 1.1;
    let view = &mut world.app().window.native.view;
    assert!(view.set_camera_recorded(camera).unwrap());
    view.camera_state().unwrap()
}

fn resize_narrow(world: &mut ApplicationWorld) {
    super::super::x11_input::resize_application_window([940, 680]);
    for _ in 0..3 {
        world.pump(Vec::new());
    }
    assert_eq!(world.extent(), [940, 680]);
}

fn assert_camera_target_centered(world: &mut ApplicationWorld, target: [f64; 3]) {
    let [left, top, right, bottom] = world.viewport_bounds();
    // ConvertWorldToPixels inverts Y in the inclusive 0..height-1 pixel range.
    let expected = [(left + right) as i32 / 2, (top + bottom) as i32 / 2 - 1];
    let actual = world.app().window.native.view.model_pixel(target).unwrap();
    assert!(
        actual
            .iter()
            .zip(expected)
            .all(|(value, center)| value.abs_diff(center) <= 1),
        "camera target projects to {actual:?}, expected the document area center {expected:?}"
    );
}

pub(super) fn assert_fixture_in_document_area(world: &mut ApplicationWorld) {
    let extent = world.extent();
    let native = &mut world.app().window.native;
    let region = native.gui.as_ref().unwrap().document_area(extent).unwrap();
    // selected_sketch_prism.rs defines this rectangle and browser_feature.rs extrudes it by 2 cm.
    for z in [0., 2.] {
        for y in [0., 3.] {
            for x in [0., 4.] {
                let point = [x, y, z];
                let [px, py] = native.view.model_pixel(point).unwrap();
                assert!(
                    px >= region[0] as i32
                        && px < region[2] as i32
                        && py >= region[1] as i32
                        && py < region[3] as i32,
                    "fixture corner {point:?} projects to [{px}, {py}], outside document area {region:?}"
                );
            }
        }
    }
}
