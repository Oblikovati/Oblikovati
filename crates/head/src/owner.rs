// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadInvalidation, HeadWork, HeadWorkStatus};
use oblikovati_automation::{
    DocumentObserverFactory, FileHost,
    dispatch::{DispatchWakeup, OwnerDispatch, QueuedCaller, owner_dispatch},
};
use oblikovati_model::identity::GuidSource;
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
use std::time::Duration;
#[path = "owner_commands.rs"]
mod commands;
pub use commands::{DesktopCommandNotice, DesktopHistoryAction};
#[derive(Clone, Copy, Debug)]
pub struct DesktopDispatchOptions {
    pub capacity: usize,
    pub timeout: Duration,
}
impl Default for DesktopDispatchOptions {
    fn default() -> Self {
        // The fixed Go head uses dispatch.New(64) and hostCallTimeout = 10 seconds.
        Self {
            capacity: 64,
            timeout: Duration::from_secs(10),
        }
    }
}
/// Documents and queue admission share one owner. The worker transport contains no model.
/// ```compile_fail
/// use oblikovati_head::DesktopOwner;
/// use oblikovati_automation::dispatch::PollingDispatchWakeup;
/// fn require_send<T: Send>() {}
/// require_send::<DesktopOwner<(), (), (), (), (), PollingDispatchWakeup>>();
/// ```
pub struct DesktopOwner<O, W, F, G, R, S: DispatchWakeup> {
    files: FileWorkspace<O, W, F>,
    guids: G,
    factory: R,
    version: String,
    dispatch: OwnerDispatch<S>,
    caller: QueuedCaller<S>,
    refused_close: Vec<u64>,
    part_scenes: crate::part_scene::DesktopPartScenes,
    command_notice: Option<DesktopCommandNotice>,
}
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Inject document ownership and queue options; example: `DesktopOwner::new(files, guids, factory, version, wake, options)`.
    pub fn new(
        files: FileWorkspace<O, W, F>,
        guids: G,
        factory: R,
        version: String,
        wake: S,
        options: DesktopDispatchOptions,
    ) -> Self {
        let (dispatch, caller) = owner_dispatch(options.capacity, options.timeout, wake);
        Self {
            files,
            guids,
            factory,
            version,
            dispatch,
            caller,
            refused_close: Vec::new(),
            part_scenes: Default::default(),
            command_notice: None,
        }
    }
    /// Clone only the queued worker transport; example: `Client::new(owner.caller())` on a worker.
    pub fn caller(&self) -> QueuedCaller<S> {
        self.caller.clone()
    }
    /// Inspect immutable document ownership; example: `owner.documents().workspace().documents()`.
    pub fn documents(&self) -> &FileWorkspace<O, W, F> {
        &self.files
    }
    /// Inspect documents that refused the latest native close; example: `owner.refused_close()` for a save prompt.
    pub fn refused_close(&self) -> &[u64] {
        &self.refused_close
    }
    /// Return document ownership after queue retirement, including a failed window run.
    /// Example: `let files = owner.into_documents();` lets the caller save or reopen a window explicitly.
    pub fn into_documents(mut self) -> FileWorkspace<O, W, F> {
        self.dispatch.close();
        self.files
    }
    /// Borrow the canonical caller for UI commands on this owner; example: `Client::new(owner.document_host())`.
    pub fn document_host(&mut self) -> FileHost<'_, O, W, F> {
        borrowed_host(
            &mut self.files,
            &mut self.guids,
            &self.version,
            &mut self.factory,
            &mut self.part_scenes,
        )
    }
    fn drain_calls(&mut self, budget: usize) -> Result<usize, String> {
        let host = borrowed_host(
            &mut self.files,
            &mut self.guids,
            &self.version,
            &mut self.factory,
            &mut self.part_scenes,
        );
        self.dispatch
            .drain(budget, &host)
            .map_err(|error| error.to_string())
    }
    fn close_blockers(&self) -> Result<Vec<u64>, String> {
        let workspace = self.files.workspace();
        let mut blocked = Vec::new();
        for id in workspace.documents() {
            let session = workspace.session(*id).map_err(|error| error.to_string())?;
            let dirty = workspace
                .needs_save(*id)
                .map_err(|error| error.to_string())?;
            if self.files.path(*id).is_err() || dirty || session.in_transaction() {
                blocked.push(*id);
            }
        }
        Ok(blocked)
    }
}
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> HeadWork for DesktopOwner<O, W, F, G, R, S>
{
    fn complete_save_path(
        &mut self,
        document: u64,
        selected: Option<&std::path::Path>,
    ) -> Result<bool, String> {
        Ok(self.complete_document_save_path(document, selected))
    }
    fn gui_command_state(&self) -> Result<oblikovati_gui::GuiCommandState, String> {
        let Some(document) = self.files.workspace().active_document() else {
            return Ok(Default::default());
        };
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        Ok(oblikovati_gui::GuiCommandState {
            undo_label: session.undo_label().map(str::to_owned),
            redo_label: session.redo_label().map(str::to_owned),
            transaction_open: session.in_transaction(),
        })
    }
    fn process_inputs(
        &mut self,
        inputs: &[oblikovati_viewport::ViewportEvent],
    ) -> Result<bool, String> {
        self.process_document_inputs(inputs)
    }
    fn take_command_notice(&mut self) -> Option<DesktopCommandNotice> {
        self.command_notice.take()
    }
    fn poll(&mut self, budget: usize) -> Result<HeadWorkStatus, String> {
        if budget == 0 {
            return Err("desktop drain budget 0: expected a positive bounded batch".into());
        }
        let serviced = self.drain_calls(budget)?;
        let changed = self.factory.take_changed() || serviced != 0;
        let pending = self.dispatch.pending().map_err(|error| error.to_string())? != 0;
        Ok(HeadWorkStatus {
            changed,
            pending,
            wake_after: None,
        })
    }
    fn take_scene(&mut self) -> Result<Option<crate::HeadSceneUpdate>, String> {
        self.part_scenes.refresh(&mut self.files)
    }
    fn close_allowed(&mut self) -> Result<bool, String> {
        self.refused_close = self.close_blockers()?;
        let pending = self.dispatch.pending().map_err(|error| error.to_string())?;
        Ok(self.refused_close.is_empty() && pending == 0)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.part_scenes.clear();
        self.dispatch.close();
        Ok(())
    }
}
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> crate::ScriptDocumentOwner for DesktopOwner<O, W, F, G, R, S>
{
    type Host<'a>
        = FileHost<'a, O, W, F>
    where
        Self: 'a;
    fn script_host(&mut self) -> Self::Host<'_> {
        self.document_host()
    }
}
fn borrowed_host<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &'a mut FileWorkspace<O, W, F>,
    guids: &'a mut dyn GuidSource,
    version: &'a str,
    factory: &'a mut dyn DocumentObserverFactory<Observer = O>,
    scenes: &'a mut crate::part_scene::DesktopPartScenes,
) -> FileHost<'a, O, W, F> {
    FileHost::new(files, guids, version)
        .with_observer_factory(factory)
        .with_asset_references(scenes)
}
#[cfg(test)]
#[path = "../tests/owner.rs"]
mod tests;
