// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Compare owned recipe values, excluding evaluation health and change queues.
    /// Example: `before.same_recipe(&after)` before recording an undo step.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self.order == other.order
            && self.units == other.units
            && self.settings == other.settings
            && self.groups == other.groups
            && same_tables(&self.derived_tables, &other.derived_tables)
            && self.records.iter().all(|(id, record)| {
                other.records.get(id).is_some_and(|right| {
                    record.formula.is_some() == right.formula.is_some()
                        && comparison_state(&record.state) == comparison_state(&right.state)
                })
            })
    }

    /// Compare recipe values and evaluation health, excluding transient change queues.
    /// Example: `before.same_evaluated_state(&after)` before publishing refreshed values.
    pub fn same_evaluated_state(&self, other: &Self) -> bool {
        self.same_recipe(other)
            && self.records.iter().all(|(id, record)| {
                other
                    .records
                    .get(id)
                    .is_some_and(|right| record.state.health == right.state.health)
            })
            && self
                .derived_tables
                .iter()
                .zip(&other.derived_tables)
                .all(|(left, right)| left.health == right.health)
    }
}

fn comparison_state<Id: Clone>(source: &ParameterState<Id>) -> (ParameterState<Id>, [u64; 3]) {
    let mut state = source.clone();
    let mut bits = [
        state.tolerance.upper.to_bits(),
        state.tolerance.lower.to_bits(),
        0,
    ];
    state.tolerance.upper = 0.0;
    state.tolerance.lower = 0.0;
    if let Some(ParameterValue::Numeric(quantity)) = &mut state.last_good_value {
        bits[2] = quantity.value.to_bits();
        quantity.value = 0.0;
    }
    // Health is rebuilt by evaluation; signed zero is stored recipe information (#3614).
    state.health = ParameterHealth::Healthy;
    (state, bits)
}

fn same_tables<Id: Clone + PartialEq>(
    left: &[DerivedTable<Id>],
    right: &[DerivedTable<Id>],
) -> bool {
    left.len() == right.len()
        && left.iter().zip(right).all(|(left, right)| {
            let mut left = left.clone();
            let mut right = right.clone();
            left.health = DerivedTableHealth::Unresolved;
            right.health = DerivedTableHealth::Unresolved;
            left == right
        })
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::quantity::{Dimension, Quantity};

    #[test]
    fn evaluation_health_is_not_an_authored_recipe_change() {
        let mut before = ParameterCollection::new(UnitsOfMeasure::default());
        before
            .insert_expression(1, "value", USER_PARAM, "2")
            .unwrap();
        before
            .restore_derived_table(1, "part.obk", &["absent".into()], false)
            .unwrap();
        let mut after = before.clone();
        after.records.get_mut(&1).unwrap().state.health = ParameterHealth::Failed(
            ParameterFailure::NonFinite(Quantity::new(f64::NAN, Dimension::Unitless)),
        );
        after.derived_tables[0].health = DerivedTableHealth::Unavailable;
        assert!(before.same_recipe(&after));
        assert!(after.same_recipe(&before));
        assert!(!before.same_evaluated_state(&after));
        assert!(before.same_evaluated_state(&before.clone()));
    }
}
