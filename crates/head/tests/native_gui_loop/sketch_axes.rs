// SPDX-License-Identifier: GPL-2.0-only
//! The preference changes sketch axes without changing the global view triad.
use super::*;
use oblikovati_viewport::ViewportCapture;

#[test]
fn sketch_axes_follow_the_applied_preference_and_preserve_the_triad() {
    let preferences = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    native.ribbon_settings_directory = Some(preferences.path().to_owned());
    let mut owner = native_owner_at(native.wake.clone(), preferences.path()).unwrap();
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    create::metric_part(&mut window, &mut owner);
    click_labels(&mut window, &mut owner, &["Start 2D Sketch", "XY Plane"]);
    let plane = owner.active_sketch_edit().unwrap().plane;
    let x = window.native.view.sketch_pixel(plane, [2., 0.]).unwrap();
    let y = window.native.view.sketch_pixel(plane, [0., 2.]).unwrap();
    let before = capture_framebuffer(&mut window.native);
    assert!(axis_near(&before, x, 0));
    assert!(axis_near(&before, y, 1));
    change_axes(&mut window, &mut owner);
    let after = capture_framebuffer(&mut window.native);
    assert!(!axis_near(&after, x, 0));
    assert!(!axis_near(&after, y, 1));
    assert!(application_bdd::window_geometry("##viewport-axis-triad").is_some());
    change_axes(&mut window, &mut owner);
    let restored = capture_framebuffer(&mut window.native);
    assert!(axis_near(&restored, x, 0));
    assert!(axis_near(&restored, y, 1));
    click_labels(&mut window, &mut owner, &["Sketch", "Finish Sketch"]);
    let finished = capture_framebuffer(&mut window.native);
    assert!(!axis_near(&finished, x, 0));
    assert!(!axis_near(&finished, y, 1));
    owner.finish().unwrap();
    window.finish().unwrap();
}

fn change_axes(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    click_labels(
        window,
        owner,
        &["Tools", "Application Options", "Axes", "Apply Sketch"],
    );
    window.scripted = [true, false]
        .map(|pressed| {
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key: oblikovati_viewport::ViewportPhysicalKey::Escape,
                layout_key: oblikovati_viewport::ViewportPhysicalKey::Escape,
                command: Some(oblikovati_viewport::ViewportKey::Cancel),
                modifiers: Default::default(),
                pressed,
            })
        })
        .into();
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
}

fn click_labels(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner, labels: &[&str]) {
    for label in labels {
        click_shell_position(window, owner, application_bdd::rendered_label_center(label));
    }
}

fn axis_near(capture: &ViewportCapture, point: [i32; 2], channel: usize) -> bool {
    (-2..=2).any(|dy| {
        (-2..=2).any(|dx| {
            let x = usize::try_from(point[0] + dx).unwrap();
            let y = usize::try_from(point[1] + dy).unwrap();
            let at = (y * capture.extent[0] as usize + x) * 4;
            let color = &capture.rgba[at..at + 3];
            color[channel] > 110 && color[channel] > color[1 - channel].saturating_add(35)
        })
    })
}
