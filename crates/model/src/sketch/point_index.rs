// SPDX-License-Identifier: GPL-2.0-only
//! Reuse stable point positions across one immutable geometry pass.
use super::{Sketch2D, SketchEntityId, SketchError};
use std::collections::BTreeMap;

pub(super) struct PointIndex(BTreeMap<SketchEntityId, usize>);
impl PointIndex {
    pub(super) fn new(sketch: &Sketch2D) -> Self {
        Self(
            sketch
                .points()
                .iter()
                .enumerate()
                .map(|(index, point)| (point.id, index))
                .collect(),
        )
    }
    pub(super) fn position(
        &self,
        sketch: &Sketch2D,
        id: SketchEntityId,
    ) -> Result<[f64; 2], SketchError> {
        let index = self.index(sketch, id)?;
        sketch
            .points()
            .get(index)
            .map(|point| point.position)
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "sketch point {id:?}: expected a valid indexed position"
                ))
            })
    }
    pub(super) fn index(
        &self,
        sketch: &Sketch2D,
        id: SketchEntityId,
    ) -> Result<usize, SketchError> {
        self.0
            .get(&id)
            .copied()
            .filter(|&index| {
                sketch
                    .points()
                    .get(index)
                    .is_some_and(|point| point.id == id)
            })
            .ok_or_else(|| {
                sketch.point(id).err().unwrap_or_else(|| {
                    SketchError::Definition(format!(
                        "sketch point {id:?}: expected an indexed existing point"
                    ))
                })
            })
    }
}
