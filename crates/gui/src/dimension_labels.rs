// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiFrameRequest;
use oblikovati_api::types::{
    TOKEN_DIMENSION_DRIVEN, TOKEN_DIMENSION_DRIVING, TOKEN_DIMENSION_SKETCH,
};
mod layout;
pub(crate) use layout::{LabelPosition, dimension_label_layout};

/// Semantic annotation colours; example: `GuiDimensionLabelRole::Driven` for a reference value.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiDimensionLabelRole {
    Driving,
    Driven,
    Preview,
}
/// Owned physical screen coordinates; example: submit one label after native camera projection.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDimensionLabel {
    pub pixel: [i32; 2],
    /// The projected dimension line, when this measurement has a linear line.
    pub line: Option<[[i32; 2]; 2]>,
    /// The measured geometry, projected independently of the offset line.
    pub witnesses: Option<[[i32; 2]; 2]>,
    pub unit: Option<String>,
    pub text: String,
    pub role: GuiDimensionLabelRole,
    /// A driving dimension opens its value editor on a double-click.
    pub edit: Option<crate::GuiDimensionEdit>,
}
pub(crate) fn draw_dimension_labels(
    ui: &imgui::Ui,
    request: &GuiFrameRequest,
    positions: &[Option<LabelPosition>],
) {
    let _numeric = crate::ui_scale::numeric_font(ui);
    let scale = request.layout_scale();
    let point = |pixel: [i32; 2]| [pixel[0] as f32 / scale[0], pixel[1] as f32 / scale[1]];
    let draw = ui.get_background_draw_list();
    for (label, position) in request.commands.dimension_labels.iter().zip(positions) {
        let Some(position) = position else { continue };
        let color = leader_color(ui, request);
        if let Some([from, to]) = label.line {
            draw.add_line(point(from), point(to), color).build();
            if let Some([a, b]) = label.witnesses {
                draw.add_line(point(a), point(from), color).build();
                draw.add_line(point(b), point(to), color).build();
            }
        }
        if position.center != position.anchor {
            let [dx, dy] =
                std::array::from_fn(|axis| position.anchor[axis] - position.center[axis]);
            let advance = (dx.abs() / (position.size[0] / 2. + 2.))
                .max(dy.abs() / (position.size[1] / 2. + 2.));
            if advance > 1. {
                let end = [
                    position.center[0] + dx / advance,
                    position.center[1] + dy / advance,
                ];
                draw.add_line(position.anchor, end, color).build();
            }
        }
    }
    // Paint labels after the measurement lines so neither a crossing witness nor
    // a nearby dimension can draw through the caption.
    for (label, position) in request.commands.dimension_labels.iter().zip(positions) {
        let Some(position) = position else { continue };
        let color = label_color(ui, request, label.role);
        let top_left = std::array::from_fn(|axis| position.center[axis] - position.size[axis] / 2.);
        let lower = std::array::from_fn(|axis| top_left[axis] + position.size[axis] + 2.);
        draw.add_rect(
            [top_left[0] - 2., top_left[1] - 2.],
            lower,
            annotation_backdrop(ui, request, position.center[1]),
        )
        .rounding(2.)
        .filled(true)
        .build();
        crate::widget_abi::background_literal_text(ui, top_left, color, &label.text);
        if let Some(unit) = label.unit.as_deref() {
            let prefix = if unit == "°" { "" } else { " " };
            let offset = ui.calc_text_size(format!("{}{prefix}", label.text))[0];
            crate::widget_abi::background_literal_text(
                ui,
                [top_left[0] + offset, top_left[1]],
                [color[0], color[1], color[2], color[3] * 0.8],
                unit,
            );
        }
    }
}
impl GuiDimensionLabel {
    fn caption(&self) -> std::borrow::Cow<'_, str> {
        match self.unit.as_deref() {
            Some("°") => format!("{}°", self.text).into(),
            Some(unit) => format!("{} {unit}", self.text).into(),
            None => (&self.text).into(),
        }
    }
}

fn annotation_backdrop(ui: &imgui::Ui, request: &GuiFrameRequest, y: f32) -> [f32; 4] {
    use oblikovati_viewport::ViewportBackground;
    let Some(scheme) = &request.commands.color_scheme else {
        return ui.style_color(imgui::StyleColor::WindowBg);
    };
    let rgb = match scheme.background() {
        ViewportBackground::Solid(rgb) => rgb.map(|channel| f32::from(channel) / 255.),
        ViewportBackground::VerticalGradient { top, bottom } => {
            let fraction = (y / ui.io().display_size[1].max(1.)).clamp(0., 1.);
            std::array::from_fn(|axis| {
                let start = f32::from(top[axis]);
                (start + (f32::from(bottom[axis]) - start) * fraction) / 255.
            })
        }
    };
    [rgb[0], rgb[1], rgb[2], 1.]
}
fn leader_color(ui: &imgui::Ui, request: &GuiFrameRequest) -> [f32; 4] {
    let [r, g, b, _] = annotation_backdrop(ui, request, ui.io().display_size[1] * 0.5);
    crate::theme::hex_rgb(if r.max(g).max(b) > 0.5 {
        0x636B7E
    } else {
        0x7C8496
    })
}
fn label_color(ui: &imgui::Ui, request: &GuiFrameRequest, role: GuiDimensionLabelRole) -> [f32; 4] {
    let token = match role {
        GuiDimensionLabelRole::Driving => TOKEN_DIMENSION_DRIVING,
        GuiDimensionLabelRole::Driven => TOKEN_DIMENSION_DRIVEN,
        GuiDimensionLabelRole::Preview => TOKEN_DIMENSION_SKETCH,
    };
    let Some(theme) = request.commands.theme.as_ref() else {
        return ui.style_color(imgui::StyleColor::Text);
    };
    let color = &theme.palette()[&token];
    let preferred = [color.r, color.g, color.b, color.a];
    request
        .commands
        .color_scheme
        .as_ref()
        .map_or(preferred, |scheme| scheme.contrasting_ink(preferred))
}
pub(crate) fn validate_dimension_labels(labels: &[GuiDimensionLabel]) -> Result<(), String> {
    for label in labels {
        if label.text.is_empty() || label.text.contains('\0') {
            return Err(format!(
                "dimension label {label:?}: expected nonempty text without NUL"
            ));
        }
    }
    Ok(())
}
