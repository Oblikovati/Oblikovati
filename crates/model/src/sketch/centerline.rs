// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, SketchLine2D, SketchRecipe};
use std::collections::BTreeSet;
impl Sketch2D {
    /// Inspect a line's independent axis role; example: `sketch.is_centerline(line)?`.
    pub fn is_centerline(&self, line: SketchEntityId) -> Result<bool, SketchError> {
        self.line(line)?;
        Ok(self.centerlines.contains(&line))
    }
    /// Mark an owned line without changing explicit construction; example: `sketch.set_line_centerline(line, true)?`.
    pub fn set_line_centerline(
        &mut self,
        line: SketchEntityId,
        enabled: bool,
    ) -> Result<bool, SketchError> {
        self.line(line)?;
        Ok(if enabled {
            self.centerlines.insert(line)
        } else {
            self.centerlines.remove(&line)
        })
    }
    /// Read centerlines in authored order; example: `let candidates = sketch.centerline_ids();`.
    pub fn centerline_ids(&self) -> Vec<SketchEntityId> {
        self.entity_order
            .iter()
            .copied()
            .filter(|id| self.centerlines.contains(id))
            .collect()
    }
    /// Resolve the implicit revolve axis, refusing ambiguity and coincident endpoints.
    /// Example: `let axis = sketch.centerline_axis()?;` before requesting a revolve.
    pub fn centerline_axis(&self) -> Result<Option<SketchLine2D>, SketchError> {
        let ids = self.centerline_ids();
        match ids.as_slice() {
            [] => Ok(None),
            [id] => self.require_centerline_axis(*id).map(Some),
            _ => Err(SketchError::Definition(format!(
                "centerline axes {ids:?}: expected at most one owned axis line"
            ))),
        }
    }
    fn require_centerline_axis(&self, id: SketchEntityId) -> Result<SketchLine2D, SketchError> {
        let line = *self.line(id)?;
        let first = self.point(line.endpoints[0])?.position;
        let second = self.point(line.endpoints[1])?.position;
        if first == second {
            return Err(SketchError::Definition(format!(
                "centerline {id:?} endpoints {first:?} and {second:?}: expected distinct axis positions"
            )));
        }
        Ok(line)
    }
    pub(super) fn saved_centerlines(&self) -> Option<Vec<SketchEntityId>> {
        (!self.centerlines.is_empty()).then(|| self.centerline_ids())
    }
    pub(super) fn restore_centerlines(&mut self, recipe: &SketchRecipe) -> Result<(), SketchError> {
        match (recipe.version, &recipe.centerlines) {
            (1..=3, None) => Ok(()),
            (4..=6, Some(ids)) => self.adopt_centerlines(ids),
            shape => Err(SketchError::Definition(format!(
                "centerline recipe {shape:?}: expected no metadata in versions 1-3 or explicit line IDs in versions 4-6"
            ))),
        }
    }
    fn adopt_centerlines(&mut self, ids: &[SketchEntityId]) -> Result<(), SketchError> {
        for id in ids {
            self.line(*id)?;
        }
        let selected: BTreeSet<SketchEntityId> = ids.iter().copied().collect();
        let expected: Vec<SketchEntityId> = self
            .entity_order
            .iter()
            .copied()
            .filter(|id| selected.contains(id))
            .collect();
        if ids != expected {
            return Err(SketchError::Definition(format!(
                "centerline IDs {ids:?}: expected unique owned lines in authored order {expected:?}"
            )));
        }
        self.centerlines = selected;
        Ok(())
    }
}
