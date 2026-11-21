// SPDX-License-Identifier: GPL-2.0-only
//! Enter a newly created Part's startup sketch, or discard the failed creation.
use super::*;
use oblikovati_model::document_sketch::SketchPlacement;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn enter_new_part_startup(
        &mut self,
        document: u64,
        previous: Option<u64>,
    ) -> Result<(), String> {
        let Some(plane) = self
            .part_store
            .as_ref()
            .and_then(|store| store.part_options().startup_sketch_plane)
        else {
            return Ok(());
        };
        if let Err(cause) = self.enter_created_startup_sketch(document, plane) {
            return Err(format!(
                "new part {document} startup edit: expected an editable sketch: {cause}; {}",
                self.rollback_new_part(document, previous)
            ));
        }
        Ok(())
    }
    fn enter_created_startup_sketch(
        &mut self,
        document: u64,
        plane: oblikovati_model::feature::work_plane::WorkPlaneOrigin,
    ) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        let sketches = session.snapshot().sketches.sketches();
        let [sketch] = sketches else {
            return Err(format!(
                "document {document} has {} sketches: expected exactly one startup sketch",
                sketches.len()
            ));
        };
        if sketch.placement() != SketchPlacement::OriginPlane(plane) {
            return Err(format!(
                "startup sketch {:?} placement {:?}: expected origin plane {plane:?}",
                sketch.id(),
                sketch.placement()
            ));
        }
        let id = sketch.id();
        self.begin_sketch_edit(document, id)
    }
    fn rollback_new_part(&mut self, document: u64, previous: Option<u64>) -> String {
        if let Err(error) = self.files.close_discarding_changes(document) {
            return format!("rollback of document {document} failed: {error}");
        }
        if let Some(previous) = previous
            && let Err(error) = self.files.activate(previous)
        {
            return format!("rollback activation of document {previous} failed: {error}");
        }
        "new document discarded and prior focus restored".into()
    }
}
