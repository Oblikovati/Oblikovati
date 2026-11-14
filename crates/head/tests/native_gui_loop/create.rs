// SPDX-License-Identifier: GPL-2.0-only
//! Create the journey's part with the same chooser a user sees.
use super::*;
use oblikovati_viewport::{ViewportModifiers, ViewportPhysicalKey};

pub(super) fn metric_part(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    let key = ViewportPhysicalKey::N;
    let modifiers = ViewportModifiers {
        control: true,
        ..Default::default()
    };
    window.scripted = [true, false]
        .map(|pressed| {
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key,
                layout_key: key,
                command: None,
                pressed,
                modifiers: if pressed {
                    modifiers
                } else {
                    Default::default()
                },
            })
        })
        .into();
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    for label in ["Metric", "Standard metric part", "Create"] {
        let position = application_bdd::rendered_label_center(label);
        click_shell_position(window, owner, position);
    }
}

#[test]
fn parameter_editor_accepts_scaled_pointer_input() {
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    metric_part(&mut window, &mut owner);
    parameters::create_size_parameter(&mut window, &mut owner);
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn sketch_plane_label_remains_clickable_at_workspace_edge() {
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    metric_part(&mut window, &mut owner);
    model_state_journey::prepare(&mut owner);
    parameters::create_size_parameter(&mut window, &mut owner);
    for label in ["Start 2D Sketch", "XY Plane"] {
        let position = application_bdd::rendered_label_center(label);
        click_shell_position(&mut window, &mut owner, position);
    }
    let edit = owner.active_sketch_edit().expect("expected an XY sketch");
    assert_eq!(edit.plane, oblikovati_model::sketch::SketchPlane::xy());
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn sketch_viewcube_input_does_not_create_line_geometry() {
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    metric_part(&mut window, &mut owner);
    for label in ["Start 2D Sketch", "XY Plane"] {
        click_shell_position(
            &mut window,
            &mut owner,
            application_bdd::rendered_label_center(label),
        );
    }
    owner
        .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Line)
        .unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(2) }, &mut owner).unwrap();
    let edit = owner.active_sketch_edit().unwrap();
    let (position, size) = application_bdd::window_geometry("##orientation-control").unwrap();
    let home = oblikovati_gui::GuiOrientationLayout::for_side(size[0]).home;
    let location = application_bdd::pointer_pixels([position[0] + home[0], position[1] + home[1]]);
    click_shell_position(&mut window, &mut owner, location);
    let geometry = owner
        .documents()
        .workspace()
        .session(edit.document)
        .unwrap()
        .snapshot()
        .sketches
        .get(edit.sketch)
        .unwrap()
        .geometry();
    assert!(
        geometry.lines().is_empty(),
        "ViewCube input must not create a line"
    );
    assert!(
        window
            .gui_actions
            .iter()
            .any(|action| matches!(action, oblikovati_gui::GuiShellAction::Home))
    );
    owner.finish().unwrap();
    window.finish().unwrap();
}
