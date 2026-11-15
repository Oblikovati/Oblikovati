// SPDX-License-Identifier: GPL-2.0-only
//! Native work-plane presentation follows history and the owner scene.
use super::ApplicationWorld;
use crate::HeadWork;
use oblikovati_viewport::{ViewportCamera, ViewportCapture, ViewportPhysicalKey};

fn camera(world: &mut ApplicationWorld) {
    world
        .app()
        .window
        .native
        .view
        .set_camera(ViewportCamera {
            eye: [20.; 3],
            up: [0., 0., 1.],
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
}

fn capture(world: &mut ApplicationWorld) -> ViewportCapture {
    super::capture_framebuffer(&mut world.app().window.native)
}

fn pixel(capture: &ViewportCapture, [x, y]: [i32; 2]) -> [u8; 4] {
    assert!(x >= 0 && y >= 0 && (x as u32) < capture.extent[0] && (y as u32) < capture.extent[1]);
    let index = ((y as usize * capture.extent[0] as usize) + x as usize) * 4;
    capture.rgba[index..index + 4].try_into().unwrap()
}

fn create_plane(world: &mut ApplicationWorld) {
    world.click_text("Plane");
    world.click_text("XY Plane");
    world.click_text("OK");
    world.pump(Vec::new());
}

fn center(world: &mut ApplicationWorld) -> [i32; 2] {
    let planes = world.app().owner.datum_planes().unwrap().planes;
    assert_eq!(planes.len(), 1, "expected one visible evaluated work plane");
    let corners = planes[0].corners;
    let center =
        std::array::from_fn(|axis| corners.iter().map(|corner| corner[axis]).sum::<f64>() / 4.);
    world.app().window.native.view.model_pixel(center).unwrap()
}

#[test]
fn created_plane_is_rendered_and_follows_undo_and_redo() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    camera(&mut world);
    let before = capture(&mut world);
    create_plane(&mut world);
    let point = center(&mut world);
    let shown = pixel(&capture(&mut world), point);
    assert_ne!(
        shown,
        pixel(&before, point),
        "plane fill should change its projected center"
    );
    world.key(ViewportPhysicalKey::Z, true);
    world.pump(Vec::new());
    assert!(world.app().owner.datum_planes().unwrap().planes.is_empty());
    assert_eq!(pixel(&capture(&mut world), point), pixel(&before, point));
    world.key(ViewportPhysicalKey::Y, true);
    world.pump(Vec::new());
    assert_eq!(pixel(&capture(&mut world), point), shown);
    world.close();
}

fn save_and_reopen(world: &mut ApplicationWorld) {
    use oblikovati_api::{
        client::Client,
        wire::{CloseDocumentArgs, OpenDocumentArgs, SaveDocumentAsArgs},
    };
    let path = world.scratch_folder().join("displayed-plane.opd");
    let document = world.active_id();
    let name = path.to_str().unwrap().to_owned();
    let client = Client::new(world.app().owner.document_host());
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: name.clone(),
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: name,
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    camera(world);
}

#[test]
fn visible_plane_survives_save_and_reopen() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    camera(&mut world);
    let before = capture(&mut world);
    create_plane(&mut world);
    let point = center(&mut world);
    let shown = pixel(&capture(&mut world), point);
    assert_ne!(shown, pixel(&before, point));
    save_and_reopen(&mut world);
    assert_eq!(center(&mut world), point);
    assert_eq!(pixel(&capture(&mut world), point), shown);
    world.close();
}

#[test]
fn a_hidden_plane_stays_hidden_after_reopen() {
    use oblikovati_api::{client::Client, types::WORK_REF_XY_PLANE, wire::CreateWorkPlaneArgs};
    let mut world = ApplicationWorld::new();
    world.add_part();
    camera(&mut world);
    let before = capture(&mut world);
    Client::new(world.app().owner.document_host())
        .work_planes()
        .create(&CreateWorkPlaneArgs {
            kind: "plane-offset".into(),
            refs: Some(vec![WORK_REF_XY_PLANE.into()]),
            offset: "10 mm".into(),
            visible: Some(Box::new(false)),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    assert!(world.app().owner.datum_planes().unwrap().planes.is_empty());
    save_and_reopen(&mut world);
    assert!(world.app().owner.datum_planes().unwrap().planes.is_empty());
    let point = world
        .app()
        .window
        .native
        .view
        .model_pixel([0., 0., 1.])
        .unwrap();
    assert_eq!(pixel(&capture(&mut world), point), pixel(&before, point));
    world.close();
}
