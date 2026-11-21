// SPDX-License-Identifier: GPL-2.0-only
//! Which sketch points are center points: a saved role that changes only how a point reads.
use super::{Sketch2D, SketchEntityId, SketchError, SketchRecipe};

impl Sketch2D {
    /// Read a point's role; example: `sketch.is_center_point(point)`.
    pub fn is_center_point(&self, point: SketchEntityId) -> bool {
        self.center_points.contains(&point)
    }
    /// A circle's or arc's center, which reads as a center point whatever its saved role.
    pub fn is_implicit_circular_center(&self, point: SketchEntityId) -> bool {
        self.circles.iter().any(|circle| circle.center == point)
            || self.arcs.iter().any(|arc| arc.center == point)
    }
    /// Change one point's role without touching its position; example: from its context menu.
    pub fn set_center_point(
        &mut self,
        point: SketchEntityId,
        center: bool,
    ) -> Result<(), SketchError> {
        self.point(point)?;
        if center {
            self.center_points.insert(point);
        } else {
            self.center_points.remove(&point);
        }
        Ok(())
    }
    pub(super) fn saved_center_points(&self) -> Option<Vec<SketchEntityId>> {
        (!self.center_points.is_empty()).then(|| self.center_points.iter().copied().collect())
    }
    pub(super) fn has_center_points(&self) -> bool {
        !self.center_points.is_empty()
    }
    pub(super) fn restore_center_points(
        &mut self,
        recipe: &SketchRecipe,
    ) -> Result<(), SketchError> {
        let Some(points) = &recipe.center_points else {
            return Ok(());
        };
        if recipe.version < 12 {
            return Err(SketchError::Definition(format!(
                "center points {points:?} in sketch recipe version {}: expected version 12",
                recipe.version
            )));
        }
        points
            .iter()
            .try_for_each(|point| self.set_center_point(*point, true))
    }
    pub(super) fn retain_center_points(
        &mut self,
        retired: &std::collections::BTreeSet<SketchEntityId>,
    ) {
        self.center_points.retain(|point| !retired.contains(point));
    }
}
