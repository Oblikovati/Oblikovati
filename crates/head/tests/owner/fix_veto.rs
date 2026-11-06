// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::placement_test_inputs::button;
use oblikovati_model::{
    history::DocumentSnapshot,
    sketch::{SketchPlacementRecipe, SketchPlane},
};

#[derive(Clone, Default)]
struct FixCommitVeto(DesktopObservations);
impl TransactionObserver for FixCommitVeto {
    fn before(
        &mut self,
        event: &TransactionEventPayload,
        snapshot: &DocumentSnapshot,
    ) -> Result<(), String> {
        if event.label == "Fix" && event.type_ == EVENT_TRANSACTION_COMMITTED {
            return Err("Fix commit: expected unlocked document".into());
        }
        self.0.before(event, snapshot)
    }
    fn after(&mut self, event: &TransactionEventPayload, snapshot: &DocumentSnapshot) {
        self.0.after(event, snapshot);
    }
    fn changed(&mut self, document: u64, generation: u64) {
        self.0.changed(document, generation);
    }
}
impl DocumentObserverFactory for FixCommitVeto {
    type Observer = Self;
    fn create(&mut self) -> Result<Self, String> {
        Ok(self.clone())
    }
}
impl HeadInvalidation for FixCommitVeto {
    fn take_changed(&self) -> bool {
        self.0.take_changed()
    }
}
#[test]
fn a_fix_commit_veto_restores_both_endpoints_and_keeps_the_tool() {
    let files = FileWorkspace::new(
        DesktopObservations::default(),
        MemoryDocumentFiles::default(),
    );
    let (signal, _awake) = mpsc::channel();
    let mut owner = DesktopOwner::new(
        files,
        OwnerThreadGuids::default(),
        FixCommitVeto::default(),
        "alpha".into(),
        RecordedWake(signal),
        DesktopDispatchOptions::default(),
    );
    owner.create_part_document().unwrap();
    let document = owner.files.workspace().active_document().unwrap();
    let mut request = super::sketch_edit::request(SketchPlane::xy());
    request.project_origin = false;
    let sketch = owner.create_sketch_for_edit(document, &request).unwrap();
    let line = SketchPlacementRecipe::line_chain(&[[1., 2.], [101., 42.]], false).unwrap();
    owner
        .commit_sketch_placement(document, sketch, &line, &[])
        .unwrap();
    let before = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    let history = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .history_entries();
    owner
        .constrain_sketch_from_gui(oblikovati_gui::GuiSketchConstraint::Fix)
        .unwrap();
    owner
        .process_window_inputs(
            &[button([51, 22], true)],
            &super::slot_input::PlanarInputWindow::default(),
        )
        .unwrap();
    let session = owner.files.workspace().session(document).unwrap();
    assert!(!session.in_transaction());
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert_eq!(session.history_entries(), history);
    assert!(owner.sketch_tool.is_some());
    assert!(!owner.check_sketch_tool_context());
    assert!(
        matches!(owner.take_command_notice(), Some(DesktopCommandNotice::AuthoringFailed { cause }) if cause.contains("unlocked"))
    );
}
