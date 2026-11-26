// SPDX-License-Identifier: GPL-2.0-only
//! Camera frames at the native window boundary.
use crate::HeadWindow;
use oblikovati_api::{
    Error,
    client::Caller,
    types::{self, Point, ProjectionTypeEnum, Vector},
    wire::{self, CameraView, GetCameraArgs, ListDocumentsResult, SetCameraArgs},
};
use oblikovati_automation::{decode_wire_arguments, encode_wire_reply};
use oblikovati_gui::GuiViewProjection;
use oblikovati_viewport::ViewportCamera;
use std::collections::BTreeMap;

pub(crate) fn apply(
    caller: &impl Caller,
    window: &mut dyn HeadWindow,
    slots: &BTreeMap<u64, u64>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    if method == wire::METHOD_VIEW_GET_CAMERA {
        let args: GetCameraArgs = if request.is_empty() {
            GetCameraArgs::default()
        } else {
            decode_wire_arguments(method, request)?
        };
        let slot = resolve_slot(caller, slots, args.document)?;
        return encode_wire_reply(&window.wire_camera(slot).map_err(Error)?);
    }
    let args: SetCameraArgs = decode_wire_arguments(method, request)?;
    let slot = resolve_slot(caller, slots, args.document)?;
    encode_wire_reply(&window.set_wire_camera(slot, &args).map_err(Error)?)
}

fn resolve_slot(
    caller: &impl Caller,
    slots: &BTreeMap<u64, u64>,
    requested: u64,
) -> Result<u64, Error> {
    // Resolve against the current owner on each call. A document may have opened or
    // activated earlier in this same owner poll, before the window advances a frame.
    let reply = caller.call(wire::METHOD_DOCUMENTS_LIST, b"{}")?;
    let list: ListDocumentsResult = serde_json::from_slice(&reply)
        .map_err(|cause| Error(format!("camera document lookup: {cause}")))?;
    let open = list.documents.unwrap_or_default();
    let document = open
        .iter()
        .find(|document| {
            if requested == 0 {
                document.active
            } else {
                document.id == requested
            }
        })
        .ok_or_else(|| {
            Error(format!(
                "camera document {requested}: expected an open document"
            ))
        })?;
    Ok(slots.get(&document.id).copied().unwrap_or(document.id))
}

pub(crate) fn prepare(
    current: ViewportCamera,
    mode: GuiViewProjection,
    args: &SetCameraArgs,
) -> Result<(ViewportCamera, GuiViewProjection, CameraView), String> {
    let projection = parse_projection(args.projection, mode)?;
    let eye = [args.eye.x, args.eye.y, args.eye.z];
    let target = [args.target.x, args.target.y, args.target.z];
    let up = [args.up.x, args.up.y, args.up.z];
    let distance = eye_distance(eye, target);
    if !(args.fov > 0. && args.fov < std::f64::consts::PI) || distance < 1e-9 {
        return Err(format!(
            "camera eye {eye:?}, target {target:?}, fov {}: expected distinct points and radians in (0, π)",
            args.fov
        ));
    }
    let mut camera = ViewportCamera {
        eye,
        target,
        up,
        scale: distance * (args.fov / 2.).tan() * 2.,
        ..current
    };
    oblikovati_model::part_design_view::validate_part_camera(&camera.part_view_camera())
        .map_err(|cause| cause.to_string())?;
    camera.perspective = crate::view_orientation::projects_in_perspective(&camera, projection);
    let mut authored = wire_frame(camera, projection);
    authored.fov = args.fov;
    Ok((camera, projection, authored))
}

pub(crate) fn wire_frame(camera: ViewportCamera, projection: GuiViewProjection) -> CameraView {
    let [x, y, z] = camera.eye;
    let eye = Point { x, y, z };
    let [x, y, z] = camera.target;
    let target = Point { x, y, z };
    let [x, y, z] = camera.up;
    CameraView {
        eye,
        target,
        up: Vector { x, y, z },
        fov: 2. * (camera.scale / 2. / eye_distance(camera.eye, camera.target)).atan(),
        projection: projection_enum(projection),
    }
}

fn eye_distance(eye: [f64; 3], target: [f64; 3]) -> f64 {
    let [x, y, z] = std::array::from_fn(|axis| eye[axis] - target[axis]);
    x.hypot(y).hypot(z)
}

fn projection_enum(mode: GuiViewProjection) -> ProjectionTypeEnum {
    match mode {
        GuiViewProjection::Orthographic => types::ORTHOGRAPHIC_PROJECTION,
        GuiViewProjection::Perspective => types::PERSPECTIVE_PROJECTION,
        GuiViewProjection::PerspectiveWithOrthoFaces => {
            types::PERSPECTIVE_WITH_ORTHO_FACES_PROJECTION
        }
    }
}

fn parse_projection(
    value: ProjectionTypeEnum,
    current: GuiViewProjection,
) -> Result<GuiViewProjection, String> {
    match value {
        ProjectionTypeEnum(0) => Ok(current),
        types::ORTHOGRAPHIC_PROJECTION => Ok(GuiViewProjection::Orthographic),
        types::PERSPECTIVE_PROJECTION => Ok(GuiViewProjection::Perspective),
        types::PERSPECTIVE_WITH_ORTHO_FACES_PROJECTION => {
            Ok(GuiViewProjection::PerspectiveWithOrthoFaces)
        }
        other => Err(format!(
            "camera projection {}: expected a supported projection",
            other.0
        )),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn request() -> SetCameraArgs {
        SetCameraArgs {
            eye: Point {
                x: 42.,
                y: 17.,
                z: 99.,
            },
            target: Point {
                x: 1.,
                y: 2.,
                z: 3.,
            },
            up: Vector {
                x: 0.,
                y: 1.,
                z: 0.,
            },
            fov: 0.6,
            ..Default::default()
        }
    }

    #[test]
    fn omitted_projection_keeps_the_view_and_fov_sets_its_height() {
        let request = request();
        let current = ViewportCamera::default();
        let (camera, projection, reply) =
            prepare(current, GuiViewProjection::Perspective, &request).unwrap();
        assert_eq!(projection, GuiViewProjection::Perspective);
        assert!(camera.perspective);
        let expected =
            2. * (41_f64.powi(2) + 15_f64.powi(2) + 96_f64.powi(2)).sqrt() * 0.3_f64.tan();
        assert!((camera.scale - expected).abs() < 1e-12);
        assert_eq!(
            (reply.eye, reply.target, reply.up, reply.fov),
            (request.eye, request.target, request.up, request.fov)
        );
    }

    #[test]
    fn rejects_parallel_up_and_invalid_projection() {
        let current = ViewportCamera::default();
        let mut request = request();
        request.up = Vector {
            x: 41.,
            y: 15.,
            z: 96.,
        };
        assert!(prepare(current, GuiViewProjection::Orthographic, &request).is_err());
        request.up = Vector {
            x: 0.,
            y: 1.,
            z: 0.,
        };
        request.projection = ProjectionTypeEnum(-1);
        assert!(prepare(current, GuiViewProjection::Orthographic, &request).is_err());
    }

    struct OpenDocuments;

    impl Caller for OpenDocuments {
        fn call(&self, method: &str, _: &[u8]) -> Result<Vec<u8>, Error> {
            assert_eq!(method, wire::METHOD_DOCUMENTS_LIST);
            encode_wire_reply(&ListDocumentsResult {
                documents: Some(vec![
                    wire::DocumentInfo {
                        id: 4,
                        active: true,
                        ..Default::default()
                    },
                    wire::DocumentInfo {
                        id: 8,
                        ..Default::default()
                    },
                ]),
            })
        }
    }

    #[test]
    fn resolves_active_and_background_documents_to_their_selected_slots() {
        let slots = BTreeMap::from([(4, 10), (8, 12)]);
        assert_eq!(resolve_slot(&OpenDocuments, &slots, 0).unwrap(), 10);
        assert_eq!(resolve_slot(&OpenDocuments, &slots, 8).unwrap(), 12);
        assert_eq!(
            resolve_slot(&OpenDocuments, &BTreeMap::new(), 8).unwrap(),
            8
        );
        assert!(resolve_slot(&OpenDocuments, &slots, 9).is_err());
    }
}

#[cfg(all(test, target_os = "linux"))]
#[test]
fn native_camera_restores_the_background_slot_and_projects_the_requested_frame() {
    let mut window = crate::NativeHeadWindow::open().unwrap();
    window.set_view_slot(1).unwrap();
    let first = window.wire_camera(1).unwrap();
    let args = SetCameraArgs {
        eye: Point {
            x: 42.,
            y: 17.,
            z: 99.,
        },
        target: Point {
            x: 1.,
            y: 2.,
            z: 3.,
        },
        up: Vector {
            x: 0.,
            y: 2.,
            z: 0.,
        },
        fov: 0.6,
        projection: types::PERSPECTIVE_PROJECTION,
        ..Default::default()
    };
    let authored = window.set_wire_camera(2, &args).unwrap();
    assert_eq!(window.wire_camera(1).unwrap(), first);
    assert_eq!(window.wire_camera(2).unwrap(), authored);
    window.set_view_slot(2).unwrap();
    let rendered = window.view_camera().unwrap();
    assert!(rendered.perspective);
    assert_eq!(rendered.eye, [42., 17., 99.]);
    for (actual, expected) in rendered.target.into_iter().zip([1., 2., 3.]) {
        assert!((actual - expected).abs() < 1e-12);
    }
    assert_eq!(window.wire_camera(2).unwrap(), authored);
    window.set_view_slot(1).unwrap();
    assert_eq!(window.wire_camera(1).unwrap(), first);
    window.finish().unwrap();
}
