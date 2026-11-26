// SPDX-License-Identifier: GPL-2.0-only
use super::{
    FIXED_VERSION, FeatureId, SketchSweepDefinition, SketchSweepError, SketchSweepOrientation,
    SketchSweepRecipe, follow_path_version,
};
use crate::feature::{fillet::reference_identity, sketch_prism::PrismBoolean};

const JOIN_VERSION: u32 = 5;
const BOOLEAN_VERSION: u32 = 6;

/// Persisted operation and body key for a Sweep Boolean.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SweepBooleanRecipe {
    operation: PrismBoolean,
    key: Vec<u8>,
}

impl SketchSweepDefinition {
    /// Join this Sweep to a named body. Example: `sweep.with_join_target(body_key)?`.
    pub fn with_join_target(self, key: Vec<u8>) -> Result<Self, SketchSweepError> {
        Self::from_recipe(SketchSweepRecipe {
            version: JOIN_VERSION,
            join_target: Some(key),
            boolean_target: None,
            ..self.0
        })
    }

    /// Read the named Join target and its producing feature. Example: `sweep.join_target()`.
    pub fn join_target(&self) -> Option<(&[u8], FeatureId)> {
        self.0.join_target.as_deref().zip(self.1)
    }

    /// Bind a solid operation to an exact target key. Example: `sweep.with_boolean_target(PrismBoolean::Cut, key)?`.
    pub fn with_boolean_target(
        self,
        operation: PrismBoolean,
        key: Vec<u8>,
    ) -> Result<Self, SketchSweepError> {
        if operation == PrismBoolean::Join {
            return self.with_join_target(key);
        }
        Self::from_recipe(SketchSweepRecipe {
            version: BOOLEAN_VERSION,
            join_target: None,
            boolean_target: Some(SweepBooleanRecipe { operation, key }),
            ..self.0
        })
    }

    /// The replacement operation and exact target. Example: `sweep.boolean_target()` during Part replay.
    pub fn boolean_target(&self) -> Option<(PrismBoolean, &[u8], FeatureId)> {
        let source = self.1?;
        match &self.0.boolean_target {
            Some(target) => Some((target.operation, &target.key, source)),
            None => self
                .join_target()
                .map(|(key, source)| (PrismBoolean::Join, key, source)),
        }
    }

    /// Clear the named target. Example: `sweep.without_boolean_target()`.
    pub fn without_boolean_target(self) -> Self {
        let version = match self.0.orientation {
            SketchSweepOrientation::Fixed => FIXED_VERSION,
            SketchSweepOrientation::FollowPath => follow_path_version(self.0.taper, self.0.twist),
        };
        Self(
            SketchSweepRecipe {
                version,
                join_target: None,
                boolean_target: None,
                ..self.0
            },
            None,
        )
    }
}

pub(super) fn target_version(recipe: &SketchSweepRecipe, base: u32) -> u32 {
    if recipe.boolean_target.is_some() {
        BOOLEAN_VERSION
    } else if recipe.join_target.is_some() {
        JOIN_VERSION
    } else {
        base
    }
}

pub(super) fn target_source(
    recipe: &SketchSweepRecipe,
) -> Result<Option<FeatureId>, SketchSweepError> {
    if let Some(target) = &recipe.boolean_target
        && (recipe.join_target.is_some() || target.operation == PrismBoolean::Join)
    {
        return Err(SketchSweepError::Definition(
            "sweep: expected one canonical Boolean target".into(),
        ));
    }
    recipe
        .join_target
        .as_deref()
        .or_else(|| {
            recipe
                .boolean_target
                .as_ref()
                .map(|target| target.key.as_slice())
        })
        .map(|key| {
            reference_identity(key, 4, "sweep Boolean target")
                .map(|(_, source)| source)
                .map_err(SketchSweepError::Definition)
        })
        .transpose()
}
