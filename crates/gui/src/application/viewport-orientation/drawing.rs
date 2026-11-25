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
        let [width, height] = ui.calc_text_size(letter);
        ui.get_window_draw_list().add_text(
            [at[0] - width / 2., at[1] - height / 2.],
            color,
            letter,
        );
    }
}
pub(super) fn draw_control(
    ui: &Ui,
    axes: &GuiViewAxes,
    layout: &GuiOrientationLayout,
    origin: [f32; 2],
    faces: &[ShownFace],
    hit: Option<&hotspots::CubeHit>,
    [alpha, chrome]: [f32; 2],
) {
    let list = ui.get_window_draw_list();
    let background = ui.style_color(StyleColor::WindowBg);
    let paint = cube_paint(background);
    draw_contact_shadow(&list, faces, layout.half, alpha);
    let at = |point: [f32; 2]| [origin[0] + point[0], origin[1] + point[1]];
    for shown in faces.iter().rev() {
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
    if let Some(hit) = hit {
        let accent = ui.style_color(StyleColor::NavHighlight);
        list.add_polyline(hit.polygon.to_vec(), faded(accent, 0.32 * chrome))
            .filled(true)
            .build();
        let mut border = hit.polygon.to_vec();
        border.push(hit.polygon[0]);
        list.add_polyline(border, faded(accent, chrome))
            .thickness(1.5)
            .build();
    }
    if chrome > 0. {
        let ink = faded(ui.style_color(StyleColor::Text), chrome);
        draw_home(&list, at(layout.home), ink);
        let [x, y] = at(layout.menu);
        list.add_triangle([x - 4., y - 2.], [x + 4., y - 2.], [x, y + 2.], ink)
            .filled(true)
            .build();
        if is_face_on(axes) {
            draw_arrows(&list, layout, origin, ink);
        }
    }
    drop(list);
    // Keep dark ink readable over the Light face while it fades. Zero still hides the control.
    let label_alpha = if background[0] > 0.5 && alpha > 0. {
        1.
    } else {
        alpha
    };
    for shown in faces.iter().filter(|shown| shown.facing > LABEL_SHOWS_FROM) {
        draw_face_label(ui, shown, layout.half, faded(paint.label, label_alpha));
    }
}
fn draw_face_label(ui: &Ui, shown: &ShownFace, half: f32, color: [f32; 4]) {
    let center: [f32; 2] = std::array::from_fn(|axis| {
        shown.corners.iter().map(|corner| corner[axis]).sum::<f32>() / 4.
    });
    crate::widget_abi::projected_text(
        ui,
        center,
        color,
        &shown.face.label().to_ascii_uppercase(),
        shown.label_basis(half),
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
        [0x3b4152, 0x2b303e, 0x1e222d, 0x4b5368, 0xeceef2]
    }
    .map(crate::theme::hex_rgb);
    CubePaint {
        faces: [top, front, side],
        edge,
        label,
    }
}
fn draw_contact_shadow(list: &imgui::DrawListMut, faces: &[ShownFace], half: f32, alpha: f32) {
    let Some(first) = faces.first() else {
        return;
    };
    let mut left = first.corners[0][0];
    let mut right = left;
    let mut bottom = first.corners[0][1];
    for &[x, y] in faces.iter().flat_map(|face| &face.corners) {
        left = left.min(x);
        right = right.max(x);
        bottom = bottom.max(y);
    }
    let center = (left + right) / 2.;
    for (width, height) in [(1.2, 0.14), (1., 0.10), (0.8, 0.06)] {
        let outline = (0..24)
            .map(|index| {
                let (sin, cos) = (index as f32 * std::f32::consts::TAU / 24.).sin_cos();
                [
                    center + half * width * cos,
                    bottom + 2. + half * height * sin,
                ]
            })
            .collect();
        list.add_polyline(outline, [0., 0., 0., 0.08 * alpha])
            .filled(true)
            .build();
    }
}

fn draw_home(list: &imgui::DrawListMut, [x, y]: [f32; 2], color: [f32; 4]) {
    list.add_polyline(vec![[x - 7., y], [x, y - 6.], [x + 7., y]], color)
        .thickness(1.4)
        .build();
    list.add_polyline(
        vec![
            [x - 5., y - 1.],
            [x - 5., y + 6.],
            [x + 5., y + 6.],
            [x + 5., y - 1.],
        ],
        color,
    )
    .thickness(1.4)
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
    draw_roll_arc(list, at(layout.roll_upper), 1., color);
    draw_roll_arc(list, at(layout.roll_lower), -1., color);
}

fn draw_roll_arc(list: &imgui::DrawListMut, center: [f32; 2], direction: f32, color: [f32; 4]) {
    let arc: Vec<_> = (0..=12)
        .map(|step| {
            let angle = 0.8 + step as f32 * 3.8 / 12.;
            [
                center[0] + direction * 5. * angle.cos(),
                center[1] + 5. * angle.sin(),
            ]
        })
        .collect();
    list.add_polyline(arc.clone(), color).thickness(1.4).build();
    let [tip_x, tip_y] = arc[12];
    list.add_polyline(
        vec![
            [tip_x - direction * 4., tip_y - 3.],
            [tip_x, tip_y],
            [tip_x - direction * 4., tip_y + 3.],
        ],
        color,
    )
    .thickness(1.4)
    .build();
}
