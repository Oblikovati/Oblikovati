// SPDX-License-Identifier: GPL-2.0-only
//! The offset-plane command starts from the native 3D Model ribbon.
use super::ApplicationWorld;

#[test]
fn plane_command_requests_a_base_before_creating_a_work_plane() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Plane");
    world.expect_text("Offset Plane");
    world.expect_text("Select Plane");
    assert!(!world.run_is_enabled("OK"));
    world.click_text("Cancel");
    assert!(world.state().browser.unwrap().features.is_empty());
    world.click_text("Plane");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    world.pump(Vec::new());
    world.expect_no_text("Offset Plane");
    world.close();
}

#[test]
fn offset_plane_from_xz_is_created_and_supports_history() {
    use oblikovati_api::client::Client;
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Plane");
    world.click_text("XZ Plane");
    world.expect_text("1 Plane");
    world.click_text("10 mm");
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("50 mm");
    world.click_text("OK");
    world.pump(Vec::new());
    let planes = Client::new(world.app().owner.document_host())
        .work_planes()
        .list()
        .unwrap()
        .planes
        .unwrap();
    let created = planes
        .iter()
        .find(|plane| plane.name == "Work Plane1")
        .expect("expected Work Plane1");
    let origin = created
        .origin
        .as_ref()
        .expect("expected evaluated plane origin");
    assert!(
        (origin[1].abs() - 5.0).abs() < 1e-9,
        "expected 50 mm offset, got {origin:?}"
    );
    world.key(ViewportPhysicalKey::Z, true);
    world.pump(Vec::new());
    assert!(
        !world
            .state()
            .browser
            .unwrap()
            .features
            .iter()
            .any(|feature| feature.name == "Work Plane1")
    );
    world.key(ViewportPhysicalKey::Y, true);
    world.pump(Vec::new());
    assert!(
        world
            .state()
            .browser
            .unwrap()
            .features
            .iter()
            .any(|feature| feature.name == "Work Plane1")
    );
    let directory = tempfile::tempdir().unwrap();
    let path = directory
        .path()
        .join("offset.opd")
        .to_str()
        .unwrap()
        .to_owned();
    let document = world.active_id();
    let client = Client::new(world.app().owner.document_host());
    client
        .documents()
        .save_as(&oblikovati_api::wire::SaveDocumentAsArgs {
            document,
            new_full_document_name: path.clone(),
        })
        .unwrap();
    client
        .documents()
        .close(&oblikovati_api::wire::CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    client
        .documents()
        .open(&oblikovati_api::wire::OpenDocumentArgs {
            full_document_name: path,
            ..Default::default()
        })
        .unwrap();
    let planes = client.work_planes().list().unwrap().planes.unwrap();
    let plane = planes
        .iter()
        .find(|plane| plane.name == "Work Plane1")
        .expect("expected reopened plane");
    assert!((plane.origin.as_ref().unwrap()[1].abs() - 5.0).abs() < 1e-9);
    world.close();
}

#[test]
fn rejected_zero_offset_keeps_the_draft_and_creates_no_plane() {
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Plane");
    world.click_text("XZ Plane");
    world.click_text("10 mm");
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("0 mm");
    super::x11_input::resize_application_window([640, 480]);
    for _ in 0..3 {
        world.pump(Vec::new());
    }
    assert_eq!(world.extent(), [640, 480]);
    world.save_capture("offset-plane-invalid-at-minimum-size");
    world.expect_text("expected a nonzero length");
    world.expect_text("Offset Plane");
    assert!(!world.run_is_enabled("OK"));
    assert!(world.state().browser.unwrap().features.is_empty());
    assert_offset_plane_fits(&mut world);
    world.click_text("0 mm");
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("-5 mm");
    world.pump(Vec::new());
    assert!(world.run_is_enabled("OK"));
    world.click_text("Cancel");
    world.expect_no_text("Offset Plane");
    assert!(world.state().browser.unwrap().features.is_empty());
    world.close();
}

fn assert_offset_plane_fits(world: &mut ApplicationWorld) {
    let viewport = world.viewport_bounds().map(|edge| edge as f32);
    let panel = super::diagnostics::window_pixel_bounds("Offset Plane").expect("offset properties");
    assert!(
        panel[0] >= viewport[0]
            && panel[1] >= viewport[1]
            && panel[2] <= viewport[2]
            && panel[3] <= viewport[3],
        "properties {panel:?}, viewport {viewport:?}"
    );
    let label = world.run_in_window("Offset Plane", "Offset");
    let value = world.run_in_window("Offset Plane", "0mm");
    assert!(
        label.maximum[0] < value.minimum[0],
        "Offset label must lead the field"
    );
    let cancel = world.run_in_window("Offset Plane", "Cancel");
    assert!(cancel.maximum[1] < panel[3] && panel[3] - cancel.maximum[1] < 40.);
}

#[test]
fn selected_solid_face_can_be_the_offset_plane_base() {
    use oblikovati_api::client::Client;
    let mut world = ApplicationWorld::new();
    super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    world.click_text("Select");
    world.click_text("Faces");
    world.pump(Vec::new());
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let point = super::solid_visibility::visible_fixture_point(
        &capture,
        &super::diagnostics::window_occlusions(),
    )
    .expect("expected a visible planar face");
    world.click(point);
    world.pump(Vec::new());
    assert!(world.state().model_face_chosen);
    world.click_text("Plane");
    world.expect_text("1 Plane");
    world.click_text("OK");
    world.pump(Vec::new());
    let planes = Client::new(world.app().owner.document_host())
        .work_planes()
        .list()
        .unwrap()
        .planes
        .unwrap();
    let plane = planes
        .iter()
        .find(|plane| plane.name == "Work Plane1")
        .expect("expected a face-based offset plane");
    assert!(plane.healthy, "expected a healthy face-based plane");
    world.close();
}

#[test]
fn existing_work_plane_can_be_the_offset_plane_base() {
    use oblikovati_api::client::Client;
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Plane");
    world.click_text("XZ Plane");
    world.click_text("OK");
    world.pump(Vec::new());
    world.click_text("Work Plane1");
    world.click_text("Plane");
    world.expect_text("1 Plane");
    world.click_text("OK");
    world.pump(Vec::new());
    let planes = Client::new(world.app().owner.document_host())
        .work_planes()
        .list()
        .unwrap()
        .planes
        .unwrap();
    let first = planes
        .iter()
        .find(|plane| plane.name == "Work Plane1")
        .unwrap()
        .origin
        .as_ref()
        .unwrap();
    let second = planes
        .iter()
        .find(|plane| plane.name == "Work Plane2")
        .unwrap()
        .origin
        .as_ref()
        .unwrap();
    assert!((first[1] - second[1]).abs() > 0.9);
    world.close();
}
