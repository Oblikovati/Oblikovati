// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches};
use crate::sketch::{SketchEntityId, SketchError};
/// Metadata edit evidence; entity identities and numerical geometry remain unchanged.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchConstructionReport {
    pub sketch: DocumentSketchId,
    pub changed: Vec<SketchEntityId>,
    pub construction: bool,
}
impl DocumentSketches {
    /// Set checked construction roles while retaining the unchanged numerical verification.
    /// Example: `sketches.set_construction(sketch, &[line], true)?;`.
    pub fn set_construction(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        construction: bool,
    ) -> Result<SketchConstructionReport, SketchError> {
        let index = self.index(sketch)?;
        let changed = self.sketches[index]
            .geometry
            .set_curve_construction(entities, construction)?;
        Ok(SketchConstructionReport {
            sketch,
            changed,
            construction,
        })
    }
}
