// SPDX-License-Identifier: GPL-2.0-only
//! Authored plane and source identities for a feature mirror.
use super::{
    FeatureId, FeatureReadFootprint, FeatureReadKey,
    extrusion_pattern::{ExtrusionPatternOccurrence, validate_sources},
    move_body::ResolvedBodyMotion,
};

/// Authored feature mirror inputs in model units.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct MirrorRecipe {
    pub sources: Vec<FeatureId>,
    pub origin: [f64; 3],
    pub normal: [f64; 3],
}

/// A validated plane and one or more authored feature identities.
#[derive(Clone, Debug, PartialEq)]
pub struct MirrorDefinition(MirrorRecipe);

impl MirrorDefinition {
    /// Admit a finite plane with nonzero normal. Example: `MirrorDefinition::new(recipe)?`.
    pub fn new(recipe: MirrorRecipe) -> Result<Self, String> {
        validate_sources(&recipe.sources)?;
        if recipe.normal == [0.; 3]
            || recipe
                .origin
                .into_iter()
                .chain(recipe.normal)
                .any(|value| !value.is_finite())
        {
            return Err(format!(
                "mirror plane origin {:?}, normal {:?}: expected a finite origin and nonzero finite normal",
                recipe.origin, recipe.normal
            ));
        }
        Ok(Self(recipe))
    }

    /// Inspect authored sources and plane. Example: `definition.recipe()`.
    pub fn recipe(&self) -> &MirrorRecipe {
        &self.0
    }

    /// Compare exact authored inputs. Example: `before.same_recipe(after)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self.0.sources == other.0.sources
            && self.0.origin.map(f64::to_bits) == other.0.origin.map(f64::to_bits)
            && self.0.normal.map(f64::to_bits) == other.0.normal.map(f64::to_bits)
    }

    /// Attribute reads to the source features. Example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        FeatureReadFootprint::new(
            self.0.sources.iter().copied().map(FeatureReadKey::Feature),
            false,
        )
    }

    /// Place one reflected copy of each source tool. Example: `definition.occurrence()`.
    pub fn occurrence(&self) -> ExtrusionPatternOccurrence {
        ExtrusionPatternOccurrence {
            offset: [1, 0],
            motion: ResolvedBodyMotion::Reflection {
                origin: self.0.origin,
                normal: self.0.normal,
            },
        }
    }
}
