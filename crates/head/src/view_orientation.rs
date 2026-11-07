// SPDX-License-Identifier: GPL-2.0-only
//! What the orientation control reads from the camera, and what it asks the camera to become.
use oblikovati_gui::{GuiCubeFace, GuiOrientCommand, GuiViewAxes, GuiViewProjection, GuiViewStep};
use oblikovati_viewport::ViewportCamera;

type Vector = [f64; 3];
/// How close to an axis the view direction counts as looking straight at a face.
const FACE_ON: f64 = 0.999;
fn dot(a: Vector, b: Vector) -> f64 {
    (0..3).map(|axis| a[axis] * b[axis]).sum()
}
fn cross(a: Vector, b: Vector) -> Vector {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}
fn unit(value: Vector) -> Vector {
    let length = dot(value, value).sqrt();
    value.map(|part| part / length)
}
fn negated(value: Vector) -> Vector {
    value.map(|part| -part)
}
/// The screen's directions in the model, from where the camera stands.
fn frame(camera: &ViewportCamera) -> (Vector, Vector, Vector) {
    let toward = unit(std::array::from_fn(|axis| {
        camera.eye[axis] - camera.target[axis]
    }));
    let right = unit(cross(camera.up, toward));
    (right, cross(toward, right), toward)
}
/// The control's view of the camera; example: each frame, to draw the cube.
pub(crate) fn view_axes(camera: &ViewportCamera) -> GuiViewAxes {
    let (right, up, toward) = frame(camera);
    let narrow = |value: Vector| value.map(|part| part as f32);
    GuiViewAxes {
        right: narrow(right),
        up: narrow(up),
        toward: narrow(toward),
    }
}
fn looking(camera: &ViewportCamera, toward: Vector, up: Vector) -> ViewportCamera {
    let delta: Vector = std::array::from_fn(|axis| camera.eye[axis] - camera.target[axis]);
    let distance = dot(delta, delta).sqrt();
    ViewportCamera {
        eye: std::array::from_fn(|axis| camera.target[axis] + toward[axis] * distance),
        up,
        ..*camera
    }
}
/// A named face reads upright; Top and Bottom keep Front on the side nearest the viewer's feet.
fn face_view(face: GuiCubeFace) -> (Vector, Vector) {
    let normal = face.normal().map(f64::from);
    let up = match face {
        GuiCubeFace::Top => [0., 0., -1.],
        GuiCubeFace::Bottom => [0., 0., 1.],
        _ => [0., 1., 0.],
    };
    (normal, up)
}
/// Whether this viewpoint draws in perspective; the hybrid keeps a face-on view flat.
pub(crate) fn projects_in_perspective(camera: &ViewportCamera, mode: GuiViewProjection) -> bool {
    match mode {
        GuiViewProjection::Orthographic => false,
        GuiViewProjection::Perspective => true,
        GuiViewProjection::PerspectiveWithOrthoFaces => {
            let (_, _, toward) = frame(camera);
            !toward.iter().any(|part| part.abs() > FACE_ON)
        }
    }
}
/// The camera a press on the control asks for; example: `oriented(&camera, command)`.
/// Scale and distance stay, so only the viewpoint changes.
pub(crate) fn oriented(camera: &ViewportCamera, command: GuiOrientCommand) -> ViewportCamera {
    let (right, up, toward) = frame(camera);
    match command {
        GuiOrientCommand::Face(face) => {
            let (normal, face_up) = face_view(face);
            looking(camera, normal, face_up)
        }
        GuiOrientCommand::Corner {
            right: to_right,
            up: to_up,
        } => {
            let across = if to_right { right } else { negated(right) };
            let over = if to_up { up } else { negated(up) };
            let corner = unit(std::array::from_fn(|axis| {
                toward[axis] + across[axis] + over[axis]
            }));
            looking(camera, corner, up)
        }
        GuiOrientCommand::Step(GuiViewStep::Right) => looking(camera, right, up),
        GuiOrientCommand::Step(GuiViewStep::Left) => looking(camera, negated(right), up),
        GuiOrientCommand::Step(GuiViewStep::Up) => looking(camera, up, negated(toward)),
        GuiOrientCommand::Step(GuiViewStep::Down) => looking(camera, negated(up), toward),
        // The face turns on screen with the label; the viewpoint stays.
        GuiOrientCommand::Roll { clockwise: true } => looking(camera, toward, negated(right)),
        GuiOrientCommand::Roll { clockwise: false } => looking(camera, toward, right),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    fn front() -> ViewportCamera {
        ViewportCamera::default()
    }
    fn close(a: Vector, b: Vector) -> bool {
        (0..3).all(|axis| (a[axis] - b[axis]).abs() < 1e-9)
    }
    #[test]
    fn stepping_right_then_left_returns_to_the_same_view() {
        let right = oriented(&front(), GuiOrientCommand::Step(GuiViewStep::Right));
        assert!(close(right.eye, [20., 0., 0.]), "{:?}", right.eye);
        let back = oriented(&right, GuiOrientCommand::Step(GuiViewStep::Left));
        assert!(close(back.eye, front().eye), "{:?}", back.eye);
        assert_eq!(back.scale, front().scale);
    }
    #[test]
    fn a_roll_turns_the_screen_about_the_viewer_and_its_opposite_undoes_it() {
        let turned = oriented(&front(), GuiOrientCommand::Roll { clockwise: true });
        assert!(close(turned.eye, front().eye), "{:?}", turned.eye);
        assert!(close(turned.up, [-1., 0., 0.]), "{:?}", turned.up);
        let back = oriented(&turned, GuiOrientCommand::Roll { clockwise: false });
        assert!(close(back.up, [0., 1., 0.]), "{:?}", back.up);
    }
    #[test]
    fn the_hybrid_is_flat_face_on_and_in_perspective_otherwise() {
        let hybrid = GuiViewProjection::PerspectiveWithOrthoFaces;
        assert!(!projects_in_perspective(&front(), hybrid));
        let corner = oriented(
            &front(),
            GuiOrientCommand::Corner {
                right: true,
                up: true,
            },
        );
        assert!(projects_in_perspective(&corner, hybrid));
        assert!(projects_in_perspective(
            &front(),
            GuiViewProjection::Perspective
        ));
        assert!(!projects_in_perspective(
            &corner,
            GuiViewProjection::Orthographic
        ));
    }
    #[test]
    fn the_top_face_looks_down_and_a_corner_looks_from_three_sides() {
        let top = oriented(&front(), GuiOrientCommand::Face(GuiCubeFace::Top));
        assert!(close(top.eye, [0., 20., 0.]), "{:?}", top.eye);
        let corner = oriented(
            &front(),
            GuiOrientCommand::Corner {
                right: true,
                up: true,
            },
        );
        let axes = view_axes(&corner);
        assert!(axes.toward.iter().all(|part| *part > 0.5), "{axes:?}");
    }
}
