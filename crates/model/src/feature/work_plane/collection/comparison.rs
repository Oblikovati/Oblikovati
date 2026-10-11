// SPDX-License-Identifier: GPL-2.0-only

use super::{AuthoredFeature, FeatureSequence, WorkPlaneDefinition};

impl WorkPlaneDefinition {
    /// Compare exact authored definition bits; example: `old.same_recipe(candidate)` before editing.
    pub fn same_recipe(self, other: Self) -> bool {
        match (self, other) {
            (Self::Axis(first), Self::Axis(second)) => first.same_recipe(second),
            (Self::Point(first), Self::Point(second)) => same_coordinates(first, second),
            (Self::AxisThroughPoints(first), Self::AxisThroughPoints(second)) => first == second,
            (Self::PointAtReference(first), Self::PointAtReference(second)) => first == second,
            (Self::Fixed(first), Self::Fixed(second)) => first.same_recipe(second),
            (first @ Self::Offset { .. }, second @ Self::Offset { .. }) => {
                same_offset(first, second)
            }
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
        && first.construction_datum == second.construction_datum
        && first.datum_visible == second.datum_visible
        && first.plane_normal_flipped == second.plane_normal_flipped
        && first.plane_display.same_recipe(&second.plane_display)
        && first.definition.same_recipe(second.definition)
        && first.definition_reads == second.definition_reads
        && first.owned_scalars == second.owned_scalars
        && match (first.condition, second.condition) {
            (Some(left), Some(right)) => left.same_recipe(right),
            (None, None) => true,
            _ => false,
        }
}

fn same_coordinates(
    first: [crate::feature::work_geometry::DatumLength; 3],
    second: [crate::feature::work_geometry::DatumLength; 3],
) -> bool {
    first
        .into_iter()
        .zip(second)
        .all(|(left, right)| left.same_recipe(right))
}

fn same_offset(first: WorkPlaneDefinition, second: WorkPlaneDefinition) -> bool {
    match (first, second) {
        (
            WorkPlaneDefinition::Offset {
                base: first,
                distance: left,
            },
            WorkPlaneDefinition::Offset {
                base: second,
                distance: right,
            },
        ) => first == second && left.same_recipe(right),
        _ => false,
    }
}
