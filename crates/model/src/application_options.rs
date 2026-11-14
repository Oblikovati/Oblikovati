// SPDX-License-Identifier: GPL-2.0-only
/// Application preferences are independent of document history and solver policy.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(default, rename_all = "camelCase", deny_unknown_fields)]
pub struct SketchApplicationOptions {
    pub grid_spacing_cm: f64,
    pub grid_visible: bool,
    pub grid_major_every: i64,
    pub snap_to_points: bool,
    pub snap_to_grid: bool,
    pub heads_up_display: bool,
    pub relax_mode: bool,
    pub pointer_input: bool,
    pub pointer_input_cartesian: bool,
    pub dimension_input: bool,
    pub dimension_input_cartesian: bool,
    pub create_dimensions_on_value_input: bool,
    pub auto_project_origin: bool,
    pub suppress_format_overrides: bool,
}
impl Default for SketchApplicationOptions {
    fn default() -> Self {
        // #3764: a legacy file without autoProjectOrigin keeps Go's default-on policy.
        Self {
            grid_spacing_cm: 1.0,
            grid_visible: true,
            grid_major_every: 5,
            snap_to_points: true,
            snap_to_grid: true,
            heads_up_display: true,
            relax_mode: false,
            pointer_input: true,
            pointer_input_cartesian: true,
            dimension_input: true,
            dimension_input_cartesian: false,
            create_dimensions_on_value_input: true,
            auto_project_origin: true,
            suppress_format_overrides: false,
        }
    }
}
impl SketchApplicationOptions {
    /// Refuse a grid that cannot form finite model-space spacing; example: `options.validate()?`.
    pub fn validate(&self) -> Result<(), String> {
        if !self.grid_spacing_cm.is_finite() || self.grid_spacing_cm <= 0.0 {
            return Err(format!(
                "sketch grid spacing {} cm: expected a positive finite value",
                self.grid_spacing_cm
            ));
        }
        Ok(())
    }
}
/// Inject the one application preference owner across UI and automation calls.
pub trait SketchPreferenceAccess {
    /// Read the current validated options; example: `preferences.sketch_options().auto_project_origin`.
    fn sketch_options(&self) -> &SketchApplicationOptions;
    /// Persist before publishing a new group; failure must preserve current options.
    /// Example: `preferences.set_sketch_options(updated)?`.
    fn set_sketch_options(&mut self, options: SketchApplicationOptions) -> Result<(), String>;
}
