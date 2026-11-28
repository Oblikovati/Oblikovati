// SPDX-License-Identifier: GPL-2.0-only
//! Interactive live cross-section diagram for the Hole Properties panel.
use super::types::{GuiHoleSeatKind, GuiHoleTermination};
use imgui::Ui;

pub(super) struct HoleDiagramSpec<'a> {
    pub seat: GuiHoleSeatKind,
    pub termination: GuiHoleTermination,
    pub drill_point_angled: bool,
    pub diameter: &'a str,
    pub depth: &'a str,
    pub seat_diameter: &'a str,
    pub seat_depth: &'a str,
    pub seat_angle: &'a str,
    pub drill_point_angle: &'a str,
    pub invalid: bool,
}

/// Draw the live Hole cross-section diagram with dimension callouts.
pub(super) fn draw_hole_diagram(ui: &Ui, spec: &HoleDiagramSpec<'_>) {
    let width = ui.content_region_avail()[0].max(160.0);
    let height = 58.0;
    let origin = ui.cursor_screen_pos();
    ui.dummy([width, height]);
    let dl = ui.get_window_draw_list();

    let bg = [0.13, 0.14, 0.16, 1.0];
    let border = if spec.invalid {
        [0.86, 0.25, 0.25, 1.0]
    } else {
        [0.28, 0.30, 0.35, 1.0]
    };
    let outline = [0.82, 0.86, 0.92, 1.0];
    let dim = [0.46, 0.78, 0.96, 1.0];
    let p_max = [origin[0] + width, origin[1] + height];
    dl.add_rect(origin, p_max, bg)
        .filled(true)
        .rounding(3.0)
        .build();
    dl.add_rect(origin, p_max, border)
        .rounding(3.0)
        .thickness(1.0)
        .build();

    let cx = origin[0] + width * 0.44;
    let top_y = origin[1] + 18.0;
    let bot_y = origin[1] + height - 14.0;
    let bore_half = 14.0;
    let seat_half = 24.0;
    let seat_y = top_y + 12.0;

    // Top material surface lines
    dl.add_line(
        [origin[0] + 14.0, top_y],
        [cx - seat_half - 10.0, top_y],
        outline,
    )
    .thickness(1.2)
    .build();
    dl.add_line(
        [cx + seat_half + 10.0, top_y],
        [p_max[0] - 14.0, top_y],
        outline,
    )
    .thickness(1.2)
    .build();

    // Seat + bore walls
    match spec.seat {
        GuiHoleSeatKind::None => {
            dl.add_line(
                [cx - seat_half - 10.0, top_y],
                [cx - bore_half, top_y],
                outline,
            )
            .thickness(1.2)
            .build();
            dl.add_line(
                [cx + bore_half, top_y],
                [cx + seat_half + 10.0, top_y],
                outline,
            )
            .thickness(1.2)
            .build();
            dl.add_line([cx - bore_half, top_y], [cx - bore_half, bot_y], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx + bore_half, top_y], [cx + bore_half, bot_y], outline)
                .thickness(1.5)
                .build();
        }
        GuiHoleSeatKind::Counterbore | GuiHoleSeatKind::Spotface => {
            let sy = if spec.seat == GuiHoleSeatKind::Spotface {
                top_y + 10.0
            } else {
                seat_y
            };
            dl.add_line(
                [cx - seat_half - 10.0, top_y],
                [cx - seat_half, top_y],
                outline,
            )
            .thickness(1.2)
            .build();
            dl.add_line(
                [cx + seat_half, top_y],
                [cx + seat_half + 10.0, top_y],
                outline,
            )
            .thickness(1.2)
            .build();
            dl.add_line([cx - seat_half, top_y], [cx - seat_half, sy], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx + seat_half, top_y], [cx + seat_half, sy], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx - seat_half, sy], [cx - bore_half, sy], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx + bore_half, sy], [cx + seat_half, sy], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx - bore_half, sy], [cx - bore_half, bot_y], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx + bore_half, sy], [cx + bore_half, bot_y], outline)
                .thickness(1.5)
                .build();
            dl.add_text([cx - 18.0, origin[1] + 5.0], dim, spec.seat_diameter);
            dl.add_text([cx + seat_half + 6.0, top_y + 2.0], dim, spec.seat_depth);
        }
        GuiHoleSeatKind::Countersink => {
            dl.add_line(
                [cx - seat_half - 10.0, top_y],
                [cx - seat_half, top_y],
                outline,
            )
            .thickness(1.2)
            .build();
            dl.add_line(
                [cx + seat_half, top_y],
                [cx + seat_half + 10.0, top_y],
                outline,
            )
            .thickness(1.2)
            .build();
            dl.add_line([cx - seat_half, top_y], [cx - bore_half, seat_y], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx + seat_half, top_y], [cx + bore_half, seat_y], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx - bore_half, seat_y], [cx - bore_half, bot_y], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx + bore_half, seat_y], [cx + bore_half, bot_y], outline)
                .thickness(1.5)
                .build();
            dl.add_text([cx - 18.0, origin[1] + 5.0], dim, spec.seat_diameter);
            let angle_text = format_angle_callout(spec.seat_angle);
            dl.add_text([cx + seat_half + 6.0, top_y + 2.0], dim, &angle_text);
        }
    }

    // Bottom / Drill point profile
    if spec.termination == GuiHoleTermination::Distance {
        if spec.drill_point_angled {
            let tip_y = bot_y + 10.0;
            dl.add_line([cx - bore_half, bot_y], [cx, tip_y], outline)
                .thickness(1.5)
                .build();
            dl.add_line([cx + bore_half, bot_y], [cx, tip_y], outline)
                .thickness(1.5)
                .build();
            let point_text = format_angle_callout(spec.drill_point_angle);
            dl.add_text([cx + bore_half + 8.0, bot_y - 4.0], dim, &point_text);
        } else {
            dl.add_line([cx - bore_half, bot_y], [cx + bore_half, bot_y], outline)
                .thickness(1.5)
                .build();
            dl.add_text([cx + bore_half + 8.0, bot_y - 6.0], dim, "Flat");
        }
        dl.add_text(
            [origin[0] + 8.0, (top_y + bot_y) * 0.5 - 6.0],
            dim,
            spec.depth,
        );
    } else {
        // Open through / to-surface dashed bottom
        dl.add_line([cx - bore_half, bot_y], [cx - 4.0, bot_y], dim)
            .thickness(1.0)
            .build();
        dl.add_line([cx + 4.0, bot_y], [cx + bore_half, bot_y], dim)
            .thickness(1.0)
            .build();
    }

    // Bore diameter callout
    let bore_callout = format!("\u{2300} {}", spec.diameter);
    dl.add_text([cx - 22.0, bot_y - 16.0], dim, &bore_callout);
}

fn format_angle_callout(raw: &str) -> String {
    let trimmed = raw.trim();
    if trimmed.contains('\u{00b0}') {
        trimmed.to_owned()
    } else {
        let cleaned = trimmed.trim_end_matches("deg").trim();
        format!("{cleaned}\u{00b0}")
    }
}
