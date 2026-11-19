// SPDX-License-Identifier: GPL-2.0-only
//! Keep persisted sketch recipe versions tied to authored data.
use super::*;
impl PlacedSketch {
    pub(super) fn recipe_version(&self) -> u32 {
        if !self.hidden_constraint_glyphs.is_empty() {
            return 17;
        }
        if self.frame_edit.is_some() {
            return 15;
        }
        if matches!(self.placement, SketchPlacement::OriginPlane(_)) {
            return 13;
        }
        if matches!(self.placement, SketchPlacement::Face(_)) {
            return 12;
        }
        if !self.block_instances.is_empty() {
            return 11;
        }
        if self.line_style != SketchLineStyle::default() {
            return 7;
        }
        if self.constraints_shown {
            return 6;
        }
        if !self.visible {
            return 5;
        }
        if !self.point_projections.is_empty() {
            return 4;
        }
        if self.orientation.is_some() {
            return 3;
        }
        if matches!(self.placement, SketchPlacement::WorkPlane(_)) {
            return 2;
        }
        1
    }
}
impl DocumentSketches {
    pub(super) fn placement_recipe_version(&self) -> u32 {
        if self
            .sketches
            .iter()
            .any(|sketch| !sketch.hidden_constraint_glyphs.is_empty())
        {
            return 17;
        }
        if self
            .block_definitions
            .iter()
            .any(|block| block.color().is_some())
            || self
                .sketches
                .iter()
                .flat_map(|sketch| sketch.block_instances())
                .any(|instance| instance.color().is_some())
        {
            return 16;
        }
        if self
            .sketches
            .iter()
            .any(|sketch| sketch.frame_edit.is_some())
        {
            return 15;
        }
        if !self.spatial_sketches.is_empty() {
            return 14;
        }
        if self
            .sketches
            .iter()
            .any(|sketch| matches!(sketch.placement, SketchPlacement::OriginPlane(_)))
        {
            return 13;
        }
        if self
            .sketches
            .iter()
            .any(|sketch| matches!(sketch.placement, SketchPlacement::Face(_)))
        {
            return 12;
        }
        if self
            .sketches
            .iter()
            .any(|sketch| !sketch.block_instances.is_empty())
        {
            return 11;
        }
        if self
            .block_definitions
            .iter()
            .any(|block| !block.geometry().owned_entity_ids().is_empty())
        {
            return 9;
        }
        if !self.block_definitions.is_empty() {
            return 8;
        }
        self.sketches
            .iter()
            .map(PlacedSketch::recipe_version)
            .max()
            .unwrap_or(1)
    }
}
