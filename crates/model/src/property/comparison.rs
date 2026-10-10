// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl DocumentProperties {
    /// Compare all ordered recipe metadata, including signed-zero double values.
    /// Example: `before.same_recipe(&after)` when deciding whether to record history.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self == other
            && self.sets.iter().zip(&other.sets).all(|(left, right)| {
                left.properties
                    .iter()
                    .zip(&right.properties)
                    .all(|(left, right)| match (&left.value, &right.value) {
                        (VariantValue::Double(left), VariantValue::Double(right)) => {
                            left.to_bits() == right.to_bits()
                        }
                        _ => true,
                    })
            })
    }
}
