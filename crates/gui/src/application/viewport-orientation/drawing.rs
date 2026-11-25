// SPDX-License-Identifier: GPL-2.0-only
//! How the orientation control looks: the cube, its labels and arrows, and the compass.
use super::*;
fn faded(color: [f32; 4], alpha: f32) -> [f32; 4] {
    [color[0], color[1], color[2], color[3] * alpha]
}
pub(super) fn draw_compass(
    ui: &Ui,
    layout: &GuiOrientationLayout,
    origin: [f32; 2],
    alpha: f32,
    north_degrees: f32,
) {
    let center = [origin[0] + layout.center[0], origin[1] + layout.center[1]];
    let radius = layout.half + COMPASS_REACH;
    let color = faded(ui.style_color(StyleColor::Text), alpha);
    ui.get_window_draw_list()
        .add_circle(center, radius, color)
        .num_segments(48)
        .build();
    // North turns clockwise by its angle; the other letters keep their quarter turns from it.
    for (letter, quarter) in [("N", 0.), ("E", 90.), ("S", 180.), ("W", 270.)] {
        let (sin, cos) = (north_degrees + quarter).to_radians().sin_cos();
        let at = [center[0] + sin * radius, center[1] - cos * radius];
        crate::widget_abi::rotated_text(ui, at, color, letter, 0.);
    }
}
pub(super) fn draw_control(
    ui: &Ui,
    axes: &GuiViewAxes,
    layout: &GuiOrientationLayout,
    origin: [f32; 2],
    alpha: f32,
) {
    let list = ui.get_window_draw_list();
    let background = ui.style_color(StyleColor::WindowBg);
    let paint = cube_paint(background);
    let mut faces = shown_faces(axes, layout, origin);
    faces.reverse();
    let at = |point: [f32; 2]| [origin[0] + point[0], origin[1] + point[1]];
    for shown in &faces {
        let fill = faded(paint.fill(shown.face), alpha);
        list.add_polyline(shown.corners.to_vec(), fill)
            .filled(true)
            .build();
        for index in 0..4 {
            let next = shown.corners[(index + 1) % 4];
            list.add_line(shown.corners[index], next, faded(paint.edge, alpha))
                .build();
        }
    }
    let accent = faded(ui.style_color(StyleColor::Text), alpha);
    draw_home(&list, at(layout.home), accent);
    if is_face_on(axes) {
        draw_arrows(&list, layout, origin, accent);
    }
    drop(list);
    // Keep dark ink readable over the Light face while it fades. Zero still hides the control.
    let label_alpha = if background[0] > 0.5 && alpha > 0. {
        1.
    } else {
        alpha
    };
    for shown in faces.iter().filter(|shown| shown.facing > LABEL_SHOWS_FROM) {
        draw_face_label(ui, shown, axes, faded(paint.label, label_alpha));
    }
}
fn draw_face_label(ui: &Ui, shown: &ShownFace, axes: &GuiViewAxes, color: [f32; 4]) {
    let center: [f32; 2] = std::array::from_fn(|axis| {
        shown.corners.iter().map(|corner| corner[axis]).sum::<f32>() / 4.
    });
    let reading = shown.face.reading();
    let angle = (-dot(reading, axes.up)).atan2(dot(reading, axes.right));
    crate::widget_abi::rotated_text(
        ui,
        center,
        color,
        &shown.face.label().to_ascii_uppercase(),
        angle,
    );
}

struct CubePaint {
    faces: [[f32; 4]; 3],
    edge: [f32; 4],
    label: [f32; 4],
}
impl CubePaint {
    fn fill(&self, face: GuiCubeFace) -> [f32; 4] {
        self.faces[match face {
            GuiCubeFace::Top | GuiCubeFace::Bottom => 0,
            GuiCubeFace::Front | GuiCubeFace::Back => 1,
            GuiCubeFace::Left | GuiCubeFace::Right => 2,
        }]
    }
}
fn cube_paint(background: [f32; 4]) -> CubePaint {
    let [top, front, side, edge, label] = if background[0] > 0.5 {
        [0xffffff, 0xeef1f6, 0xdce1eb, 0xaeb6c6, 0x1e2430]
    } else if background[..3].iter().all(|channel| *channel < 0.01) {
        [0x212734, 0x0a0a0d, 0x141820, 0xa7aebd, 0xffffff]
    } else {
        [0x323744, 0x252934, 0x1c1f28, 0x4b5368, 0xeceef2]
    }
    .map(crate::theme::hex_rgb);
    CubePaint {
        faces: [top, front, side],
        edge,
        label,
    }
}
fn draw_home(list: &imgui::DrawListMut, [x, y]: [f32; 2], color: [f32; 4]) {
    list.add_triangle([x - 6., y], [x + 6., y], [x, y - 6.], color)
        .filled(true)
        .build();
    list.add_rect([x - 4., y], [x + 4., y + 5.], color)
        .filled(true)
        .build();
}
fn draw_arrows(
    list: &imgui::DrawListMut,
    layout: &GuiOrientationLayout,
    origin: [f32; 2],
    color: [f32; 4],
) {
    let at = |point: [f32; 2]| [origin[0] + point[0], origin[1] + point[1]];
    const R: f32 = 5.;
    let triangle = |tip: [f32; 2], base: [f32; 2]| {
        let across = [-(tip[1] - base[1]), tip[0] - base[0]];
        list.add_triangle(
            tip,
            [base[0] + across[0], base[1] + across[1]],
            [base[0] - across[0], base[1] - across[1]],
            color,
        )
        .filled(true)
        .build();
    };
    let [x, y] = at(layout.step_right);
    triangle([x + R, y], [x - R / 2., y]);
    let [x, y] = at(layout.step_left);
    triangle([x - R, y], [x + R / 2., y]);
    let [x, y] = at(layout.step_up);
    triangle([x, y - R], [x, y + R / 2.]);
    let [x, y] = at(layout.step_down);
    triangle([x, y + R], [x, y - R / 2.]);
    for (center, tip) in [(layout.roll_upper, R), (layout.roll_lower, -R)] {
        let [x, y] = at(center);
        list.add_circle([x, y], R, color).num_segments(10).build();
        triangle([x + R + 2., y + tip], [x + R - 3., y]);
    }
}
