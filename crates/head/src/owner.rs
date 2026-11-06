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
#[path = "owner_documents.rs"]
mod documents_gui;
#[path = "owner_sketch_tools.rs"]
mod sketch_tools_gui;
use sketch_tools_gui::gui_placement_tool;
#[path = "owner_browser_selection.rs"]
mod browser_selection;
#[path = "owner_browser_visibility.rs"]
mod browser_visibility;
#[path = "owner_dimension_annotations.rs"]
mod dimension_annotations;
#[path = "owner_dimension_query.rs"]
mod dimension_query;
#[path = "owner_geometric_batch.rs"]
mod geometric_batch;
#[path = "owner_line_commit.rs"]
mod line_commit;
#[path = "owner_line_scope.rs"]
mod line_scope;
#[path = "owner_browser.rs"]
mod model_browser;
#[path = "owner_parameters.rs"]
mod parameters_gui;
#[path = "owner_sketch_delete.rs"]
mod sketch_delete;
#[path = "owner_sketch_dimension.rs"]
mod sketch_dimension;
#[path = "owner_sketch_formats.rs"]
mod sketch_formats;
#[path = "owner_sketch_geometric.rs"]
mod sketch_geometric;
#[path = "owner_sketch_interaction.rs"]
mod sketch_interaction;
#[path = "owner_sketch_picking.rs"]
mod sketch_picking;
#[path = "owner_work.rs"]
mod work;
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
    color_schemes: oblikovati_model::color_scheme::ColorSchemeRegistry,
    theme_store: Option<Box<dyn oblikovati_model::theme::ThemeSelectionStore>>,
    themes: Option<oblikovati_model::theme::ThemeLibrary>,
    window_captures: Option<crate::DesktopWindowCaptures>,
    files: FileWorkspace<O, W, F>,
    guids: G,
    factory: R,
    version: String,
    dispatch: OwnerDispatch<S>,
    caller: QueuedCaller<S>,
    refused_close: Vec<u64>,
    part_scenes: crate::part_scene::DesktopPartScenes,
    sketch_scenes: crate::sketch_scene::DesktopSketchScenes,
    command_notice: Option<DesktopCommandNotice>,
    sketch_edit: Option<(
        u64,
        oblikovati_session::SketchContinuityWatch,
        oblikovati_session::workspace::WorkspaceFocusIdentity,
    )>,
    sketch_tool: Option<crate::sketch_tool::SketchToolState>,
    dimension_serial: u64,
    sketch_formats: oblikovati_api::types::SketchFormatModes,
    sketch_construction: oblikovati_martensite::planar_curve::MartensiteSketchConstruction,
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
            color_schemes: Default::default(),
            theme_store: None,
            themes: None,
            window_captures: None,
            files,
            guids,
            factory,
            version,
            dispatch,
            caller,
            refused_close: Vec::new(),
            part_scenes: Default::default(),
            sketch_scenes: Default::default(),
            command_notice: None,
            sketch_construction: Default::default(),
            sketch_formats: Default::default(),
            sketch_tool: None,
            dimension_serial: 0,
            sketch_edit: None,
        }
    }
    /// Enable view capture with an injected output configuration.
    /// Example: `owner.with_window_captures(DesktopWindowCaptures::new(directory, namespace))`.
    pub fn with_window_captures(mut self, captures: crate::DesktopWindowCaptures) -> Self {
        self.window_captures = Some(captures);
        self
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
        let editing = self.sketch_inspection_edit();
        let host = borrowed_host(
            &mut self.files,
            &mut self.guids,
            &self.version,
            &mut self.factory,
            &mut self.part_scenes,
            &self.sketch_construction,
            &mut self.window_captures,
        )
        .with_sketch_edit(editing)
        .with_color_schemes(&mut self.color_schemes);
        match &self.themes {
            Some(library) => host.with_themes(library),
            None => host,
        }
    }
    fn drain_calls(&mut self, budget: usize) -> Result<usize, String> {
        let editing = self.sketch_inspection_edit();
        let host = borrowed_host(
            &mut self.files,
            &mut self.guids,
            &self.version,
            &mut self.factory,
            &mut self.part_scenes,
            &self.sketch_construction,
            &mut self.window_captures,
        )
        .with_sketch_edit(editing)
        .with_color_schemes(&mut self.color_schemes);
        let host = match &self.themes {
            Some(library) => host.with_themes(library),
            None => host,
        };
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
    construction: &'a dyn oblikovati_model::sketch::SketchCircleConstruction,
    captures: &'a mut Option<crate::DesktopWindowCaptures>,
) -> FileHost<'a, O, W, F> {
    let host = FileHost::new(files, guids, version)
        .with_observer_factory(factory)
        .with_part_topology(scenes)
        .with_sketch_construction(construction, Default::default());
    match captures {
        Some(captures) => host.with_window_capture(captures),
        None => host,
    }
}
#[path = "owner_placement.rs"]
mod placement;
#[cfg(test)]
#[path = "../tests/owner.rs"]
mod tests;

#[path = "owner_placement_input.rs"]
mod placement_input;

#[path = "owner_sketch_edit.rs"]
mod sketch_environment;

#[path = "owner_themes.rs"]
mod application_themes;
