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
        self.require_active_count(values.len(), "grounded candidate")?;
        Ok(self
            .slots
            .iter()
            .map(|slot| match slot {
                GroundedSlot::Constant(value) => *value,
                GroundedSlot::Active(index) => values[*index],
            })
            .collect())
    }
    pub fn restore_freedoms(&self, values: &[bool]) -> Result<Vec<bool>, SketchError> {
        self.require_active_count(values.len(), "grounded coordinate freedom")?;
        Ok(self
            .slots
            .iter()
            .map(|slot| match slot {
                GroundedSlot::Constant(_) => false,
                GroundedSlot::Active(index) => values[*index],
            })
            .collect())
    }
    fn require_active_count(&self, actual: usize, context: &str) -> Result<(), SketchError> {
        if actual != self.active_count {
            return Err(SketchError::Definition(format!(
                "{context} has {actual} variables: expected {} active values",
                self.active_count
            )));
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn freedom_restore_maps_constants_and_checks_reduced_shape() {
        let layout = GroundedLayout::new(&[1.0, 2.0, 3.0], &BTreeSet::from([1]));
        assert_eq!(
            layout.restore_freedoms(&[true, false]).unwrap(),
            [true, false, false]
        );
        let cause = layout.restore_freedoms(&[]).unwrap_err().to_string();
        assert!(cause.contains("grounded coordinate freedom has 0"));
        assert!(cause.contains("expected 2 active values"));
        assert!(layout.restore(&[]).is_err());
    }
}
