// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl SketchPlacementRecipe {
    /// Author one visible point with one identity; example: `SketchPlacementRecipe::point([2.,3.])?`.
    pub fn point(position: [f64; 2]) -> Result<Self, SketchError> {
        require_position(position)?;
        Ok(Self {
            points: vec![position],
            members: vec![SketchPlacementMember {
                geometry: SketchPlacementGeometry::Point(0),
                construction: false,
            }],
            relations: vec![],
            fields: vec![],
            circular_checks: vec![],
        })
    }
}
