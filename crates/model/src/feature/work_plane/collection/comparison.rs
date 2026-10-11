// SPDX-License-Identifier: GPL-2.0-only

use super::{AuthoredFeature, FeatureSequence, WorkPlaneDefinition};
use crate::feature::work_plane::WorkPlaneDistance;

impl WorkPlaneDefinition {
    /// Compare exact authored definition bits; example: `old.same_recipe(candidate)` before editing.
    pub fn same_recipe(self, other: Self) -> bool {
        match (self, other) {
            (Self::Fixed(first), Self::Fixed(second)) => first.same_recipe(second),
            (
                Self::Offset {
                    base: first,
                    distance: left,
                },
                Self::Offset {
                    base: second,
                    distance: right,
                },
            ) => first == second && same_distance(left, right),
            _ => false,
        }
    }
}

impl FeatureSequence<WorkPlaneDefinition> {
    /// Compare complete authored state without derived frames or health; example: `baseline.same_work_plane_recipe(&candidate)` for dirty checks.
    pub fn same_work_plane_recipe(&self, other: &Self) -> bool {
        self.end_of_part == other.end_of_part
            && self.features.len() == other.features.len()
            && self
                .features
                .iter()
                .zip(&other.features)
                .all(|(first, second)| same_feature(first, second))
    }
}

fn same_feature(
    first: &AuthoredFeature<WorkPlaneDefinition>,
    second: &AuthoredFeature<WorkPlaneDefinition>,
) -> bool {
    first.id == second.id
        && first.name == second.name
        && first.suppressed == second.suppressed
        && first.definition.same_recipe(second.definition)
        && first.definition_reads == second.definition_reads
        && match (first.condition, second.condition) {
            (Some(left), Some(right)) => left.same_recipe(right),
            (None, None) => true,
            _ => false,
        }
}

fn same_distance(first: WorkPlaneDistance, second: WorkPlaneDistance) -> bool {
    match (first, second) {
        (WorkPlaneDistance::Constant(left), WorkPlaneDistance::Constant(right)) => {
            left.to_bits() == right.to_bits()
        }
        (WorkPlaneDistance::Parameter(left), WorkPlaneDistance::Parameter(right)) => left == right,
        _ => false,
    }
}
