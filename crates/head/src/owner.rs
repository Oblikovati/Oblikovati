// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadInvalidation, HeadWork, HeadWorkStatus};
use oblikovati_automation::{
    DocumentObserverFactory, FileHost,
    dispatch::{DispatchWakeup, OwnerDispatch, QueuedCaller, owner_dispatch},
};
use oblikovati_model::{
    application_options::{SketchApplicationOptions, SketchPreferenceAccess},
    identity::GuidSource,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{
    TransactionObserver,
    files::FileWorkspace,
    selection::{AmbientSelectionFilter, SelectionKind},
    workspace::WorkspaceObserver,
};
use std::time::Duration;
#[path = "owner_commands.rs"]
mod commands;
#[path = "owner_cube.rs"]
mod cube;
#[path = "owner_parameters.rs"]
mod parameter_view;
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
    sketch_scenes: crate::sketch_scene::DesktopSketchScenes,
    command_notice: Option<DesktopCommandNotice>,
    sketch_edit: Option<(
        u64,
        oblikovati_session::SketchContinuityWatch,
        oblikovati_session::workspace::WorkspaceFocusIdentity,
    )>,
    placement: Option<crate::placement_input::SketchPlacementState>,
    cube_rectangle: Option<(u64, oblikovati_model::document_sketch::DocumentSketchId)>,
    sketch_construction: oblikovati_martensite::planar_curve::MartensiteSketchConstruction,
    sketch_preferences: Box<dyn SketchPreferenceAccess>,
    selection_filter: AmbientSelectionFilter,
}

fn selection_kind_name(kind: SelectionKind) -> &'static str {
    match kind {
        SelectionKind::WorkPoint => "workPoint",
        SelectionKind::WorkAxis => "workAxis",
        SelectionKind::PointCloudPoint => "pointCloudPoint",
        SelectionKind::Vertex => "vertex",
        SelectionKind::Edge => "edge",
        SelectionKind::Occurrence => "occurrence",
        SelectionKind::Body => "body",
        SelectionKind::Face => "face",
        SelectionKind::MeshFace => "meshFace",
        SelectionKind::WorkPlane => "workPlane",
        SelectionKind::Profile => "profile",
        SelectionKind::SketchEntity => "sketchEntity",
        _ => "unsupported",
    }
}

fn selection_kind_label(kind: SelectionKind) -> &'static str {
    match kind {
        SelectionKind::WorkPoint => "Work points",
        SelectionKind::WorkAxis => "Work axes",
        SelectionKind::PointCloudPoint => "Point cloud points",
        SelectionKind::Vertex => "Vertices",
        SelectionKind::Edge => "Edges",
        SelectionKind::Occurrence => "Occurrences",
        SelectionKind::Body => "Bodies",
        SelectionKind::Face => "Faces",
        SelectionKind::MeshFace => "Mesh faces",
        SelectionKind::WorkPlane => "Work planes",
        SelectionKind::Profile => "Profiles",
        SelectionKind::SketchEntity => "Sketch entities",
        _ => "Unsupported",
    }
}

fn gui_selection_filter(filter: &AmbientSelectionFilter) -> oblikovati_gui::GuiSelectionFilter {
    oblikovati_gui::GuiSelectionFilter {
        entries: filter
            .order()
            .iter()
            .map(|kind| {
                let name = selection_kind_name(*kind);
                oblikovati_gui::GuiSelectionFilterEntry {
                    kind: name.into(),
                    label: selection_kind_label(*kind).into(),
                    enabled: filter.enabled(*kind),
                }
            })
            .collect(),
    }
}

fn selection_kind(value: &str) -> Result<SelectionKind, String> {
    let kind = match value {
        "workPoint" => SelectionKind::WorkPoint,
        "workAxis" => SelectionKind::WorkAxis,
        "pointCloudPoint" => SelectionKind::PointCloudPoint,
        "vertex" => SelectionKind::Vertex,
        "edge" => SelectionKind::Edge,
        "occurrence" => SelectionKind::Occurrence,
        "body" => SelectionKind::Body,
        "face" => SelectionKind::Face,
        "meshFace" => SelectionKind::MeshFace,
        "workPlane" => SelectionKind::WorkPlane,
        "profile" => SelectionKind::Profile,
        "sketchEntity" => SelectionKind::SketchEntity,
        _ => {
            return Err(format!(
                "selection kind {value:?}: expected a filterable desktop kind"
            ));
        }
    };
    Ok(kind)
}

/// Volatile sketch preferences for embedding hosts that do not own a preference file.
/// Example: `DesktopSketchPreferences::default()` for a test-only desktop owner.
#[derive(Default)]
pub struct DesktopSketchPreferences {
    options: SketchApplicationOptions,
}
impl SketchPreferenceAccess for DesktopSketchPreferences {
    fn sketch_options(&self) -> &SketchApplicationOptions {
        &self.options
    }
    fn set_sketch_options(&mut self, options: SketchApplicationOptions) -> Result<(), String> {
        options.validate()?;
        self.options = options;
        Ok(())
    }
}

fn gui_model_tree(
    tree: oblikovati_api::wire::ModelTreeResult,
    bodies: oblikovati_api::wire::BodyListResult,
) -> Result<oblikovati_gui::GuiModelTree, String> {
    let sketches = usize::try_from(tree.sketches).map_err(|cause| {
        format!(
            "model.tree sketches {}: expected a non-negative count: {cause}",
            tree.sketches
        )
    })?;
    let expected_bodies = usize::try_from(tree.bodies).map_err(|cause| {
        format!(
            "model.tree bodies {}: expected a non-negative count: {cause}",
            tree.bodies
        )
    })?;
    let bodies = bodies
        .bodies
        .unwrap_or_default()
        .into_iter()
        .map(|body| {
            let faces = usize::try_from(body.faces).map_err(|cause| {
                format!(
                    "body {} faces {}: expected a non-negative count: {cause}",
                    body.name, body.faces
                )
            })?;
            Ok(oblikovati_gui::GuiModelBody {
                index: body.index,
                name: body.name,
                visible: body.visible,
                faces,
            })
        })
        .collect::<Result<Vec<_>, String>>()?;
    if bodies.len() != expected_bodies {
        return Err(format!(
            "model.tree bodies {}: expected body.list count {}",
            expected_bodies,
            bodies.len()
        ));
    }
    let features = tree
        .features
        .unwrap_or_default()
        .into_iter()
        .map(|feature| oblikovati_gui::GuiModelFeature {
            name: feature.name,
            kind: feature.kind,
            health: feature.health,
        })
        .collect();
    Ok(oblikovati_gui::GuiModelTree {
        document: tree.document,
        parameters: tree.parameters.unwrap_or_default(),
        sketches,
        features,
        bodies,
    })
}

fn gui_document_history(
    history: oblikovati_api::wire::TransactionHistory,
) -> oblikovati_gui::GuiDocumentHistory {
    oblikovati_gui::GuiDocumentHistory {
        document: history.document,
        name: history.name,
        position: history.position,
        entries: history
            .entries
            .unwrap_or_default()
            .into_iter()
            .map(|entry| oblikovati_gui::GuiHistoryEntry {
                label: entry.label,
                saved: entry.saved,
            })
            .collect(),
    }
}

fn gui_document_units(
    units: oblikovati_api::wire::DocumentUnitsInfo,
) -> oblikovati_gui::GuiDocumentUnits {
    oblikovati_gui::GuiDocumentUnits {
        length_unit: units.length_unit,
        angle_unit: units.angle_unit,
        mass_unit: units.mass_unit,
        time_unit: units.time_unit,
        length_precision: units.length_display_precision,
        angle_precision: units.angle_display_precision,
        length_format: units.length_display_format,
    }
}

fn gui_units_edit(
    action: &oblikovati_gui::GuiShellAction,
) -> Option<oblikovati_api::wire::SetDocumentUnitsArgs> {
    use oblikovati_gui::GuiShellAction;
    let mut edit = oblikovati_api::wire::SetDocumentUnitsArgs::default();
    match action {
        GuiShellAction::SetLengthUnit(value) => edit.length_unit = Some(Box::new(value.clone())),
        GuiShellAction::SetAngleUnit(value) => edit.angle_unit = Some(Box::new(value.clone())),
        GuiShellAction::SetMassUnit(value) => edit.mass_unit = Some(Box::new(value.clone())),
        GuiShellAction::SetTimeUnit(value) => edit.time_unit = Some(Box::new(value.clone())),
        GuiShellAction::SetLengthPrecision(value) => {
            edit.length_display_precision = Some(Box::new(*value))
        }
        GuiShellAction::SetAnglePrecision(value) => {
            edit.angle_display_precision = Some(Box::new(*value))
        }
        GuiShellAction::SetLengthFormat(value) => {
            edit.length_display_format = Some(Box::new(value.clone()))
        }
        _ => return None,
    }
    Some(edit)
}

fn gui_sketch_preferences(
    options: &SketchApplicationOptions,
) -> oblikovati_gui::GuiSketchPreferences {
    oblikovati_gui::GuiSketchPreferences {
        grid_spacing_cm: options.grid_spacing_cm,
        grid_visible: options.grid_visible,
        grid_major_every: options.grid_major_every,
        heads_up_display: options.heads_up_display,
        pointer_input: options.pointer_input,
        dimension_input: options.dimension_input,
        snap_to_points: options.snap_to_points,
        snap_to_grid: options.snap_to_grid,
        relax_mode: options.relax_mode,
        auto_project_origin: options.auto_project_origin,
        suppress_format_overrides: options.suppress_format_overrides,
    }
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
    /// Construct a desktop owner with volatile preferences; example: `DesktopOwner::new(files, guids, factory, version, wake, options)`.
    pub fn new(
        files: FileWorkspace<O, W, F>,
        guids: G,
        factory: R,
        version: String,
        wake: S,
        options: DesktopDispatchOptions,
    ) -> Self {
        Self::with_sketch_preferences(
            files,
            guids,
            factory,
            version,
            wake,
            options,
            Box::new(DesktopSketchPreferences::default()),
        )
    }
    /// Inject the application preference owner; example: `DesktopOwner::with_sketch_preferences(..., preferences)`.
    pub fn with_sketch_preferences(
        files: FileWorkspace<O, W, F>,
        guids: G,
        factory: R,
        version: String,
        wake: S,
        options: DesktopDispatchOptions,
        sketch_preferences: Box<dyn SketchPreferenceAccess>,
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
            sketch_scenes: Default::default(),
            command_notice: None,
            sketch_construction: Default::default(),
            placement: None,
            cube_rectangle: None,
            sketch_edit: None,
            sketch_preferences,
            selection_filter: Default::default(),
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
            &self.sketch_construction,
            self.sketch_preferences.as_mut(),
        )
    }
    fn drain_calls(&mut self, budget: usize) -> Result<usize, String> {
        let host = borrowed_host(
            &mut self.files,
            &mut self.guids,
            &self.version,
            &mut self.factory,
            &mut self.part_scenes,
            &self.sketch_construction,
            self.sketch_preferences.as_mut(),
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
    fn gui_command_state(&mut self) -> Result<oblikovati_gui::GuiCommandState, String> {
        let Some(document) = self.files.workspace().active_document() else {
            return Ok(Default::default());
        };
        let (undo_label, redo_label, transaction_open) = {
            let history = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?;
            (
                history.undo_label().map(str::to_owned),
                history.redo_label().map(str::to_owned),
                history.in_transaction(),
            )
        };
        let documents: Vec<oblikovati_gui::GuiDocumentTab> =
            oblikovati_api::client::Client::new(self.document_host())
                .documents()
                .list()
                .map_err(|cause| cause.to_string())?
                .documents
                .unwrap_or_default()
                .into_iter()
                .filter(|document| document.visible)
                .map(|document| oblikovati_gui::GuiDocumentTab {
                    id: document.id,
                    name: document.name,
                    dirty: document.dirty,
                    active: document.active,
                })
                .collect();
        let ribbon_tabs = oblikovati_api::client::Client::new(self.document_host())
            .ribbon()
            .list()
            .map_err(|cause| cause.to_string())?
            .tabs
            .unwrap_or_default()
            .into_iter()
            .map(|tab| oblikovati_gui::GuiRibbonTab {
                name: tab.name,
                controls: tab
                    .panels
                    .unwrap_or_default()
                    .into_iter()
                    .flat_map(|panel| panel.controls.unwrap_or_default())
                    .map(|control| oblikovati_gui::GuiRibbonControl {
                        command_id: control.command_id,
                        display_name: control.display_name,
                        icon: control.icon,
                        enabled: control.enabled,
                    })
                    .collect(),
            })
            .collect();
        let tree = oblikovati_api::client::Client::new(self.document_host())
            .model()
            .tree()
            .map_err(|cause| cause.to_string())?;
        let bodies = oblikovati_api::client::Client::new(self.document_host())
            .body()
            .list(&oblikovati_api::json::EmptyObject {})
            .map_err(|cause| cause.to_string())?;
        let histories = documents
            .iter()
            .map(|tab| {
                let history = oblikovati_api::client::Client::new(self.document_host())
                    .transactions()
                    .history(&oblikovati_api::wire::TransactionHistoryArgs { document: tab.id })
                    .map_err(|cause| cause.to_string())?;
                Ok(gui_document_history(history))
            })
            .collect::<Result<Vec<_>, String>>()?;
        let document_units = oblikovati_api::client::Client::new(self.document_host())
            .documents()
            .get_units()
            .map(gui_document_units)
            .map_err(|cause| cause.to_string())?;
        Ok(oblikovati_gui::GuiCommandState {
            undo_label,
            redo_label,
            transaction_open,
            documents,
            histories,
            document_units: Some(document_units),
            ribbon_tabs,
            model_tree: Some(gui_model_tree(tree, bodies)?),
            sketch_preferences: Some(gui_sketch_preferences(
                self.sketch_preferences.sketch_options(),
            )),
            selection_filter: Some(gui_selection_filter(&self.selection_filter)),
        })
    }
    fn gui_parameter_snapshot(
        &self,
    ) -> Result<Option<oblikovati_gui::GuiParameterSnapshot>, String> {
        parameter_view::owner_parameter_snapshot(&self.files)
    }
    fn apply_gui_parameter(
        &mut self,
        action: &oblikovati_gui::GuiParameterAction,
    ) -> Result<bool, String> {
        if let oblikovati_gui::GuiParameterAction::GroupManagement(edit) = action {
            edit.validate()?;
            return self.apply_parameter_group_management(edit);
        }
        if let oblikovati_gui::GuiParameterAction::Link(edit) = action {
            edit.validate()?;
            let (source_document, linked) =
                parameter_view::derived::linked_source_names(&self.files, edit)?;
            oblikovati_api::client::Client::new(self.document_host())
                .parameters()
                .add_derived_table(&oblikovati_api::wire::DerivedParameterTableAddArgs {
                    source_document,
                    linked: Some(linked),
                })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        }
        if let oblikovati_gui::GuiParameterAction::DeleteLinkedTable { document, table } = action {
            parameter_view::derived::require_linked_table_owner(&self.files, *document, *table)?;
            oblikovati_api::client::Client::new(self.document_host())
                .parameters()
                .delete_derived_table(&oblikovati_api::wire::DerivedParameterTableDeleteArgs {
                    id: *table,
                })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        }
        parameter_view::apply::apply_owner_parameter(&mut self.files, action)
    }
    fn apply_gui_shell_action(
        &mut self,
        action: oblikovati_gui::GuiShellAction,
    ) -> Result<bool, String> {
        if let oblikovati_gui::GuiShellAction::SetSelectionEnabled { kind, enabled } = &action {
            self.selection_filter
                .set_enabled(selection_kind(kind)?, *enabled);
            return Ok(true);
        }
        if let oblikovati_gui::GuiShellAction::MoveSelectionKind { from, to } = action {
            self.selection_filter.move_kind(from, to);
            return Ok(true);
        }
        if let oblikovati_gui::GuiShellAction::ActivateDocument(id) = &action {
            oblikovati_api::client::Client::new(self.document_host())
                .documents()
                .activate(&oblikovati_api::wire::ActivateDocumentArgs { id: *id })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        }
        if let oblikovati_gui::GuiShellAction::SetBodyVisible { index, visible } = &action {
            oblikovati_api::client::Client::new(self.document_host())
                .body()
                .set_visible(&oblikovati_api::wire::BodySetVisibleArgs {
                    body_index: *index,
                    visible: *visible,
                })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        }
        if let oblikovati_gui::GuiShellAction::CloseDocument(id) = &action {
            let result = oblikovati_api::client::Client::new(self.document_host())
                .documents()
                .close(&oblikovati_api::wire::CloseDocumentArgs {
                    id: *id,
                    force: false,
                });
            if let Err(cause) = result {
                self.command_notice = Some(DesktopCommandNotice::CloseFailed {
                    document: *id,
                    cause: cause.to_string(),
                });
            }
            return Ok(true);
        }
        if let oblikovati_gui::GuiShellAction::JumpHistoryTo { document, position } = &action {
            oblikovati_api::client::Client::new(self.document_host())
                .transactions()
                .jump_to(&oblikovati_api::wire::TransactionJumpToArgs {
                    document: *document,
                    position: *position,
                })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        }
        if let Some(units) = gui_units_edit(&action) {
            oblikovati_api::client::Client::new(self.document_host())
                .documents()
                .set_units(&units)
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        }
        let mut options = self.sketch_preferences.sketch_options().clone();
        match action {
            oblikovati_gui::GuiShellAction::SetGridSpacingCm(value) => {
                options.grid_spacing_cm = value
            }
            oblikovati_gui::GuiShellAction::SetGridVisible(value) => options.grid_visible = value,
            oblikovati_gui::GuiShellAction::SetGridMajorEvery(value) => {
                options.grid_major_every = value
            }
            oblikovati_gui::GuiShellAction::SetHeadsUpDisplay(value) => {
                options.heads_up_display = value
            }
            oblikovati_gui::GuiShellAction::SetPointerInput(value) => options.pointer_input = value,
            oblikovati_gui::GuiShellAction::SetDimensionInput(value) => {
                options.dimension_input = value
            }
            oblikovati_gui::GuiShellAction::SetSnapToPoints(value) => {
                options.snap_to_points = value
            }
            oblikovati_gui::GuiShellAction::SetSnapToGrid(value) => options.snap_to_grid = value,
            oblikovati_gui::GuiShellAction::SetRelaxMode(value) => options.relax_mode = value,
            oblikovati_gui::GuiShellAction::SetAutoProjectOrigin(value) => {
                options.auto_project_origin = value
            }
            oblikovati_gui::GuiShellAction::SetSuppressFormatOverrides(value) => {
                options.suppress_format_overrides = value
            }
            _ => return self.apply_cube_action(action),
        }
        options.validate()?;
        self.sketch_preferences.set_sketch_options(options)?;
        Ok(true)
    }

    fn process_inputs(
        &mut self,
        inputs: &[oblikovati_viewport::ViewportEvent],
    ) -> Result<bool, String> {
        self.process_document_inputs(inputs)
    }
    fn decline_sketch_edit(&mut self, cause: String) -> Result<(), String> {
        self.sketch_edit = None;
        self.placement = None;
        self.command_notice = Some(DesktopCommandNotice::SketchEditFailed { cause });
        Ok(())
    }
    fn sketch_edit(&self) -> Option<crate::HeadSketchEdit> {
        self.active_sketch_edit()
    }

    fn sketch_preview(&self) -> Option<oblikovati_viewport::ViewportSketchPreview> {
        let recipe = self.placement_preview()?.clone();
        let tool = self.placement.as_ref()?;
        let session = self.files.workspace().session(tool.document).ok()?;
        Some(oblikovati_viewport::ViewportSketchPreview::Recipe {
            solve_options: session.sketch_solve_options(),
            plane: tool.plane,
            recipe,
        })
    }

    fn process_window_inputs(
        &mut self,
        inputs: &[oblikovati_viewport::ViewportEvent],
        window: &dyn crate::HeadWindow,
    ) -> Result<bool, String> {
        self.process_placement_inputs(inputs, window)
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
        let changed = self.synchronize_sketch_edit() || changed;
        let pending = self.dispatch.pending().map_err(|error| error.to_string())? != 0;
        Ok(HeadWorkStatus {
            changed,
            pending,
            wake_after: None,
        })
    }
    fn take_sketch_scene(&mut self) -> Result<Option<crate::HeadSketchSceneUpdate>, String> {
        self.sketch_scenes.refresh(&self.files)
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
        self.placement = None;
        self.sketch_edit = None;
        self.part_scenes.clear();
        self.sketch_scenes.clear();
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
> DesktopOwner<O, W, F, G, R, S>
{
    fn apply_parameter_group_management(
        &mut self,
        edit: &oblikovati_gui::GuiParameterGroupAction,
    ) -> Result<bool, String> {
        use oblikovati_gui::GuiParameterGroupAction;
        let exists = !matches!(edit, GuiParameterGroupAction::Create { .. });
        let key = match edit {
            GuiParameterGroupAction::Create { internal_name, .. }
            | GuiParameterGroupAction::Rename { internal_name, .. }
            | GuiParameterGroupAction::Delete { internal_name, .. }
            | GuiParameterGroupAction::RemoveMember { internal_name, .. } => internal_name,
        };
        parameter_view::require_group_owner(&self.files, edit.document(), key, exists)?;
        let member_name = match edit {
            GuiParameterGroupAction::RemoveMember { parameter, .. } => Some(
                parameter_view::group_member_name(&self.files, edit.document(), *parameter)?,
            ),
            _ => None,
        };
        let client = oblikovati_api::client::Client::new(self.document_host());
        match edit {
            GuiParameterGroupAction::Create {
                internal_name,
                display_name,
                client_id,
                ..
            } => client
                .parameters()
                .add_group(&oblikovati_api::wire::ParameterGroupAddArgs {
                    internal_name: internal_name.clone(),
                    display_name: display_name.clone(),
                    client_id: client_id.clone(),
                }),
            GuiParameterGroupAction::Rename {
                internal_name,
                display_name,
                ..
            } => client.parameters().set_group_display_name(
                &oblikovati_api::wire::ParameterGroupDisplayNameArgs {
                    internal_name: internal_name.clone(),
                    display_name: display_name.clone(),
                },
            ),
            GuiParameterGroupAction::Delete {
                internal_name,
                delete_parameters,
                ..
            } => {
                client
                    .parameters()
                    .delete_group(&oblikovati_api::wire::ParameterGroupDeleteArgs {
                        internal_name: internal_name.clone(),
                        delete_parameters: *delete_parameters,
                    })
                    .map_err(|cause| cause.to_string())?;
                return Ok(true);
            }
            GuiParameterGroupAction::RemoveMember { parameter, .. } => {
                let parameter = member_name.ok_or_else(|| {
                    format!("GUI parameter group member {parameter}: expected a resolved name")
                })?;
                client.parameters().remove_group_member(
                    &oblikovati_api::wire::ParameterGroupMemberArgs {
                        internal_name: key.to_owned(),
                        parameter,
                    },
                )
            }
        }
        .map_err(|cause| cause.to_string())?;
        Ok(true)
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
    preferences: &'a mut dyn SketchPreferenceAccess,
) -> FileHost<'a, O, W, F> {
    FileHost::new(files, guids, version)
        .with_observer_factory(factory)
        .with_part_topology(scenes)
        .with_sketch_construction(construction, Default::default())
        .with_sketch_preferences(preferences)
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
