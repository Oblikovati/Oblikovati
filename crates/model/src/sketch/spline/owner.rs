// SPDX-License-Identifier: GPL-2.0-only
use super::{
    SketchOffsetSpline2D, SketchSpline2D, SketchSplineHandle2D, SketchSplineKind,
    require_spline_definition,
};
use crate::sketch::{Sketch2D, SketchEntityId, SketchError};
impl Sketch2D {
    /// Adopt a parent after resolving every shared point; example: `sketch.add_spline(parent)?`.
    pub fn add_spline(&mut self, spline: SketchSpline2D) -> Result<(), SketchError> {
        self.require_new_entity(spline.id())?;
        require_spline_definition(spline.id(), spline.points(), spline.kind())?;
        for point in spline.points() {
            self.point(*point)?;
        }
        self.entity_order.push(spline.id());
        self.splines.push(spline);
        Ok(())
    }
    /// Adopt an offset only against an owned parent spline; example: `sketch.add_offset_spline(offset)?`.
    pub fn add_offset_spline(&mut self, offset: SketchOffsetSpline2D) -> Result<(), SketchError> {
        self.require_new_entity(offset.id())?;
        SketchOffsetSpline2D::new(offset.id(), offset.parent(), offset.distance())?;
        self.spline(offset.parent())?;
        self.entity_order.push(offset.id());
        self.offset_splines.push(offset);
        Ok(())
    }
    /// Adopt one handle after its parent and end point have been resolved; example: `sketch.add_spline_handle(handle)?`.
    pub fn add_spline_handle(&mut self, handle: SketchSplineHandle2D) -> Result<(), SketchError> {
        let parent = self.spline(handle.parent())?;
        if !matches!(parent.kind(), SketchSplineKind::Fit { .. })
            || handle.fit_index() >= parent.points().len()
        {
            return Err(SketchError::Definition(format!(
                "spline handle {handle:?}: expected an in-range fit-parent point"
            )));
        }
        self.point(handle.end())?;
        if !self.defining_points.contains(&handle.end()) {
            return Err(SketchError::Definition(format!(
                "spline handle {handle:?}: expected a hidden owned end point"
            )));
        }
        if self.spline_handles.iter().any(|saved| {
            saved.parent() == handle.parent() && saved.fit_index() == handle.fit_index()
        }) {
            return Err(SketchError::Definition(format!(
                "spline handle {handle:?}: expected one handle per fit point"
            )));
        }
        self.spline_handles.push(handle);
        Ok(())
    }
    /// Resolve one active handle by parent and fit-point index; example: `sketch.spline_handle(parent, 1)?`.
    pub fn spline_handle(
        &self,
        parent: SketchEntityId,
        fit_index: usize,
    ) -> Result<SketchSplineHandle2D, SketchError> {
        self
            .spline_handles
            .iter()
            .copied()
            .find(|handle| handle.parent() == parent && handle.fit_index() == fit_index)
            .ok_or_else(|| SketchError::Definition(format!("spline handle parent {parent:?} fit point {fit_index}: expected an active handle")))
    }
    /// Resolve the exact parent identity; example: `sketch.spline(parent_id)?`.
    pub fn spline(&self, id: SketchEntityId) -> Result<&SketchSpline2D, SketchError> {
        self.splines
            .iter()
            .find(|spline| spline.id() == id)
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "sketch spline {id:?}: expected an owned parent spline"
                ))
            })
    }
    /// Resolve an offset without geometric proximity; example: `sketch.offset_spline(id)?`.
    pub fn offset_spline(&self, id: SketchEntityId) -> Result<&SketchOffsetSpline2D, SketchError> {
        self.offset_splines
            .iter()
            .find(|offset| offset.id() == id)
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "sketch offset spline {id:?}: expected an owned offset spline"
                ))
            })
    }
    /// Inspect parents in insertion order; example: `sketch.splines()`.
    pub fn splines(&self) -> &[SketchSpline2D] {
        &self.splines
    }
    /// Inspect derived offsets in insertion order; example: `sketch.offset_splines()`.
    pub fn offset_splines(&self) -> &[SketchOffsetSpline2D] {
        &self.offset_splines
    }
    /// Inspect active handles in insertion order; example: `sketch.spline_handles()`.
    pub fn spline_handles(&self) -> &[SketchSplineHandle2D] {
        &self.spline_handles
    }
}
