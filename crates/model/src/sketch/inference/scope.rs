// SPDX-License-Identifier: GPL-2.0-only
//! Validate and resolve inference candidates from owned sketch geometry (#4107).
use super::{Sketch2D, SketchEntityId, SketchError};
use oblikovati_api::types::{
    INFERENCE_SCOPE_ALL_GEOMETRY, INFERENCE_SCOPE_CURRENT_COMMAND,
    INFERENCE_SCOPE_SELECTED_GEOMETRY, SketchInferenceScope,
};
use std::collections::BTreeSet;

/// One validated inference mode and its explicit sketch entity identities.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchInferenceCandidates {
    mode: SketchInferenceScope,
    selected: Vec<SketchEntityId>,
}
impl SketchInferenceCandidates {
    /// Validate the scope shape; example: `SketchInferenceCandidates::new(mode, &ids)?`.
    pub fn new(
        mode: SketchInferenceScope,
        entities: &[SketchEntityId],
    ) -> Result<Self, SketchError> {
        let mut selected = entities.to_vec();
        selected.sort();
        let valid_mode = mode == INFERENCE_SCOPE_ALL_GEOMETRY
            || mode == INFERENCE_SCOPE_CURRENT_COMMAND
            || mode == INFERENCE_SCOPE_SELECTED_GEOMETRY;
        let valid_list = selected.iter().all(|id| id.0 != 0)
            && selected.windows(2).all(|pair| pair[0] != pair[1]);
        let selected_mode = mode == INFERENCE_SCOPE_SELECTED_GEOMETRY;
        if !valid_mode || !valid_list || selected_mode == selected.is_empty() {
            return Err(SketchError::Definition(format!(
                "inference scope {mode:?} with entities {entities:?}: expected a known mode, unique nonzero IDs, and a nonempty list only for selected geometry"
            )));
        }
        Ok(Self { mode, selected })
    }
    /// Return the mode; example: `scope.mode()`.
    pub fn mode(&self) -> &SketchInferenceScope {
        &self.mode
    }
    /// Return selected IDs in stable order; example: `scope.selected()`.
    pub fn selected(&self) -> &[SketchEntityId] {
        &self.selected
    }
    /// Check selected identities against current geometry; example: `scope.validate_geometry(&sketch)?`.
    pub fn validate_geometry(&self, sketch: &Sketch2D) -> Result<(), SketchError> {
        self.resolve(sketch, &[]).map(|_| ())
    }
    pub(super) fn resolve(
        &self,
        sketch: &Sketch2D,
        current: &[SketchEntityId],
    ) -> Result<ResolvedCandidates, SketchError> {
        if self.mode == INFERENCE_SCOPE_ALL_GEOMETRY {
            return Ok(ResolvedCandidates::all());
        }
        let roots = if self.mode == INFERENCE_SCOPE_CURRENT_COMMAND {
            current
        } else {
            &self.selected
        };
        expand_owned_candidates(sketch, roots).map(|ids| ResolvedCandidates(Some(ids)))
    }
}

impl oblikovati_api::contract::SketchInferenceScope for SketchInferenceCandidates {
    fn inference_scope(&self) -> SketchInferenceScope {
        self.mode.clone()
    }
    fn inference_scope_entity_count(&self) -> i64 {
        i64::try_from(self.selected.len()).expect("sketch candidate count fits in i64")
    }
}

fn expand_owned_candidates(
    sketch: &Sketch2D,
    roots: &[SketchEntityId],
) -> Result<BTreeSet<SketchEntityId>, SketchError> {
    let catalogue = crate::sketch::entity_geometry::catalogue(sketch);
    let mut pending = roots.to_vec();
    let mut resolved = BTreeSet::new();
    while let Some(id) = pending.pop() {
        if !resolved.insert(id) {
            continue;
        }
        let entity = catalogue.get(&id).ok_or_else(|| {
            SketchError::Definition(format!(
                "inference candidate {id:?}: expected current owned sketch geometry"
            ))
        })?;
        pending.extend(entity.ownership_dependencies());
    }
    Ok(resolved)
}

pub(super) struct ResolvedCandidates(Option<BTreeSet<SketchEntityId>>);
impl ResolvedCandidates {
    pub(super) fn all() -> Self {
        Self(None)
    }
    pub(super) fn contains(&self, id: SketchEntityId) -> bool {
        self.0.as_ref().is_none_or(|ids| ids.contains(&id))
    }
}
