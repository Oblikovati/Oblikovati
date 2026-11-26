// SPDX-License-Identifier: GPL-2.0-only
use super::{DrawingError, spelling, validation::require_label};
use oblikovati_api::types::{
    BaseViewOrientation, DRAWING_VIEW_BASE, DrawingViewStyle, DrawingViewType,
    HIDDEN_LINE_VIEW_STYLE,
};
use serde::{Deserialize, Serialize};

/// Changes to a view caption. Omitted fields retain their authored values.
#[derive(Default)]
pub struct DrawingViewLabelEdit<'a> {
    pub text: Option<&'a str>,
    pub show_label: Option<bool>,
    pub show_name: Option<bool>,
    pub show_scale: Option<bool>,
    pub position_mm: Option<[f64; 2]>,
}

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
    #[serde(default, skip_serializing_if = "is_zero")]
    pub rotation_deg: f64,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    label_text: String,
    #[serde(default, skip_serializing_if = "is_false")]
    hide_label: bool,
    #[serde(default, skip_serializing_if = "is_false")]
    hide_name: bool,
    #[serde(default, skip_serializing_if = "is_false")]
    hide_scale: bool,
    #[serde(default, rename = "labelXmm", skip_serializing_if = "is_zero")]
    label_x_mm: f64,
    #[serde(default, rename = "labelYmm", skip_serializing_if = "is_zero")]
    label_y_mm: f64,
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
            rotation_deg: 0.,
            label_text: String::new(),
            hide_label: false,
            hide_name: false,
            hide_scale: false,
            label_x_mm: 0.,
            label_y_mm: 0.,
        };
        view.validate()?;
        Ok(view)
    }
    pub(super) fn validate(&self) -> Result<(), DrawingError> {
        require_label("view name", &self.name)?;
        self.require_view_options()?;
        self.require_view_measurements()?;
        self.validate_label(&self.label_text, self.label_position_mm())
    }
    /// Compose the visible caption from its override or enabled default components.
    pub fn label(&self) -> String {
        if self.hide_label {
            return String::new();
        }
        if !self.label_text.is_empty() {
            return self.label_text.clone();
        }
        let scale = || scale_note(self.scale);
        match (self.show_name(), self.show_scale()) {
            (true, true) => format!("{}  SCALE {}", self.name, scale()),
            (true, false) => self.name.clone(),
            (false, true) => format!("SCALE {}", scale()),
            (false, false) => String::new(),
        }
    }
    pub fn show_label(&self) -> bool {
        !self.hide_label
    }
    pub fn show_name(&self) -> bool {
        !self.hide_name
    }
    pub fn show_scale(&self) -> bool {
        !self.hide_scale
    }
    /// Return the authored caption position. The zero pair selects automatic placement.
    pub fn label_position_mm(&self) -> [f64; 2] {
        [self.label_x_mm, self.label_y_mm]
    }
    pub(super) fn set_label(&mut self, edit: DrawingViewLabelEdit<'_>) -> Result<(), DrawingError> {
        let position = edit.position_mm.unwrap_or_else(|| self.label_position_mm());
        self.validate_label(edit.text.unwrap_or(&self.label_text), position)?;
        if let Some(text) = edit.text {
            self.label_text = text.into();
        }
        self.hide_label = !edit.show_label.unwrap_or_else(|| self.show_label());
        self.hide_name = !edit.show_name.unwrap_or_else(|| self.show_name());
        self.hide_scale = !edit.show_scale.unwrap_or_else(|| self.show_scale());
        [self.label_x_mm, self.label_y_mm] = position;
        Ok(())
    }
    fn validate_label(&self, text: &str, [x, y]: [f64; 2]) -> Result<(), DrawingError> {
        if text.contains('\0') || !x.is_finite() || !y.is_finite() {
            return Err(DrawingError(format!(
                "view {:?}: expected caption text without NUL and finite caption coordinates",
                self.name
            )));
        }
        Ok(())
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
        if !self.rotation_deg.is_finite() {
            return Err(DrawingError(format!(
                "view {:?} rotation {}: expected finite degrees",
                self.name, self.rotation_deg
            )));
        }
        Ok(())
    }
}
fn unit_scale() -> f64 {
    1.
}
fn is_zero(value: &f64) -> bool {
    *value == 0.
}
fn is_false(value: &bool) -> bool {
    !*value
}
fn scale_note(scale: f64) -> String {
    if scale >= 1. {
        format!("{scale}:1")
    } else {
        format!("1:{}", 1. / scale)
    }
}
