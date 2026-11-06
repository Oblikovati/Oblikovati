// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiFrameRequest;
use oblikovati_api::types::{
    TOKEN_DIMENSION_DRIVEN, TOKEN_DIMENSION_DRIVING, TOKEN_DIMENSION_SKETCH,
};

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
    pub text: String,
    pub role: GuiDimensionLabelRole,
}
pub(crate) fn draw_dimension_labels(ui: &imgui::Ui, request: &GuiFrameRequest) {
    let draw = ui.get_background_draw_list();
    for label in request.commands.dimension_labels.iter().filter(|label| {
        label
            .pixel
            .iter()
            .zip(request.extent)
            .all(|(pixel, extent)| *pixel >= 0 && i64::from(*pixel) < i64::from(extent))
    }) {
        let size = ui.calc_text_size(&label.text);
        let position = std::array::from_fn::<_, 2, _>(|axis| {
            label.pixel[axis] as f32 / request.framebuffer_scale[axis] - size[axis] / 2.
        });
        draw.add_text(position, label_color(ui, request, label.role), &label.text);
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
