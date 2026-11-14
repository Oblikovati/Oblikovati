// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchCopyOperation, SketchError, require_position};
use std::collections::BTreeSet;

/// Grid counts include the seed. Vectors are signed spacing or fitted spans.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchRectangularPatternLayout {
    pub counts: [u32; 2],
    pub vectors: [[f64; 2]; 2],
    pub fitted: bool,
    pub suppressed: BTreeSet<[u32; 2]>,
}
/// A non-seed grid occurrence and its transform from the original selection.
#[derive(Clone, Copy, Debug)]
pub struct SketchRectangularOccurrence {
    pub index: [u32; 2],
    pub operation: SketchCopyOperation,
}
impl SketchRectangularPatternLayout {
    /// Calculate visible copies in Go's column-first order, excluding the original seed.
    /// Example: `let occurrences = layout.occurrences()?;`.
    pub fn occurrences(&self) -> Result<Vec<SketchRectangularOccurrence>, SketchError> {
        self.validate()?;
        let count = (self.counts[0] as usize)
            .checked_mul(self.counts[1] as usize)
            .and_then(|total| total.checked_sub(1 + self.suppressed.len()))
            .ok_or_else(|| self.invalid("representable occurrence count"))?;
        let mut result = Vec::new();
        result
            .try_reserve_exact(count)
            .map_err(|_| self.invalid("allocatable occurrence count"))?;
        for column in 0..self.counts[0] {
            for row in 0..self.counts[1] {
                let index = [column, row];
                if index != [0, 0] && !self.suppressed.contains(&index) {
                    result.push(self.occurrence(index)?);
                }
            }
        }
        Ok(result)
    }
    fn validate(&self) -> Result<(), SketchError> {
        if self.counts.contains(&0) {
            return Err(self.invalid("positive counts in both directions"));
        }
        for vector in self.vectors {
            require_position(vector)?;
        }
        if self.suppressed.iter().any(|index| {
            *index == [0, 0] || index[0] >= self.counts[0] || index[1] >= self.counts[1]
        }) {
            return Err(self.invalid("suppressed indices within the grid, excluding the seed"));
        }
        Ok(())
    }
    fn occurrence(&self, index: [u32; 2]) -> Result<SketchRectangularOccurrence, SketchError> {
        let factors = std::array::from_fn::<_, 2, _>(|axis| {
            if self.fitted && self.counts[axis] > 1 {
                f64::from(index[axis]) / f64::from(self.counts[axis] - 1)
            } else {
                f64::from(index[axis])
            }
        });
        let offset = std::array::from_fn(|axis| {
            self.vectors[0][axis] * factors[0] + self.vectors[1][axis] * factors[1]
        });
        require_position(offset)?;
        Ok(SketchRectangularOccurrence {
            index,
            operation: SketchCopyOperation::Translate(offset),
        })
    }
    fn invalid(&self, expected: &str) -> SketchError {
        SketchError::Definition(format!("rectangular pattern {self:?}: expected {expected}"))
    }
}
