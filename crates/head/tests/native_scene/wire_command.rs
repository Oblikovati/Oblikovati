// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_viewport::ViewportCamera;

#[test]
fn native_wire_command_homes_the_camera_and_reports_unavailable_commands() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    open_native_part(&mut owner, &directory.path().join("home.opd"));
    window.set_view_slot(owner.view_slot()).unwrap();
    let initial = ViewportCamera {
        eye: [0., 0., 50.],
        ..Default::default()
    };
    window.set_view_camera(initial).unwrap();
    let caller =
        crate::window_caller::WindowCaller::new(&mut owner, &mut window, Default::default());
    let client = Client::new(caller);
    let run = |id: &str| {
        client
            .commands()
            .execute(&ExecuteCommandArgs { id: id.into() })
    };
    assert!(
        run("Unknown.Command")
            .unwrap_err()
            .to_string()
            .contains("supported command ID")
    );
    assert!(
        run("Sketch.Finish")
            .unwrap_err()
            .to_string()
            .contains("disabled")
    );
    assert!(run("View.Home").unwrap().ok);
    let home = client.view().camera().unwrap();
    assert_ne!(home.eye.x, home.target.x);
    assert_ne!(home.eye.y, home.target.y);
    assert_ne!(home.eye.z, home.target.z);
    drop(client);
    owner.finish().unwrap();
    window.finish().unwrap();
}
