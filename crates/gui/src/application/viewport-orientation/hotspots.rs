// SPDX-License-Identifier: GPL-2.0-only
//! Use the same projected regions for hover ink and orientation commands.
use super::*;
use cube::inside;

pub(super) struct CubeHit {
    pub(super) polygon: [[f32; 2]; 4],
    pub(super) command: GuiOrientCommand,
}

pub(super) fn cube_hit(
    point: [f32; 2],
    axes: &GuiViewAxes,
    layout: &GuiOrientationLayout,
    faces: &[ShownFace],
) -> Option<CubeHit> {
    let shown = faces.iter().find(|face| inside(point, &face.corners))?;
    let [[rx, ry], [dx, dy]] = shown.label_basis(layout.half);
    let [left, bottom] = shown.corners[0];
    let [x, y] = [point[0] - left, point[1] - bottom];
    let det = (rx * dy - ry * dx) * layout.half;
    let across = (x * dy - y * dx) / det - 1.;
    let down = (rx * y - ry * x) / det + 1.;
    let face_on = shown.facing > FACE_ON;
    let boundary = if face_on { 0.6 } else { 0.84 };
    let (horizontal, vertical) = (strip(across, boundary), strip(down, boundary));
    let command = hit_command(shown.face, axes, face_on, horizontal, vertical);
    let [xmin, xmax] = limits(horizontal, boundary);
    let [ymin, ymax] = limits(vertical, boundary);
    let polygon = [(xmin, ymax), (xmax, ymax), (xmax, ymin), (xmin, ymin)].map(|(u, v)| {
        [
            left + layout.half * (rx * (u + 1.) + dx * (v - 1.)),
            bottom + layout.half * (ry * (u + 1.) + dy * (v - 1.)),
        ]
    });
    Some(CubeHit { polygon, command })
}

fn strip(value: f32, boundary: f32) -> i8 {
    if value > boundary {
        1
    } else if value < -boundary {
        -1
    } else {
        0
    }
}

fn limits(side: i8, boundary: f32) -> [f32; 2] {
    match side {
        -1 => [-1., -boundary],
        1 => [boundary, 1.],
        _ => [-boundary, boundary],
    }
}

fn hit_command(
    face: GuiCubeFace,
    axes: &GuiViewAxes,
    face_on: bool,
    horizontal: i8,
    vertical: i8,
) -> GuiOrientCommand {
    if horizontal == 0 && vertical == 0 {
        return GuiOrientCommand::Face(face);
    }
    if !face_on {
        let normal = face.normal();
        let reading = face.reading();
        let up = cross(normal, reading);
        return GuiOrientCommand::Bevel(std::array::from_fn(|axis| {
            normal[axis] as i8 + reading[axis] as i8 * horizontal - up[axis] as i8 * vertical
        }));
    }
    face_on_rim(face, axes, horizontal, vertical)
}

fn face_on_rim(
    face: GuiCubeFace,
    axes: &GuiViewAxes,
    horizontal: i8,
    vertical: i8,
) -> GuiOrientCommand {
    // The strips rotate with the label basis when the face is rolled.
    let reading = face.reading();
    let up = cross(face.normal(), reading);
    let offset =
        std::array::from_fn(|i| reading[i] * f32::from(horizontal) - up[i] * f32::from(vertical));
    if horizontal != 0 && vertical != 0 {
        return GuiOrientCommand::Corner {
            right: dot(offset, axes.right) > 0.,
            up: dot(offset, axes.up) > 0.,
        };
    }
    GuiOrientCommand::Face(GuiCubeFace::nearest(offset))
}
