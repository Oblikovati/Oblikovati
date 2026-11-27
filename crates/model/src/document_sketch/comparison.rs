// SPDX-License-Identifier: GPL-2.0-only
//! Authored placed-sketch equality excludes recomputed state.
use super::PlacedSketch;

impl PartialEq for PlacedSketch {
    fn eq(&self, other: &Self) -> bool {
        // Derived verification is not authored content and is rebuilt after loading (#3722).
        self.id == other.id
            && self.name == other.name
            && self.visible == other.visible
            && self.constraints_shown == other.constraints_shown
            && self.hidden_constraint_glyphs == other.hidden_constraint_glyphs
            && self.line_style == other.line_style
            && self.placement == other.placement
            && self.orientation == other.orientation
            && self.point_projections == other.point_projections
            && self.block_instances == other.block_instances
            && self.images == other.images
            && self.geometry == other.geometry
    }
}
