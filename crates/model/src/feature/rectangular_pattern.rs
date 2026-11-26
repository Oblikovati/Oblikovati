// SPDX-License-Identifier: GPL-2.0-only
//! Authored feature grid whose occurrence identities survive replay.
use super::{FeatureId, FeatureReadFootprint, FeatureReadKey};
use std::collections::BTreeSet;

/// Authored grid and source feature identities stored in the part.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct RectangularPatternRecipe {
    pub sources: Vec<FeatureId>,
    pub counts: [u32; 2],
    pub steps: [[f64; 3]; 2],
    pub mid_plane: [bool; 2],
    pub fitted: bool,
    pub suppressed: BTreeSet<u64>,
}

/// Validated grid recipe.
#[derive(Clone, Debug, PartialEq)]
pub struct RectangularPatternDefinition(RectangularPatternRecipe);

/// Model-space placement addressed by signed offsets from the authored seed cell.
#[derive(Clone, Copy, Debug)]
pub struct RectangularPatternOccurrence {
    pub offset: [i64; 2],
    pub translation: [f64; 3],
}

/// A resolved extrusion tool and its operation, borrowed from the authored source feature.
pub struct PatternPrismSource<'a> {
    pub source: FeatureId,
    pub prism: super::sketch_prism::ResolvedSketchPrism<'a>,
    pub operation: super::sketch_prism::PrismBoolean,
}

impl RectangularPatternDefinition {
    /// Validate a saved grid. Example: `RectangularPatternDefinition::new(recipe)?`.
    pub fn new(recipe: RectangularPatternRecipe) -> Result<Self, String> {
        validate_recipe(&recipe)?;
        Ok(Self(recipe))
    }
    /// Capture authored inputs. Example: `definition.recipe()`.
    pub fn recipe(&self) -> &RectangularPatternRecipe {
        &self.0
    }
    /// Compare authored identity including signed coordinate zero. Example: `left.same_recipe(right)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self == other
            && self.0.steps.map(|step| step.map(f64::to_bits))
                == other.0.steps.map(|step| step.map(f64::to_bits))
    }
    /// Check dependencies on the source features. Example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        FeatureReadFootprint::new(
            self.0.sources.iter().copied().map(FeatureReadKey::Feature),
            false,
        )
    }
    /// Compute the current translations in authored grid order. Example: `definition.occurrences()?`.
    pub fn occurrences(&self) -> Result<Vec<RectangularPatternOccurrence>, String> {
        let [columns, rows] = self.0.counts.map(i64::from);
        let [sx, sy] = std::array::from_fn(|axis| {
            if self.0.mid_plane[axis] {
                i64::from(self.0.counts[axis] - 1) / 2
            } else {
                0
            }
        });
        let seed = sy as u64 * columns as u64 + sx as u64;
        let count = (columns as u64) * (rows as u64);
        let mut occurrences = Vec::new();
        occurrences
            .try_reserve_exact(usize::try_from(count - 1).map_err(|error| error.to_string())?)
            .map_err(|error| error.to_string())?;
        for y in 0..rows {
            for x in 0..columns {
                let index = y as u64 * columns as u64 + x as u64;
                if index == seed
                    || self
                        .0
                        .suppressed
                        .contains(&(index + u64::from(index < seed)))
                {
                    continue;
                }
                occurrences.push(self.occurrence([x - sx, y - sy])?);
            }
        }
        Ok(occurrences)
    }

    fn occurrence(&self, offset: [i64; 2]) -> Result<RectangularPatternOccurrence, String> {
        let scale = std::array::from_fn::<_, 2, _>(|axis| {
            if self.0.fitted && self.0.counts[axis] > 1 {
                offset[axis] as f64 / f64::from(self.0.counts[axis] - 1)
            } else {
                offset[axis] as f64
            }
        });
        let translation = std::array::from_fn(|axis| {
            self.0.steps[0][axis] * scale[0] + self.0.steps[1][axis] * scale[1]
        });
        if translation.iter().any(|value| !value.is_finite()) {
            return Err(format!(
                "rectangular pattern cell {offset:?}: expected a finite translation"
            ));
        }
        Ok(RectangularPatternOccurrence {
            offset,
            translation,
        })
    }
}

fn validate_recipe(recipe: &RectangularPatternRecipe) -> Result<(), String> {
    let unique: BTreeSet<_> = recipe.sources.iter().copied().collect();
    if unique.is_empty() || unique.len() != recipe.sources.len() || unique.contains(&FeatureId(0)) {
        return Err(format!(
            "rectangular pattern sources {:?}: expected distinct feature identities",
            recipe.sources
        ));
    }
    if recipe.counts.contains(&0) || recipe.suppressed.contains(&0) {
        return Err(
            "rectangular pattern: expected positive counts and suppression outside the seed".into(),
        );
    }
    for (count, step) in recipe.counts.into_iter().zip(recipe.steps) {
        if step.iter().any(|value| !value.is_finite()) || (count > 1 && step == [0.; 3]) {
            return Err(format!(
                "rectangular pattern count {count}, step {step:?}: expected a finite step and spacing for multiple occurrences"
            ));
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rectangular_pattern_uses_row_order_and_seed_relative_cells() {
        let definition = RectangularPatternDefinition::new(RectangularPatternRecipe {
            sources: vec![FeatureId(7)],
            counts: [4, 2],
            steps: [[6., 0., 0.], [0., 2., 0.]],
            mid_plane: [true, false],
            fitted: true,
            suppressed: BTreeSet::from([2, 6, 11]),
        })
        .unwrap();
        let positions: Vec<_> = definition
            .occurrences()
            .unwrap()
            .into_iter()
            .map(|item| (item.offset, item.translation))
            .collect();
        assert_eq!(
            positions,
            vec![
                ([-1, 0], [-2., 0., 0.]),
                ([2, 0], [4., 0., 0.]),
                ([-1, 1], [-2., 2., 0.]),
                ([0, 1], [0., 2., 0.]),
                ([2, 1], [4., 2., 0.]),
            ]
        );
    }
}
