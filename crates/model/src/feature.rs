// SPDX-License-Identifier: GPL-2.0-only
//! Ordered authored feature state, separate from kernel execution and caches.
//! Go provenance: `model/feature/{engine,reorder,engine_graph}.go`, aa8297d1.

mod edits;
mod error;
mod footprint;
mod replay;
mod validation;

use crate::dependency::DependencyGraph;
pub use error::FeatureSequenceError;
pub use footprint::{FeatureReadFootprint, FeatureReadKey};
pub use replay::FeatureReplayRange;

/// Permanent document-local identity; reservation belongs outside undo snapshots.
#[derive(Clone, Copy, Debug, Eq, Ord, PartialEq, PartialOrd)]
pub struct FeatureId(pub u64);

/// Owned definition and its authored edit boundaries, without geometry caches.
#[derive(Clone, Debug, PartialEq)]
pub struct AuthoredFeature<Definition> {
    id: FeatureId,
    name: String,
    definition: Definition,
    suppressed: bool,
    reads: FeatureReadFootprint,
}

/// Recipe order shared by future concrete feature definitions and the evaluator.
#[derive(Clone, Debug)]
pub struct FeatureSequence<Definition> {
    features: Vec<AuthoredFeature<Definition>>,
    dependencies: DependencyGraph<FeatureId>,
    end_of_part: Option<usize>,
}

impl<Definition> Default for FeatureSequence<Definition> {
    fn default() -> Self {
        Self {
            features: Vec::new(),
            dependencies: DependencyGraph::default(),
            end_of_part: None,
        }
    }
}

impl<Definition> AuthoredFeature<Definition> {
    /// Construct an owned authored entry; example: `AuthoredFeature::new(FeatureId(1), "Plane", plane, reads)`.
    pub fn new(
        id: FeatureId,
        name: &str,
        definition: Definition,
        reads: FeatureReadFootprint,
    ) -> Self {
        Self {
            id,
            name: name.into(),
            definition,
            suppressed: false,
            reads,
        }
    }

    /// Inspect stable identity; example: `feature.id()` survives reorder.
    pub fn id(&self) -> FeatureId {
        self.id
    }

    /// Inspect the editable label; example: `feature.name()` for the browser.
    pub fn name(&self) -> &str {
        &self.name
    }

    /// Read the owned typed recipe; example: `feature.definition()` during replay.
    pub fn definition(&self) -> &Definition {
        &self.definition
    }

    /// Inspect explicit suppression; example: `feature.suppressed()` selects pass-through.
    pub fn suppressed(&self) -> bool {
        self.suppressed
    }

    /// Inspect attributed reads; example: `feature.reads().keys()` for dependency queries.
    pub fn reads(&self) -> &FeatureReadFootprint {
        &self.reads
    }
}

impl<Definition> FeatureSequence<Definition> {
    /// Read program order without permitting edits outside validation; example: `sequence.features()[0].id()`.
    pub fn features(&self) -> &[AuthoredFeature<Definition>] {
        &self.features
    }

    /// Resolve exactly one stable identity; example: `sequence.get(FeatureId(1))?`.
    pub fn get(&self, id: FeatureId) -> Result<&AuthoredFeature<Definition>, FeatureSequenceError> {
        Ok(&self.features[self.position(id)?])
    }

    /// Read the exclusive cutoff, or None for all features; example: `sequence.end_of_part()`.
    pub fn end_of_part(&self) -> Option<usize> {
        self.end_of_part
    }

    /// Read the evaluated prefix length; example: `&sequence.features()[..sequence.effective_end()]`.
    pub fn effective_end(&self) -> usize {
        self.end_of_part
            .unwrap_or(self.features.len())
            .min(self.features.len())
    }
}
