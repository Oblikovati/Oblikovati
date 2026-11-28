// SPDX-License-Identifier: GPL-2.0-only
use crate::application::workspace_layout::browser_view::BrowserView;
use crate::icons::{IconGlyph, IconSize};
use imgui::{StyleColor, TextureId, Ui};

pub(crate) fn tree_label_anchor(ui: &Ui) -> [f32; 2] {
    let [left, top] = ui.item_rect_min();
    [
        left + ui.current_font_size() + 2. * ui.clone_style().frame_padding[0],
        top,
    ]
}

pub(crate) fn prefix(ui: &Ui, view: &BrowserView, glyph: IconGlyph, [x, y]: [f32; 2]) {
    let Some(icons) = &view.icons else { return };
    let y = y + (ui.text_line_height() - 16.) * 0.5;
    let image = TextureId::new(icons.key(glyph, IconSize::Small).raw());
    ui.get_window_draw_list()
        .add_image(image, [x, y], [x + 16., y + 16.])
        .build();
}

pub(super) fn rollback_marker(ui: &Ui, text: &str) {
    let [left, top] = ui.cursor_screen_pos();
    let width = ui.content_region_avail()[0];
    let color = crate::theme::strong_border(ui);
    let draw = ui.get_window_draw_list();
    let mut x = left;
    while x < left + width {
        draw.add_line([x, top + 2.], [(x + 4.).min(left + width), top + 2.], color)
            .build();
        x += 7.;
    }
    ui.dummy([0., 4.]);
    ui.text(text);
}

pub(crate) fn strike_feature_name(ui: &Ui, name: &str, start: f32) {
    let top = ui.item_rect_min()[1];
    let bottom = ui.item_rect_max()[1];
    let end = start + ui.calc_text_size(name)[0];
    let height = (top + bottom) * 0.5;
    ui.get_window_draw_list()
        .add_line(
            [start, height],
            [end, height],
            ui.style_color(StyleColor::TextDisabled),
        )
        .thickness((ui.current_font_size() / 13.).max(1.))
        .build();
}

pub(crate) fn feature_glyph(kind: &str) -> Option<IconGlyph> {
    Some(match kind {
        "Extrusion" => IconGlyph::Extrude,
        "Sweep" => IconGlyph::Sweep,
        "Loft" => IconGlyph::Loft,
        "Revolution" | "Revolve" => IconGlyph::Revolve,
        "Coil" => IconGlyph::Coil,
        "Emboss" => IconGlyph::Emboss,
        "Rib" => IconGlyph::Rib,
        "Decal" => IconGlyph::Decal,
        "Imported Model" | "Imported Reference" => IconGlyph::Import,
        "Unwrap" => IconGlyph::Unwrap,
        "Derived Part" => IconGlyph::Derive,
        "Hole" => IconGlyph::Hole,
        "Fillet" => IconGlyph::Fillet,
        "Chamfer" => IconGlyph::Chamfer,
        "Shell" => IconGlyph::Shell,
        "Split" => IconGlyph::Split,
        "Finish" => IconGlyph::Finish,
        "Extend Surface" => IconGlyph::ExtendSurface,
        "Boundary Patch" => IconGlyph::BoundaryPatch,
        "Delete Face" => IconGlyph::DeleteFace,
        "Move Body" => IconGlyph::MoveBodies,
        "Direct Edit" => IconGlyph::DirectEdit,
        "Work Plane" => IconGlyph::WorkPlane,
        "Work Axis" => IconGlyph::WorkAxis,
        "Work Point" => IconGlyph::WorkPoint,
        "Delete Body" => IconGlyph::Delete,
        "Work Feature" | "User Coordinate System" => IconGlyph::WorkCoordinateSystem,
        _ => return None,
    })
}

pub(crate) const ORIGIN_ROWS: [&str; 7] = [
    "YZ Plane",
    "XZ Plane",
    "XY Plane",
    "X Axis",
    "Y Axis",
    "Z Axis",
    "Center Point",
];
