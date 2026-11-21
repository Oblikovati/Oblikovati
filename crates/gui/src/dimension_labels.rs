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
    for (label, position) in request.commands.dimension_labels.iter().zip(positions) {
        let Some(position) = position else { continue };
        if let Some([from, to]) = label.line {
            let scale = request.layout_scale();
            let point = |pixel: [i32; 2]| [pixel[0] as f32 / scale[0], pixel[1] as f32 / scale[1]];
            ui.get_background_draw_list()
                .add_line(point(from), point(to), label_color(ui, request, label.role))
                .build();
        }
        let color = label_color(ui, request, label.role);
        if position.center != position.anchor {
            ui.get_background_draw_list()
                .add_line(position.anchor, position.center, color)
                .build();
        }
        let top_left = std::array::from_fn(|axis| position.center[axis] - position.size[axis] / 2.);
        crate::widget_abi::background_literal_text(ui, top_left, color, &label.text);
    }
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
    [color.r, color.g, color.b, color.a]
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
