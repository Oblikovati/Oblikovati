// SPDX-License-Identifier: GPL-2.0-only
//! Authored axis, angular driver, and source identities for circular feature repetition.
use super::{
    FeatureId, FeatureReadFootprint, FeatureReadKey,
    extrusion_pattern::{ExtrusionPatternOccurrence, validate_sources},
    move_body::ResolvedBodyMotion,
    work_geometry::DatumAngle,
};
use crate::parameter::ParameterCollection;
use std::collections::BTreeSet;

/// How the authored angle determines the interval between occurrences.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum CircularPatternSpacing {
    Sweep,
    Between,
    Fitted,
}

/// Authored source IDs and axis placement in model units.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct CircularPatternRecipe {
    pub sources: Vec<FeatureId>,
    pub count: u32,
    pub angle: DatumAngle,
    pub origin: [f64; 3],
    pub direction: [f64; 3],
    pub mid_plane: bool,
    pub spacing: CircularPatternSpacing,
    pub suppressed: BTreeSet<u64>,
}

/// Validated recipe for a circular extrusion feature pattern.
#[derive(Clone, Debug, PartialEq)]
pub struct CircularPatternDefinition(CircularPatternRecipe);

impl CircularPatternDefinition {
    /// Validate a stored axis and occurrence recipe. Example: `CircularPatternDefinition::new(recipe)?`.
    pub fn new(recipe: CircularPatternRecipe) -> Result<Self, String> {
        validate_sources(&recipe.sources)?;
        recipe
            .angle
            .validate_source()
            .map_err(|cause| cause.to_string())?;
        if recipe.count == 0 || recipe.suppressed.contains(&0) {
            return Err(
                "circular pattern: expected a positive count and suppression outside the seed"
                    .into(),
            );
        }
        if recipe.direction == [0.; 3]
            || recipe
                .origin
                .into_iter()
                .chain(recipe.direction)
                .any(|value| !value.is_finite())
        {
            return Err("circular pattern: expected a finite axis with nonzero direction".into());
        }
        Ok(Self(recipe))
    }

    /// Capture authored placement and sources. Example: `definition.recipe()`.
    pub fn recipe(&self) -> &CircularPatternRecipe {
        &self.0
    }

    /// Compare authored IDs and coordinate bits. Example: `before.same_recipe(after)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self == other
            && self.0.angle.same_recipe(other.0.angle)
            && self.0.origin.map(f64::to_bits) == other.0.origin.map(f64::to_bits)
            && self.0.direction.map(f64::to_bits) == other.0.direction.map(f64::to_bits)
    }

    /// Attribute reads to the source features and current angle parameter. Example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let features = self.0.sources.iter().copied().map(FeatureReadKey::Feature);
        let angle = self.0.angle.parameter_id().map(FeatureReadKey::Parameter);
        FeatureReadFootprint::new(features.chain(angle), false)
    }

    /// Resolve rotations in element order, skipping the seed. Example: `definition.occurrences(parameters)?`.
    pub fn occurrences(
        &self,
        parameters: &ParameterCollection<u64>,
    ) -> Result<Vec<ExtrusionPatternOccurrence>, String> {
        let recipe = &self.0;
        let angle = recipe
            .angle
            .resolve(parameters)
            .map_err(|cause| cause.to_string())?;
        if recipe.count == 1 {
            return Ok(Vec::new());
        }
        let increment = angle
            / match recipe.spacing {
                CircularPatternSpacing::Sweep => f64::from(recipe.count),
                CircularPatternSpacing::Between => 1.,
                CircularPatternSpacing::Fitted => f64::from(recipe.count - 1),
            };
        let seed = if recipe.mid_plane {
            (recipe.count - 1) / 2
        } else {
            0
        };
        let mut occurrences = Vec::new();
        for index in 0..recipe.count {
            if index == seed
                || recipe
                    .suppressed
                    .contains(&(u64::from(index) + u64::from(index < seed)))
            {
                continue;
            }
            let offset = i64::from(index) - i64::from(seed);
            let radians = increment * offset as f64;
            if !radians.is_finite() {
                return Err(format!(
                    "circular pattern element {index}: expected a finite rotation"
                ));
            }
            occurrences.push(ExtrusionPatternOccurrence {
                offset: [offset, 0],
                motion: ResolvedBodyMotion::Rotation {
                    origin: recipe.origin,
                    direction: recipe.direction,
                    radians,
                },
            });
        }
        Ok(occurrences)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fitted_mid_plane_places_the_extra_element_on_the_positive_side() {
        let definition = CircularPatternDefinition::new(CircularPatternRecipe {
            sources: vec![FeatureId(2)],
            count: 4,
            angle: DatumAngle::constant(1.5).unwrap(),
            origin: [1., 2., 3.],
            direction: [0., 0., 1.],
            mid_plane: true,
            spacing: CircularPatternSpacing::Fitted,
            suppressed: BTreeSet::from([2]),
        })
        .unwrap();
        let parameters = ParameterCollection::new(crate::units::UnitsOfMeasure::default());
        let occurrences = definition.occurrences(&parameters).unwrap();
        assert_eq!(occurrences.len(), 2);
        assert_eq!(occurrences[0].offset, [-1, 0]);
        assert_eq!(occurrences[1].offset, [2, 0]);
        assert_eq!(
            occurrences[0].motion,
            ResolvedBodyMotion::Rotation {
                origin: [1., 2., 3.],
                direction: [0., 0., 1.],
                radians: -0.5,
            },
        );
        assert_eq!(
            occurrences[1].motion,
            ResolvedBodyMotion::Rotation {
                origin: [1., 2., 3.],
                direction: [0., 0., 1.],
                radians: 1.,
            },
        );
    }
}
