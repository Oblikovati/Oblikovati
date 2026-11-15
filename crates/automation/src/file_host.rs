// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    EDIT_METHODS, FileQueryHost, TRANSACTION_METHODS, command_owner, decode, encode, host::busy,
    transactions,
};
use oblikovati_api::{Error, client::Caller, wire::*};
use oblikovati_model::{attachment::ExternalAttachmentFiles, identity::GuidSource};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
use std::cell::RefCell;
mod assembly_occurrence_edits;
mod assembly_placement;
mod assembly_references;
mod attachments;
mod document_update;
pub use assembly_references::{AssemblyReferenceFailure, AssemblyReferenceReport};
pub use document_update::DOCUMENT_UPDATE_METHODS;
mod assembly_snapshots;
mod close;
mod derived_links;
mod drawing_snapshots;
mod selection;
pub use derived_links::DERIVED_COMMAND_METHODS;
mod create;
pub use create::creation_package;
mod lifecycle;
mod native_files;
mod open;
mod sketch_construction;
mod sketch_inspection;
mod sketch_preferences;
pub use open::DocumentObserverFactory;
pub(crate) use sketch_construction::SketchConstructionAccess;

/// Implemented calls that require the file owner.
pub const FILE_COMMAND_METHODS: [&str; 12] = [
    METHOD_DOCUMENT_SET_END_OF_PART,
    METHOD_DOCUMENT_SET_SKETCH_SETTINGS,
    METHOD_DOCUMENTS_ADD_ATTACHMENT,
    METHOD_DOCUMENTS_ACTIVATE,
    METHOD_DOCUMENTS_OPEN,
    METHOD_DOCUMENTS_CREATE,
    METHOD_DOCUMENTS_BATCH_SAVE,
    METHOD_DOCUMENTS_SAVE,
    METHOD_DOCUMENTS_SAVE_AS,
    METHOD_DOCUMENTS_SAVE_COPY_AS,
    METHOD_DOCUMENTS_CLOSE,
    METHOD_DOCUMENTS_CLOSE_ALL,
];
/// Assembly mutations currently owned by the file host.
pub const ASSEMBLY_COMMAND_METHODS: [&str; 2] = [METHOD_ASSEMBLY_SUPPRESS, METHOD_ASSEMBLY_PLACE];
/// Application preferences that require the injected persistent preference owner.
pub const APPLICATION_PREFERENCE_METHODS: [&str; 2] = [
    METHOD_APPLICATION_GET_HUD_OPTIONS,
    METHOD_APPLICATION_SET_HUD_OPTIONS,
];

/// Synchronous API access that retains file, history, and identity ownership.
pub struct FileHost<'a, O: TransactionObserver, W, F> {
    context: RefCell<FileContext<'a, O, W, F>>,
}
struct FileContext<'a, O: TransactionObserver, W, F> {
    marking_menus: Option<&'a mut oblikovati_model::marking_menu::MarkingMenus>,
    marking_store: Option<&'a mut dyn oblikovati_model::marking_menu::MarkingMenuPreferenceAccess>,
    keymap: Option<&'a mut oblikovati_model::keymap::Keymap>,
    keymap_store: Option<&'a mut dyn oblikovati_model::keymap::KeymapPreferenceAccess>,
    color_schemes: Option<&'a mut oblikovati_model::color_scheme::ColorSchemeRegistry>,
    themes: Option<&'a oblikovati_model::theme::ThemeLibrary>,
    window_capture: Option<&'a mut dyn crate::WindowCaptureAccess>,
    sketch_edit: Option<crate::SketchInspectionEdit>,
    files: &'a mut FileWorkspace<O, W, F>,
    source: &'a mut dyn GuidSource,
    version: &'a str,
    external: Option<&'a mut dyn ExternalAttachmentFiles>,
    observers: Option<&'a mut dyn DocumentObserverFactory<Observer = O>>,
    sketch_construction: Option<SketchConstructionAccess<'a>>,
    part_topology: Option<&'a mut dyn crate::PartTopologyAccess<O, W, F>>,
    part_update: Option<&'a mut dyn crate::PartUpdateAccess<O, W, F>>,
    extrusion: Option<&'a dyn crate::ExtrudeRegionAccess>,
    drawing_views: Option<&'a dyn crate::DrawingViewProjectionAccess>,
    sketch_preferences:
        Option<&'a mut dyn oblikovati_model::application_options::SketchPreferenceAccess>,
}
impl<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'a, O, W, F> {
    /// Inject part replay ownership; example: `host.with_part_update(&mut replay)`.
    pub fn with_part_update(
        mut self,
        replay: &'a mut dyn crate::PartUpdateAccess<O, W, F>,
    ) -> Self {
        self.context.get_mut().part_update = Some(replay);
        self
    }
    /// Share the desktop's marking menus; example: `host.with_marking_menus(&mut menus)`.
    pub fn with_marking_menus(
        mut self,
        menus: &'a mut oblikovati_model::marking_menu::MarkingMenus,
    ) -> Self {
        self.context.get_mut().marking_menus = Some(menus);
        self
    }
    /// Share marking-menu storage; example: `host.with_marking_store(&mut store)`.
    pub fn with_marking_store(
        mut self,
        store: &'a mut dyn oblikovati_model::marking_menu::MarkingMenuPreferenceAccess,
    ) -> Self {
        self.context.get_mut().marking_store = Some(store);
        self
    }
    /// Share the application keymap; example: `host.with_keymap(&mut keymap)`.
    pub fn with_keymap(mut self, keymap: &'a mut oblikovati_model::keymap::Keymap) -> Self {
        self.context.get_mut().keymap = Some(keymap);
        self
    }
    /// Share keymap storage; example: `host.with_keymap_store(&mut store)`.
    pub fn with_keymap_store(
        mut self,
        store: &'a mut dyn oblikovati_model::keymap::KeymapPreferenceAccess,
    ) -> Self {
        self.context.get_mut().keymap_store = Some(store);
        self
    }
    /// Supply exact drawing projection; example: `host.with_drawing_views(&backend)`.
    pub fn with_drawing_views(
        mut self,
        backend: &'a dyn crate::DrawingViewProjectionAccess,
    ) -> Self {
        self.context.get_mut().drawing_views = Some(backend);
        self
    }
    /// Supply profile discovery and candidate validation; example: `host.with_extrusion(&backend)`.
    pub fn with_extrusion(mut self, backend: &'a dyn crate::ExtrudeRegionAccess) -> Self {
        self.context.get_mut().extrusion = Some(backend);
        self
    }
    /// Borrow the application viewport palettes; example: `host.with_color_schemes(&mut registry)`.
    pub fn with_color_schemes(
        mut self,
        registry: &'a mut oblikovati_model::color_scheme::ColorSchemeRegistry,
    ) -> Self {
        self.context.get_mut().color_schemes = Some(registry);
        self
    }
    /// Share the application theme library; example: `host.with_themes(&library)`.
    pub fn with_themes(mut self, library: &'a oblikovati_model::theme::ThemeLibrary) -> Self {
        self.context.get_mut().themes = Some(library);
        self
    }
    /// Inject the native capture queue; example: `host.with_window_capture(&mut captures)`.
    pub fn with_window_capture(mut self, capture: &'a mut dyn crate::WindowCaptureAccess) -> Self {
        self.context.get_mut().window_capture = Some(capture);
        self
    }
    /// Supply current desktop edit state without changing the document recipe.
    /// Example: `host.with_sketch_edit(Some(edit.clone()))`.
    pub fn with_sketch_edit(mut self, edit: Option<crate::SketchInspectionEdit>) -> Self {
        self.context.get_mut().sketch_edit = edit;
        self
    }
    /// Share the native exact-reference owner; example: `host.with_part_topology(&mut scenes)`.
    pub fn with_part_topology(
        mut self,
        references: &'a mut dyn crate::PartTopologyAccess<O, W, F>,
    ) -> Self {
        self.context.get_mut().part_topology = Some(references);
        self
    }
    /// Borrow owned files and identity generation; example: `FileHost::new(&mut files, &mut guids, "alpha")`.
    pub fn new(
        files: &'a mut FileWorkspace<O, W, F>,
        source: &'a mut dyn GuidSource,
        version: &'a str,
    ) -> Self {
        Self {
            context: RefCell::new(FileContext {
                marking_menus: None,
                marking_store: None,
                keymap: None,
                keymap_store: None,
                color_schemes: None,
                themes: None,
                window_capture: None,
                sketch_edit: None,
                files,
                source,
                version,
                observers: None,
                external: None,
                sketch_preferences: None,
                part_topology: None,
                part_update: None,
                extrusion: None,
                drawing_views: None,
                sketch_construction: None,
            }),
        }
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> Caller
    for FileHost<'_, O, W, F>
{
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        let mut context = self.context.try_borrow_mut().map_err(|_| busy(method))?;
        if crate::marking_menu::METHODS.contains(&method) {
            let FileContext {
                marking_menus,
                marking_store,
                ..
            } = &mut *context;
            let menus = marking_menus.as_deref_mut().ok_or_else(|| {
                Error(format!(
                    "marking menu call {method:?}: expected an injected menu owner"
                ))
            })?;
            let mut candidate = menus.clone();
            let reply = crate::marking_menu::apply(&mut candidate, method, request)?;
            if method == METHOD_UI_SET_MARKING_MENU {
                if let Some(store) = marking_store.as_deref_mut() {
                    store.save(&candidate).map_err(Error)?;
                }
                *menus = candidate;
            }
            return Ok(reply);
        }
        if crate::keymap::METHODS.contains(&method) {
            let FileContext {
                keymap,
                keymap_store,
                ..
            } = &mut *context;
            let keymap = keymap.as_deref_mut().ok_or_else(|| {
                Error(format!(
                    "keymap call {method:?}: expected an injected application keymap"
                ))
            })?;
            let mut candidate = keymap.clone();
            let reply = crate::keymap::apply(&mut candidate, method, request)?;
            if method != METHOD_KEYMAP_LIST && method != METHOD_KEYMAP_EXPORT {
                if let Some(store) = keymap_store.as_deref_mut() {
                    store.set_keymap(&candidate.export()).map_err(Error)?;
                }
                *keymap = candidate;
            }
            return Ok(reply);
        }
        if crate::COLOR_SCHEME_METHODS.contains(&method) {
            let registry = context.color_schemes.as_deref_mut().ok_or_else(|| Error(format!(
                "color scheme call {method:?}: expected an injected application palette registry"
            )))?;
            return crate::color_schemes::apply(registry, method, request);
        }
        if crate::THEME_QUERY_METHODS.contains(&method) {
            let library = context.themes.ok_or_else(|| {
                Error(format!(
                    "theme call {method:?}: expected an injected application theme library"
                ))
            })?;
            return crate::ThemeQueryHost::new(library).call(method, request);
        }
        if crate::WINDOW_CAPTURE_METHODS.contains(&method) {
            return crate::window_capture::apply(&mut context.window_capture, request);
        }
        if APPLICATION_PREFERENCE_METHODS.contains(&method) {
            return context.hud_options(method, request);
        }
        if crate::CLIENT_COMMAND_METHODS.contains(&method) {
            return crate::clients::apply(context.files.connected_clients_mut(), method, request);
        }
        if method == METHOD_DOCUMENTS_LIST_ATTACHMENTS {
            return context.list_attachments(decode(method, request)?);
        }
        if crate::DATUM_COMMAND_METHODS.contains(&method) {
            if method == METHOD_WORK_PLANES_CREATE {
                let FileContext {
                    files,
                    part_update,
                    part_topology,
                    ..
                } = &mut *context;
                if let Some(reply) =
                    crate::datums::face_offset::apply(files, part_update, part_topology, request)?
                {
                    return Ok(reply);
                }
            }
            return crate::datums::apply(context.files, method, request);
        }
        if crate::SKETCH_COMMAND_METHODS.contains(&method) {
            return context.sketch_command(method, request);
        }
        if method == METHOD_DOCUMENT_SET_SKETCH_SETTINGS {
            return crate::document_sketch_settings::apply(context.files, decode(method, request)?);
        }
        if method == METHOD_DOCUMENT_SET_END_OF_PART {
            let FileContext {
                files,
                part_topology,
                ..
            } = &mut *context;
            return crate::end_of_part::apply(files, part_topology, request);
        }
        if method == METHOD_FEATURES_GET {
            let FileContext {
                files,
                part_topology,
                ..
            } = &mut *context;
            return crate::feature_lifecycle::query(files, part_topology, request);
        }
        if method == METHOD_FEATURES_DELETE {
            let FileContext {
                files,
                part_topology,
                ..
            } = &mut *context;
            return crate::feature_lifecycle::delete(files, part_topology, request);
        }
        if method == METHOD_MODEL_TREE {
            let FileContext {
                files,
                part_topology,
                ..
            } = &mut *context;
            return crate::model_tree::query(files, part_topology, request);
        }
        if crate::FEATURE_LIFECYCLE_METHODS.contains(&method) {
            let FileContext {
                files,
                part_topology,
                ..
            } = &mut *context;
            return crate::feature_lifecycle::apply(files, part_topology, method, request);
        }
        context.owned_command(method, request)
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    fn owned_command(&mut self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        if [METHOD_SKETCH_LIST, METHOD_SKETCH_GET].contains(&method)
            && let Some(reply) = self.part_sketch_inspection(method, request)?
        {
            return Ok(reply);
        }
        if crate::DATUM_QUERY_METHODS.contains(&method)
            && (self.part_update.is_some() || self.part_topology.is_some())
        {
            let document = crate::part_owner::active_part(self.files)?;
            let owner: &mut dyn crate::PartUpdateAccess<O, W, F> =
                match self.part_update.as_deref_mut() {
                    Some(owner) => owner,
                    None => self.part_topology.as_deref_mut().ok_or_else(|| {
                        Error(format!(
                            "document {document} plane query: expected a part replay owner"
                        ))
                    })?,
                };
            let evaluation = owner
                .datum_evaluation(self.files, document)
                .map_err(Error)?;
            return crate::datums::query_evaluated(self.files, method, request, &evaluation);
        }
        if DOCUMENT_UPDATE_METHODS.contains(&method) {
            return document_update::apply(
                self.files,
                &mut self.part_update,
                &mut self.part_topology,
                method,
                request,
            );
        }
        if method == METHOD_ASSEMBLY_PLACE {
            return assembly_placement::place(self.files, decode(method, request)?);
        }
        if method == METHOD_ASSEMBLY_SUPPRESS {
            return assembly_occurrence_edits::suppress_wire(self.files, decode(method, request)?);
        }
        if method == oblikovati_api::wire::METHOD_DRAWING_SET_MODEL_REFERENCE {
            let result = crate::drawing_reference::apply(self.files, decode(method, request)?)?;
            self.synchronize_drawing_references();
            return Ok(result);
        }
        if method == oblikovati_api::wire::METHOD_DRAWING_VIEWS_ADD_BASE {
            let result = crate::drawing_views::add_base(
                self.files,
                self.drawing_views,
                decode(method, request)?,
            )?;
            self.synchronize_drawing_references();
            return Ok(result);
        }
        if method == METHOD_FEATURES_ADD {
            return crate::extrusion::apply(self.files, self.extrusion, request);
        }
        if method == METHOD_FEATURES_SET_SUPPRESSED {
            return crate::feature_suppression::apply(self.files, self.extrusion, request);
        }
        if crate::BODY_COMMAND_METHODS.contains(&method) {
            return crate::body_commands::apply(
                self.files,
                &mut self.part_topology,
                method,
                request,
            );
        }
        if crate::BODY_QUERY_METHODS.contains(&method) {
            return crate::part_topology::query(
                self.files,
                &mut self.part_topology,
                method,
                request,
            );
        }
        if crate::ASSET_ASSIGNMENT_METHODS.contains(&method) {
            return crate::asset_assignments::apply(
                self.files,
                &mut self.part_topology,
                method,
                request,
            );
        }
        if crate::ASSET_COMMAND_METHODS.contains(&method) {
            return crate::asset_commands::apply(self.files, method, request);
        }
        if crate::SKETCH_OPTION_METHODS.contains(&method) {
            return self.sketch_option_command(method, request);
        }
        if EDIT_METHODS.contains(&method) {
            return command_owner::edit(self.files, method, request);
        }
        if TRANSACTION_METHODS.contains(&method) {
            let result = transactions::apply(self.files, method, request)?;
            if method != METHOD_TRANSACTION_BEGIN {
                self.synchronize_drawing_references();
                self.synchronize_assembly_references().map_err(|error| {
                    Error(format!("transaction {method} was applied, but reference synchronization failed: {error}"))
                })?;
            }
            return Ok(result);
        }
        if DERIVED_COMMAND_METHODS.contains(&method) {
            return self.derived_command(method, request);
        }
        if FILE_COMMAND_METHODS.contains(&method) {
            return self.lifecycle_command(method, request);
        }
        FileQueryHost::new(self.files)
            .with_sketch_edit(self.sketch_edit.clone())
            .call(method, request)
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    fn synchronize_drawing_references(&mut self) {
        for document in self.files.workspace().documents().to_vec() {
            self.open_drawing_reference(document);
        }
    }
    fn synchronize_assembly_references(&mut self) -> Result<(), Error> {
        let documents = self.files.workspace().documents().to_vec();
        for document in documents {
            let Ok(session) = self.files.workspace().session(document) else {
                continue;
            };
            let is_assembly = session.snapshot().assembly.is_some();
            if is_assembly {
                self.resolve_assembly_references(document)?;
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests;
