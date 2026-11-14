// SPDX-License-Identifier: GPL-2.0-only
use super::{
    AuthoredFeature, FeatureId, FeatureSequence, WorkGeometryDefinition, WorkGeometryError,
};
use crate::{sketch::SketchPlane, units::UnitsOfMeasure};
mod corners;

/// Display and associativity source; rectangle coordinates use API centimetres, not working units.
#[derive(Clone, Debug, Default, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(default, deny_unknown_fields)]
pub struct WorkPlaneDisplaySettings {
    auto_resize: bool,
    grounded: bool,
    corners: Option<[[f64; 3]; 2]>,
}
impl WorkPlaneDisplaySettings {
    /// Apply optional flags and size together; example: `settings.edited(Some(true), None, Some(corners))?` disables auto-resize.
    pub fn edited(
        &self,
        auto_resize: Option<bool>,
        grounded: Option<bool>,
        corners: Option<[[f64; 3]; 2]>,
    ) -> Result<Self, WorkGeometryError> {
        let next = Self {
            auto_resize: if corners.is_some() {
                false
            } else {
                auto_resize.unwrap_or(self.auto_resize)
            },
            grounded: grounded.unwrap_or(self.grounded),
            corners: corners.or(self.corners),
        };
        next.validate()?;
        Ok(next)
    }
    /// Read the saved auto-resize request; example: `settings.auto_resize()`.
    pub fn auto_resize(&self) -> bool {
        self.auto_resize
    }
    /// Read the saved associativity flag; example: `settings.grounded()`.
    pub fn grounded(&self) -> bool {
        self.grounded
    }
    /// Detect metadata that older recipes can omit; example: `settings.is_default()`.
    pub fn is_default(&self) -> bool {
        !self.auto_resize && !self.grounded && self.corners.is_none()
    }
    pub(crate) fn validate(&self) -> Result<(), WorkGeometryError> {
        if let Some(corners) = self.corners {
            corners::validate(corners)?;
        }
        Ok(())
    }
    /// Compare exact authored bits; example: `settings.same_recipe(&candidate)` distinguishes signed zero.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self.auto_resize == other.auto_resize
            && self.grounded == other.grounded
            && corners::same_recipe(self.corners, other.corners)
    }
    /// Read explicit corners or derive them from the current frame; example: `settings.current_corners(Some(frame), &units)?`.
    pub fn current_corners(
        &self,
        frame: Option<SketchPlane>,
        units: &UnitsOfMeasure,
    ) -> Result<Option<[[f64; 3]; 2]>, WorkGeometryError> {
        self.validate()?;
        if let Some(corners) = self.corners {
            return Ok(Some(corners));
        }
        frame
            .map(|frame| corners::derived(frame, units))
            .transpose()
    }
}
impl AuthoredFeature<WorkGeometryDefinition> {
    /// Read authored settings without an evaluated frame; example: `plane.plane_display_settings()`.
    pub fn plane_display_settings(&self) -> &WorkPlaneDisplaySettings {
        &self.plane_display
    }
    pub(crate) fn validate_plane_display(&self) -> Result<(), WorkGeometryError> {
        if !self.plane_display.is_default() {
            self.definition.require_plane()?;
        }
        self.plane_display.validate()
    }
}
impl FeatureSequence<WorkGeometryDefinition> {
    /// Adopt validated settings on a placed plane; example: `datums.set_plane_display(id, settings)?`.
    pub fn set_plane_display(
        &mut self,
        id: FeatureId,
        settings: WorkPlaneDisplaySettings,
    ) -> Result<(), WorkGeometryError> {
        let position = self
            .position(id)
            .map_err(|_| WorkGeometryError::Unavailable {
                feature: id,
                expected: "placed plane",
            })?;
        self.features[position].definition.require_plane()?;
        settings.validate()?;
        self.features[position].plane_display = settings;
        Ok(())
    }
}
