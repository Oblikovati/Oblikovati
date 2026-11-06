// SPDX-License-Identifier: GPL-2.0-only
#[cfg(test)]
use crate::selection::SketchPickTarget;
use crate::selection::{SelectionSet, SketchPick};
use crate::{DocumentSession, TransactionObserver};
use oblikovati_model::document_sketch::DocumentSketchId;
pub(crate) struct EditorSelectionBaseline {
    generation: u64,
    sketches: oblikovati_model::document_sketch::DocumentSketches,
    parameters: oblikovati_model::parameter::ParameterCollection<u64>,
}
impl EditorSelectionBaseline {
    pub(crate) fn capture(
        snapshot: &oblikovati_model::history::DocumentSnapshot,
        generation: u64,
    ) -> Self {
        Self {
            generation,
            sketches: snapshot.sketches.clone(),
            parameters: snapshot.parameters.clone(),
        }
    }
    fn is_verified_rollback(
        &self,
        snapshot: &oblikovati_model::history::DocumentSnapshot,
        generation: u64,
    ) -> bool {
        generation != self.generation
            && snapshot.sketches.same_recipe(&self.sketches)
            && snapshot.parameters.same_recipe(&self.parameters)
    }
}
impl<O: TransactionObserver> DocumentSession<O> {
    pub(crate) fn retain_editor_pick_identities(&mut self, picks: &SelectionSet<SketchPick>) {
        self.editor_selection = restamped_editor_picks(picks, self.document, self.generation());
    }
    pub(crate) fn restore_refused_editor_picks(
        &mut self,
        before: &EditorSelectionBaseline,
        sketch: Option<DocumentSketchId>,
        picks: &SelectionSet<SketchPick>,
    ) {
        let Some(sketch) = sketch else {
            return;
        };
        let original_scope = picks
            .validate_scope(self.document, before.generation, sketch)
            .is_ok();
        if !original_scope || !before.is_verified_rollback(self.snapshot(), self.generation()) {
            return;
        }
        // #3841: a verified veto rollback changes generation, not selected identities.
        self.editor_selection = restamped_editor_picks(picks, self.document, self.generation());
    }
}
fn restamped_editor_picks(
    picks: &SelectionSet<SketchPick>,
    document: u64,
    generation: u64,
) -> SelectionSet<SketchPick> {
    let mut restored = SelectionSet::default();
    for pick in picks.items() {
        restored.add(SketchPick::checked(
            document,
            generation,
            pick.sketch(),
            pick.target(),
        ));
    }
    // A narrowed filter controls future picks, not existing membership (#3841).
    restored.set_filter(picks.filter().clone());
    restored
}

#[cfg(test)]
#[path = "../tests/lifecycle/owned_selection_restamp_unit.rs"]
mod tests;
