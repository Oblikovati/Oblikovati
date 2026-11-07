// SPDX-License-Identifier: GPL-2.0-only
use super::{DrawingError, spelling, validation::require_label};
use oblikovati_api::types::{
    BaseViewOrientation, DRAWING_VIEW_BASE, DrawingViewStyle, DrawingViewType,
    HIDDEN_LINE_VIEW_STYLE,
};
use serde::{Deserialize, Serialize};
/// An authored base-view definition. Its source must still be resolved and projected before display.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct DrawingBaseViewDefinition {
    pub name: String,
    #[serde(rename = "type", default, with = "spelling::view_kind")]
    pub kind: DrawingViewType,
    #[serde(default, with = "spelling::base_orientation")]
    pub orientation: BaseViewOrientation,
    #[serde(default = "unit_scale")]
    pub scale: f64,
    #[serde(default, with = "spelling::view_style")]
    pub style: DrawingViewStyle,
    #[serde(default, rename = "centerXmm")]
    pub center_x_mm: f64,
    #[serde(default, rename = "centerYmm")]
    pub center_y_mm: f64,
}
impl DrawingBaseViewDefinition {
    /// Define a base view without inventing its projection; example: `DrawingBaseViewDefinition::new("Front", BASE_VIEW_FRONT, 1., [100.,100.])?`.
    pub fn new(
        name: &str,
        orientation: BaseViewOrientation,
        scale: f64,
        center: [f64; 2],
    ) -> Result<Self, DrawingError> {
        let view = Self {
            name: name.into(),
            kind: DRAWING_VIEW_BASE,
            orientation,
            scale,
            style: HIDDEN_LINE_VIEW_STYLE,
            center_x_mm: center[0],
            center_y_mm: center[1],
        };
        view.validate()?;
        Ok(view)
    }
    pub(super) fn validate(&self) -> Result<(), DrawingError> {
        require_label("view name", &self.name)?;
        self.require_view_options()?;
        self.require_view_measurements()
    }
    fn require_view_options(&self) -> Result<(), DrawingError> {
        if self.kind != DRAWING_VIEW_BASE
            || spelling::base_orientation::name(&self.orientation).is_none()
            || spelling::view_style::name(&self.style).is_none()
        {
            return Err(DrawingError(format!(
                "view {:?}, kind {:?}, orientation {:?}, style {:?}: expected a base view with canonical orientation and style",
                self.name, self.kind, self.orientation, self.style
            )));
        }
        Ok(())
    }
    fn require_view_measurements(&self) -> Result<(), DrawingError> {
        if !self.scale.is_finite()
            || self.scale <= 0.
            || !self.center_x_mm.is_finite()
            || !self.center_y_mm.is_finite()
        {
            return Err(DrawingError(format!(
                "view {:?} scale {}, center [{}, {}]: expected positive finite scale and finite sheet coordinates",
                self.name, self.scale, self.center_x_mm, self.center_y_mm
            )));
        }
        Ok(())
    }
}
fn unit_scale() -> f64 {
    1.
}
