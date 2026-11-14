// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn open_parameter_panel(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    for label in ["Manage", "Parameters"] {
        click_shell_position(window, owner, application_bdd::rendered_label_center(label));
    }
}

#[test]
fn native_parameter_panel_opens_with_owner_metadata_and_displays_local_results() {
    let directory = tempfile::tempdir().unwrap();
    let preferences = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    native.ribbon_settings_directory = Some(preferences.path().to_owned());
    let mut owner = native_owner_at(native.wake.clone(), preferences.path()).unwrap();
    open_native_part(&mut owner, &directory.path().join("native parameters.opd"));
    let mut window = scripted_window(native);
    let closed = publish_gui_framebuffer(&mut owner, &mut window.native);
    assert!(!window.gui_parameters_open());
    open_parameter_panel(&mut window, &mut owner);
    assert!(window.gui_parameters_open());
    let open = capture_framebuffer(&mut window.native);
    assert_ne!(closed.rgba, open.rgba);
    let snapshot = owner.gui_parameter_snapshot().unwrap().unwrap();
    let action = oblikovati_gui::GuiParameterAction::Toggle {
        document: snapshot.document,
        parameter: snapshot.rows[0].parameter,
        field: oblikovati_gui::GuiParameterToggleField::Key,
        value: true,
    };
    window
        .acknowledge_gui_parameter(
            &action,
            Err("Parameter edit failed: expected an editable row".into()),
        )
        .unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    let failed = capture_framebuffer(&mut window.native);
    assert_ne!(open.rgba, failed.rgba);
    window.acknowledge_gui_parameter(&action, Ok(())).unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    let recovered = capture_framebuffer(&mut window.native);
    assert_ne!(failed.rgba, recovered.rgba);
    owner.finish().unwrap();
    window.finish().unwrap();
}
#[test]
fn native_parameter_view_failure_is_visible_and_recovers_without_closing_the_window() {
    use oblikovati_api::{client::Client, wire::ParameterSetArgs};
    let directory = tempfile::tempdir().unwrap();
    let preferences = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    native.ribbon_settings_directory = Some(preferences.path().to_owned());
    let mut owner = native_owner_at(native.wake.clone(), preferences.path()).unwrap();
    open_native_part(
        &mut owner,
        &directory.path().join("parameter view recovery.opd"),
    );
    let mut window = scripted_window(native);
    publish_gui_framebuffer(&mut owner, &mut window.native);
    open_parameter_panel(&mut window, &mut owner);
    let healthy = capture_framebuffer(&mut window.native);
    Client::new(owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "material".into(),
            expression: "bad\0text".into(),
            value_type: "text".into(),
            ..Default::default()
        })
        .unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    let failed = capture_framebuffer(&mut window.native);
    assert_ne!(healthy.rgba, failed.rgba);
    assert!(window.gui_parameters_open());
    let view = owner.gui_parameter_snapshot().unwrap().unwrap();
    let action = oblikovati_gui::GuiParameterAction::Text(oblikovati_gui::GuiParameterTextEdit {
        target: oblikovati_gui::GuiParameterTextTarget {
            document: view.document,
            parameter: view.rows.last().unwrap().parameter,
            field: oblikovati_gui::GuiParameterTextField::Equation,
        },
        request: 1,
        value: "steel".into(),
    });
    owner.apply_gui_parameter(&action).unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    let recovered = capture_framebuffer(&mut window.native);
    assert_ne!(failed.rgba, recovered.rgba);
    assert!(window.gui_parameters_open());
    owner.finish().unwrap();
    window.finish().unwrap();
}
