// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchEntityEditError, SketchEntityId, SketchError, SketchLine2D, SketchLineEnd,
};
use crate::identity::RecipeIdError;
mod projection;
/// The original first half, fresh second half, and their shared new defining point.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchLineSplitReport {
    pub parts: [SketchEntityId; 2],
    pub junction: SketchEntityId,
}
impl Sketch2D {
    /// Split at the orthogonal pick projection without moving original endpoints or constraints.
    /// Example: `sketch.split_line(line, [2.0, 1.0], &mut reserve)?;`.
    pub fn split_line(
        &mut self,
        line: SketchEntityId,
        pick: [f64; 2],
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<SketchLineSplitReport, SketchEntityEditError> {
        let original = *self.line(line).map_err(SketchEntityEditError::Sketch)?;
        let position = self
            .prepare_line_split(original, pick)
            .map_err(SketchEntityEditError::Sketch)?;
        let report = SketchLineSplitReport::reserve(self, line, reserve)?;
        let mut candidate = self.clone();
        candidate
            .adopt_line_split(self, original, position, &report)
            .map_err(SketchEntityEditError::Sketch)?;
        *self = candidate;
        Ok(report)
    }
    fn prepare_line_split(
        &self,
        line: SketchLine2D,
        pick: [f64; 2],
    ) -> Result<[f64; 2], SketchError> {
        let first = self.point(line.endpoints[0])?.position;
        let last = self.point(line.endpoints[1])?.position;
        projection::split_position(line.id, [first, last], pick)
    }
    fn adopt_line_split(
        &mut self,
        source: &Self,
        original: SketchLine2D,
        position: [f64; 2],
        report: &SketchLineSplitReport,
    ) -> Result<(), SketchError> {
        self.add_defining_point(report.junction, position)?;
        self.add_line(report.parts[1], [report.junction, original.endpoints[1]])?;
        self.copy_entity_roles(source, original.id, report.parts[1])?;
        self.rewire_line_endpoint(original.id, SketchLineEnd::Last, report.junction)
    }
}

impl SketchLineSplitReport {
    fn reserve(
        owner: &Sketch2D,
        line: SketchEntityId,
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<Self, SketchEntityEditError> {
        let ids = owner.reserve_edit_identities(2, reserve)?;
        Ok(Self {
            parts: [line, ids[1]],
            junction: ids[0],
        })
    }
}
