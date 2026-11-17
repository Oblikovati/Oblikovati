// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadInvalidation, HeadWork, HeadWorkStatus};
use oblikovati_automation::{
    DocumentObserverFactory, FileHost,
    dispatch::{DispatchWakeup, OwnerDispatch, QueuedCaller, owner_dispatch},
};
use oblikovati_model::identity::GuidSource;
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
#[path = "owner_commands.rs"]
mod commands;
#[path = "owner_datum_overlay.rs"]
mod datum_overlay;
#[path = "owner_datum_pick.rs"]
mod datum_pick;
#[path = "owner_documents.rs"]
mod documents_gui;
#[path = "owner_offset_plane.rs"]
mod offset_plane;
#[path = "owner_sketch_tools.rs"]
mod sketch_tools_gui;
use sketch_tools_gui::gui_placement_tool;
#[path = "owner_block_placement.rs"]
pub(crate) mod block_placement;
#[path = "owner_body_properties.rs"]
mod body_properties;
#[path = "owner_browser_selection.rs"]
mod browser_selection;
#[path = "owner_browser_visibility.rs"]
mod browser_visibility;
#[path = "owner_clear_appearance.rs"]
mod clear_appearance;
#[path = "owner_dimension_annotations.rs"]
mod dimension_annotations;
#[path = "owner_dimension_query.rs"]
mod dimension_query;
#[path = "application/document-creation/mod.rs"]
mod document_creation;
#[path = "owner_drawing_create.rs"]
mod drawing_create;
#[path = "owner_drawing_projection.rs"]
mod drawing_projection;
#[path = "owner_geometric_batch.rs"]
mod geometric_batch;
#[path = "owner_inspect.rs"]
mod inspect;
#[path = "owner_line_commit.rs"]
mod line_commit;
#[path = "owner_metadata.rs"]
mod metadata;
#[path = "owner_browser.rs"]
mod model_browser;
#[path = "owner_model_pick_cue.rs"]
mod model_pick_cue;
#[path = "owner_precise_input.rs"]
pub(crate) mod owner_precise_input;
#[path = "owner_work_sketch.rs"]
mod owner_work_sketch;
#[path = "owner_parameters.rs"]
mod parameters_gui;
#[path = "owner_part_assets.rs"]
mod part_assets;
#[path = "owner_part_measure.rs"]
mod part_measure;
#[path = "owner_recent.rs"]
mod recent;
#[path = "owner_selection_filter.rs"]
mod selection_filter;
#[path = "owner_sketch_block.rs"]
mod sketch_block;
#[path = "owner_sketch_block_pick.rs"]
mod sketch_block_pick;
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
#[path = "owner_sketch_move.rs"]
pub(crate) mod sketch_move;
#[path = "owner_sketch_pattern.rs"]
mod sketch_pattern;
#[path = "owner_sketch_pick_cue.rs"]
mod sketch_pick_cue;
#[path = "owner_sketch_picking.rs"]
mod sketch_picking;
#[path = "owner_sketch_relation_context.rs"]
mod sketch_relation_context;
#[path = "owner_sketch_relation_delete.rs"]
mod sketch_relation_delete;
#[path = "owner_windows.rs"]
mod windows;
#[path = "owner_work.rs"]
mod work;
pub use commands::{DesktopCommandNotice, DesktopHistoryAction};
mod dispatch_options;
pub use dispatch_options::DesktopDispatchOptions;
/// Documents and queue admission share one owner. The worker transport contains no model.
/// ```compile_fail
/// use oblikovati_head::DesktopOwner;
/// use oblikovati_automation::dispatch::PollingDispatchWakeup;
/// fn require_send<T: Send>() {}
/// require_send::<DesktopOwner<(), (), (), (), (), PollingDispatchWakeup>>();
/// ```
pub struct DesktopOwner<O, W, F, G, R, S: DispatchWakeup> {
    marking_menus: oblikovati_model::marking_menu::MarkingMenus,
    classic_marking_menu: bool,
    marking_store: Option<Box<dyn oblikovati_model::marking_menu::MarkingMenuPreferenceAccess>>,
    keymap: oblikovati_model::keymap::Keymap,
    keymap_store: Option<Box<dyn oblikovati_model::keymap::KeymapPreferenceAccess>>,
    shortcut_keys: owner_shortcuts::ShortcutKeys,
    ambient_selection: oblikovati_session::selection::AmbientSelectionFilter,
    selection_store: Option<Box<dyn crate::SelectionPreferenceAccess>>,
    selection_mode: oblikovati_gui::GuiSelectionMode,
    sketch_alias: String,
    color_schemes: oblikovati_model::color_scheme::ColorSchemeRegistry,
    ui_scale_store: Option<Box<dyn oblikovati_model::ui_scale::UiScalePreferenceAccess>>,
    new_document_counts: std::collections::BTreeMap<&'static str, u32>,
    prompt_store: Option<Box<dyn oblikovati_model::application_prompts::PromptPreferenceAccess>>,
    prompt_frequencies: oblikovati_model::application_prompts::PromptPreferences,
    orientation_lock: bool,
    look_at_cache: std::cell::RefCell<Option<orientation_selection::LookAtCache>>,
    orientation_store:
        Option<Box<dyn oblikovati_model::orientation_preferences::OrientationPreferenceAccess>>,
    theme_store: Option<Box<dyn oblikovati_model::theme::ThemeSelectionStore>>,
    themes: Option<oblikovati_model::theme::ThemeLibrary>,
    datum_fallback_theme: Option<Result<oblikovati_model::theme::ThemeDefinition, String>>,
    window_captures: Option<crate::DesktopWindowCaptures>,
    files: FileWorkspace<O, W, F>,
    guids: G,
    factory: R,
    version: String,
    dispatch: OwnerDispatch<S>,
    caller: QueuedCaller<S>,
    refused_close: Vec<u64>,
    part_scenes: crate::part_scene::DesktopPartScenes,
    drawing_projections: std::cell::RefCell<drawing_projection::DrawingProjectionCache>,
    drawing_projection_access: oblikovati_drawing::DrawingProjectionService,
    drawing_view_preview: Option<drawing_projection::DrawingViewPreview>,
    extrusion_access: Option<crate::part_scene::extrusion::CertifiedExtrudeAccess>,
    fillet_access: Option<oblikovati_automation::CertifiedFilletAccess>,
    extrusion_preview: Option<owner_extrusion_preview::ScenePreview>,
    extrusion_distance_validation: Option<(u64, oblikovati_gui::GuiExtrudeDistanceValidation)>,
    offset_plane_validation: Option<oblikovati_gui::GuiOffsetPlaneValidation>,
    component_preview: Option<owner_extrusion_preview::ComponentPreview>,
    sketch_scenes: crate::sketch_scene::DesktopSketchScenes,
    command_notice: Option<DesktopCommandNotice>,
    /// Sketches being edited in documents that are not active, by document.
    parked_sketch_edits:
        std::collections::BTreeMap<u64, oblikovati_model::document_sketch::DocumentSketchId>,
    sketch_edit: Option<(
        u64,
        oblikovati_session::SketchContinuityWatch,
        oblikovati_session::workspace::WorkspaceFocusIdentity,
    )>,
    sketch_tool: Option<crate::sketch_tool::SketchToolState>,
    pattern_preview: Option<(
        crate::HeadSketchEdit,
        u64,
        oblikovati_viewport::ViewportSketchPreview,
    )>,
    pattern_result: Option<(u64, bool)>,
    dimension_serial: u64,
    sketch_formats: oblikovati_api::types::SketchFormatModes,
    sketch_memory: owner_precise_input::SketchCommandMemory,
    sketch_pick_cue: Option<sketch_pick_cue::SketchPickCue>,
    model_pick_cue: Option<model_pick_cue::ModelPickCue>,
    block_selection: Option<crate::HeadSketchEdit>,
    block_created: Option<String>,
    opened_hidden_sketch: Option<oblikovati_model::document_sketch::DocumentSketchId>,
    model_face: Option<browser_selection::ChosenModelFace>,
    model_edges: Vec<browser_selection::ChosenModelEdge>,
    face_appearance_picks: Vec<clear_appearance::SelectedFaceAppearance>,
    selected_occurrence: Option<(u64, oblikovati_gui::GuiOccurrencePath)>,
    part_measure: Option<part_measure::PartMeasure>,
    inspected_file: Option<oblikovati_gui::GuiInspectedFile>,
    /// The feature whose dimensions a row's Show Dimensions put in the view.
    shown_feature_dimensions: Option<(u64, oblikovati_model::feature::FeatureId)>,
    /// Values a user asked for, by document and body, with the model state they came from.
    body_physical:
        std::collections::BTreeMap<(u64, String), (u64, oblikovati_gui::GuiBodyPhysical)>,
    /// The last outlined browser row, keyed by what it was worked out from.
    browser_outline: Option<(browser_selection::BrowserOutlineSource, Vec<[[f64; 3]; 2]>)>,
    selected_edge_outlines: Option<browser_selection::EdgeOutlineCache>,
    document_windows: Vec<oblikovati_gui::GuiDocumentWindow>,
    active_window: Option<u64>,
    window_serial: u64,
    recent_documents: Vec<oblikovati_model::application_recent::RecentDocument>,
    recent_rows: Vec<oblikovati_gui::GuiRecentDocument>,
    recent_store: Option<Box<dyn oblikovati_model::application_recent::RecentDocumentAccess>>,
    sketch_options: oblikovati_model::application_options::SketchApplicationOptions,
    sketch_store: Option<Box<dyn oblikovati_model::application_options::SketchPreferenceAccess>>,
    part_store: Option<Box<dyn oblikovati_model::application_options::PartPreferenceAccess>>,
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
            marking_menus: Default::default(),
            classic_marking_menu: false,
            marking_store: None,
            keymap: oblikovati_model::keymap::Keymap::desktop_defaults(),
            keymap_store: None,
            shortcut_keys: Default::default(),
            ambient_selection: Default::default(),
            selection_store: None,
            selection_mode: Default::default(),
            sketch_alias: String::new(),
            color_schemes: Default::default(),
            ui_scale_store: None,
            new_document_counts: Default::default(),
            prompt_store: None,
            prompt_frequencies: Default::default(),
            orientation_lock: false,
            look_at_cache: std::cell::RefCell::new(None),
            orientation_store: None,
            theme_store: None,
            themes: None,
            datum_fallback_theme: None,
            window_captures: None,
            files,
            guids,
            factory,
            version,
            dispatch,
            caller,
            refused_close: Vec::new(),
            part_scenes: Default::default(),
            drawing_projections: Default::default(),
            drawing_projection_access: oblikovati_drawing::DrawingProjectionService,
            drawing_view_preview: None,
            extrusion_access: None,
            fillet_access: None,
            extrusion_preview: None,
            extrusion_distance_validation: None,
            offset_plane_validation: None,
            component_preview: None,
            sketch_scenes: Default::default(),
            command_notice: None,
            sketch_construction: Default::default(),
            sketch_formats: Default::default(),
            sketch_memory: Default::default(),
            sketch_pick_cue: None,
            model_pick_cue: None,
            block_selection: None,
            block_created: None,
            opened_hidden_sketch: None,
            model_face: None,
            model_edges: Vec::new(),
            face_appearance_picks: Vec::new(),
            selected_occurrence: None,
            part_measure: None,
            inspected_file: None,
            shown_feature_dimensions: None,
            body_physical: Default::default(),
            browser_outline: None,
            selected_edge_outlines: None,
            document_windows: Vec::new(),
            active_window: None,
            window_serial: 0,
            recent_documents: Vec::new(),
            recent_rows: Vec::new(),
            recent_store: None,
            sketch_options: Default::default(),
            sketch_store: None,
            part_store: None,
            sketch_tool: None,
            pattern_preview: None,
            pattern_result: None,
            dimension_serial: 0,
            parked_sketch_edits: Default::default(),
            sketch_edit: None,
        }
    }
    /// Enable view capture with an injected output configuration.
    /// Example: `owner.with_window_captures(DesktopWindowCaptures::new(directory, namespace))`.
    pub fn with_window_captures(mut self, captures: crate::DesktopWindowCaptures) -> Self {
        self.window_captures = Some(captures);
        self
    }
    /// Configure certified extrusion and scene replay; example: `owner.with_extrusion_resolution(resolution)?`.
    pub fn with_extrusion_resolution(
        self,
        resolution: oblikovati_martensite::PrismResolution,
    ) -> Result<Self, String> {
        self.with_extrusion_policy(oblikovati_martensite::PrismResolutionPolicy::Fixed(
            resolution,
        ))
    }
    /// Configure one policy for preview, feature creation and replay.
    /// Example: `owner.with_extrusion_policy(policy)?`.
    pub fn with_extrusion_policy(
        mut self,
        policy: oblikovati_martensite::PrismResolutionPolicy,
    ) -> Result<Self, String> {
        let policy = policy.validate()?;
        self.part_scenes.use_certified_resolution(policy);
        self.extrusion_access = Some(crate::part_scene::extrusion::CertifiedExtrudeAccess(policy));
        self.fillet_access = Some(oblikovati_automation::CertifiedFilletAccess::new(policy)?);
        Ok(self)
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
        self.synchronize_sketch_edit();
        let editing = self.sketch_inspection_edit();
        let host = owner_host::borrowed_host(
            &mut self.files,
            &mut self.guids,
            &self.version,
            &mut self.factory,
            &mut self.part_scenes,
            &self.sketch_construction,
            &mut self.window_captures,
        )
        .with_drawing_views(&self.drawing_projection_access)
        .with_sketch_edit(editing)
        .with_marking_menus(&mut self.marking_menus)
        .with_keymap(&mut self.keymap)
        .with_color_schemes(&mut self.color_schemes)
        .with_startup_projection(self.sketch_options.auto_project_origin);
        let host = match self.part_store.as_mut() {
            Some(store) => host.with_part_preferences(store.as_mut()),
            None => host,
        };
        let host = match self.marking_store.as_mut() {
            Some(store) => host.with_marking_store(store.as_mut()),
            None => host,
        };
        let host = match self.keymap_store.as_mut() {
            Some(store) => host.with_keymap_store(store.as_mut()),
            None => host,
        };
        let host = match &self.extrusion_access {
            Some(backend) => host.with_extrusion(backend),
            None => host,
        };
        let host = match &self.fillet_access {
            Some(backend) => host.with_fillet(backend),
            None => host,
        };
        match &self.themes {
            Some(library) => host.with_themes(library),
            None => host,
        }
    }
    fn drain_calls(&mut self, budget: usize) -> Result<usize, String> {
        let editing = self.sketch_inspection_edit();
        let host = owner_host::borrowed_host(
            &mut self.files,
            &mut self.guids,
            &self.version,
            &mut self.factory,
            &mut self.part_scenes,
            &self.sketch_construction,
            &mut self.window_captures,
        )
        .with_drawing_views(&self.drawing_projection_access)
        .with_sketch_edit(editing)
        .with_marking_menus(&mut self.marking_menus)
        .with_keymap(&mut self.keymap)
        .with_color_schemes(&mut self.color_schemes)
        .with_startup_projection(self.sketch_options.auto_project_origin);
        let host = match self.part_store.as_mut() {
            Some(store) => host.with_part_preferences(store.as_mut()),
            None => host,
        };
        let host = match self.marking_store.as_mut() {
            Some(store) => host.with_marking_store(store.as_mut()),
            None => host,
        };
        let host = match self.keymap_store.as_mut() {
            Some(store) => host.with_keymap_store(store.as_mut()),
            None => host,
        };
        let host = match &self.extrusion_access {
            Some(backend) => host.with_extrusion(backend),
            None => host,
        };
        let host = match &self.fillet_access {
            Some(backend) => host.with_fillet(backend),
            None => host,
        };
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
#[path = "owner_themes.rs"]
mod application_themes;
#[path = "owner_marking_menu.rs"]
mod marking_menu;
#[path = "owner_orientation_selection.rs"]
mod orientation_selection;
#[path = "owner_blend.rs"]
mod owner_blend;
#[path = "owner_command_state.rs"]
mod owner_command_state;
#[path = "owner_extrusion.rs"]
mod owner_extrusion;
#[path = "owner_host.rs"]
mod owner_host;
#[path = "owner_prompts.rs"]
mod owner_prompts;
#[path = "owner_shortcuts.rs"]
mod owner_shortcuts;
#[path = "owner_startup_sketch.rs"]
mod owner_startup_sketch;
#[path = "owner_placement.rs"]
mod placement;
#[path = "owner_placement_input.rs"]
mod placement_input;
#[path = "owner_point_preview.rs"]
mod point_preview;
#[path = "owner_sketch_edit.rs"]
mod sketch_environment;
#[cfg(test)]
#[path = "../tests/owner.rs"]
pub(crate) mod tests;
#[path = "owner_ui_scale.rs"]
mod ui_scale;

#[path = "owner_extrusion_preview.rs"]
mod owner_extrusion_preview;
