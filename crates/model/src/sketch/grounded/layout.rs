// SPDX-License-Identifier: GPL-2.0-only
use super::SketchError;
use std::collections::BTreeSet;

#[derive(Clone, Copy)]
pub(super) enum GroundedSlot {
    Constant(f64),
    Active(usize),
}
pub(super) struct GroundedLayout {
    slots: Vec<GroundedSlot>,
    active_count: usize,
}
impl GroundedLayout {
    pub fn new(values: &[f64], columns: &BTreeSet<usize>) -> Self {
        let mut active_count = 0;
        let slots = values
            .iter()
            .enumerate()
            .map(|(column, value)| {
                if columns.contains(&column) {
                    return GroundedSlot::Constant(*value);
                }
                let slot = GroundedSlot::Active(active_count);
                active_count += 1;
                slot
            })
            .collect();
        Self {
            slots,
            active_count,
        }
    }
    pub fn active_values(&self, original: &[f64]) -> Vec<f64> {
        self.slots
            .iter()
            .zip(original)
            .filter_map(|(slot, value)| matches!(slot, GroundedSlot::Active(_)).then_some(*value))
            .collect()
    }
    pub fn local_slots(&self, columns: &[usize]) -> Vec<GroundedSlot> {
        columns.iter().map(|column| self.slots[*column]).collect()
    }
    pub fn restore(&self, values: &[f64]) -> Result<Vec<f64>, SketchError> {
        if values.len() != self.active_count {
            return Err(SketchError::Definition(format!(
                "grounded candidate has {} variables: expected {} active values",
                values.len(),
                self.active_count
            )));
        }
        Ok(self
            .slots
            .iter()
            .map(|slot| match slot {
                GroundedSlot::Constant(value) => *value,
                GroundedSlot::Active(index) => values[*index],
            })
            .collect())
    }
}
