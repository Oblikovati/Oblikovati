// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::DocumentSnapshot;
#[derive(Clone)]
pub(super) struct ToolCommitVeto {
    label: &'static str,
    observations: DesktopObservations,
}
impl TransactionObserver for ToolCommitVeto {
    fn before(
        &mut self,
        event: &TransactionEventPayload,
        snapshot: &DocumentSnapshot,
    ) -> Result<(), String> {
        if event.label == self.label && event.type_ == EVENT_TRANSACTION_COMMITTED {
            return Err(format!("{} commit: expected unlocked document", self.label));
        }
        self.observations.before(event, snapshot)
    }
    fn after(&mut self, event: &TransactionEventPayload, snapshot: &DocumentSnapshot) {
        self.observations.after(event, snapshot);
    }
    fn changed(&mut self, document: u64, generation: u64) {
        self.observations.changed(document, generation);
    }
}
impl DocumentObserverFactory for ToolCommitVeto {
    type Observer = Self;
    fn create(&mut self) -> Result<Self, String> {
        Ok(self.clone())
    }
}
impl HeadInvalidation for ToolCommitVeto {
    fn take_changed(&self) -> bool {
        self.observations.take_changed()
    }
}
pub(super) type VetoOwner = DesktopOwner<
    ToolCommitVeto,
    DesktopObservations,
    MemoryDocumentFiles,
    OwnerThreadGuids,
    ToolCommitVeto,
    RecordedWake,
>;
pub(super) fn veto_owner(label: &'static str) -> (VetoOwner, Receiver<()>) {
    let files = FileWorkspace::new(
        DesktopObservations::default(),
        MemoryDocumentFiles::default(),
    );
    let (signal, awake) = mpsc::channel();
    let observer = ToolCommitVeto {
        label,
        observations: Default::default(),
    };
    (
        DesktopOwner::new(
            files,
            OwnerThreadGuids::default(),
            observer,
            "alpha".into(),
            RecordedWake(signal),
            DesktopDispatchOptions::default(),
        ),
        awake,
    )
}
