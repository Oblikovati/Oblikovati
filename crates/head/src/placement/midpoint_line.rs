// SPDX-License-Identifier: GPL-2.0-only
//! Resolve the endpoint from a midpoint line's full length and direction.
use super::*;

impl SketchPlacementState {
    #[cfg(any(target_os = "linux", target_os = "windows"))]
    pub(crate) fn midpoint_awaits_center(&self) -> bool {
        self.tool == crate::SketchPlacementTool::MidpointLine && self.accepted.is_empty()
    }

    pub(super) fn bind_midpoint_vertex(&self, recipe: &mut SketchPlacementRecipe) {
        if let Some(id) = self.midpoint_vertex {
            recipe.picked_points.insert(2, id);
        }
    }

    pub(super) fn midpoint_line_end(&self, pointer: [f64; 2]) -> [f64; 2] {
        let Some(center) = self.accepted.first().copied() else {
            return pointer;
        };
        let delta = [pointer[0] - center[0], pointer[1] - center[1]];
        let [length, angle, _] = self.fields.typed_values();
        if length.is_none() && angle.is_none() {
            return pointer;
        }
        let direction = angle.unwrap_or_else(|| delta[1].atan2(delta[0]));
        let radius = length.map_or_else(|| delta[0].hypot(delta[1]), |value| value / 2.);
        [
            center[0] + radius * direction.cos(),
            center[1] + radius * direction.sin(),
        ]
    }
}
