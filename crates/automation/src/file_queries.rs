// SPDX-License-Identifier: GPL-2.0-only

use crate::{SessionQueryHost, encode};
use oblikovati_api::{Error, client::Caller, wire::*};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Additional queries that require a file owner, rather than bare recipe sessions.
pub const FILE_QUERY_METHODS: [&str; 18] = [
    METHOD_ASSEMBLY_OCCURRENCES,
    METHOD_SKETCH_LIST,
    METHOD_SKETCH_GET,
    METHOD_DOCUMENT_GET_END_OF_PART,
    METHOD_DOCUMENT_GET_SKETCH_SETTINGS,
    METHOD_DOCUMENTS_LIST,
    METHOD_PARAMETERS_DERIVED_TABLES_LIST,
    METHOD_SKETCH_CONSTRAINT_STATUS,
    METHOD_SKETCH_CONSTRAINTS,
    METHOD_SKETCH_CONSTRAINT_GLYPH_VISIBILITY,
    METHOD_SKETCH_DIMENSIONS,
    METHOD_SKETCH_ENTITIES,
    METHOD_SKETCH_PROFILES,
    METHOD_SKETCH_REFERENCE_KEY,
    METHOD_SKETCH_RESOLVE_REFERENCE,
    METHOD_WORK_AXES_LIST,
    METHOD_WORK_POINTS_LIST,
    METHOD_WORK_PLANES_LIST,
];
pub(crate) mod assembly;
pub(crate) mod derived;

/// Immutable API access to file-owned documents and their session state.
pub struct FileQueryHost<'a, O, W, F> {
    files: &'a FileWorkspace<O, W, F>,
    sketch_edit: Option<crate::SketchInspectionEdit>,
    profile_policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
    part_sketch_continuity: Option<crate::PartSketchContinuity>,
    sketch_datums: std::cell::OnceCell<
        Result<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation, String>,
    >,
}
impl<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>
    FileQueryHost<'a, O, W, F>
{
    /// Borrow file-owned state; example: `Client::new(FileQueryHost::new(&files))`.
    pub fn new(files: &'a FileWorkspace<O, W, F>) -> Self {
        Self {
            files,
            sketch_edit: None,
            profile_policy: None,
            part_sketch_continuity: None,
            sketch_datums: std::cell::OnceCell::new(),
        }
    }
    /// Supply the live desktop edit guards; example: `host.with_sketch_edit(Some(edit.clone()))`.
    pub fn with_sketch_edit(mut self, edit: Option<crate::SketchInspectionEdit>) -> Self {
        self.sketch_edit = edit;
        self
    }
    /// Supply the owning extrusion policy used for profile indices and areas.
    /// Example: `host.with_profile_policy(Some(policy))`.
    pub fn with_profile_policy(
        mut self,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
    ) -> Self {
        self.profile_policy = policy;
        self
    }
    /// Replay the current part for sketch queries; example: `host.with_part_sketch_continuity(resolver)`.
    pub fn with_part_sketch_continuity(mut self, resolver: crate::PartSketchContinuity) -> Self {
        self.part_sketch_continuity = Some(resolver);
        self
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> Caller
    for FileQueryHost<'_, O, W, F>
{
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        if method == METHOD_MODEL_STATES_VERIFY_TABLE {
            let request = crate::decode(method, request)?;
            return encode(&crate::model_states::verify_table(
                self.files.workspace(),
                request,
                |id, candidate| self.files.verify_named_candidate(id, candidate),
            )?);
        }
        if method == METHOD_ASSEMBLY_OCCURRENCES {
            let _: oblikovati_api::json::EmptyObject = crate::decode(method, request)?;
            return encode(&assembly::occurrences(self.files)?);
        }
        if crate::ASSET_QUERY_METHODS.contains(&method) {
            return crate::assets::query(self.files, method, request);
        }
        if crate::DATUM_QUERY_METHODS.contains(&method) {
            return crate::datums::query(self.files, method, request);
        }
        if method == METHOD_SKETCH_PROFILES {
            return crate::sketches::profiles::query(
                self.files,
                crate::decode(method, request)?,
                self.profile_policy,
            );
        }
        if crate::SKETCH_QUERY_METHODS.contains(&method) {
            let editing = self
                .sketch_edit
                .as_ref()
                .and_then(|edit| edit.current(self.files.workspace()));
            let focused = self
                .sketch_edit
                .as_ref()
                .and_then(|edit| edit.focused_identity(self.files.workspace()));
            let editing = pending_sketch_edit(self.files.workspace(), editing, focused);
            if [METHOD_SKETCH_LIST, METHOD_SKETCH_GET].contains(&method)
                && let Some(resolver) = &self.part_sketch_continuity
            {
                let datums = self.sketch_datums_for_query(resolver, method, request)?;
                return crate::sketches::inspection::query_with_datums(
                    self.files, method, request, editing, datums,
                );
            }
            return crate::sketches::query(self.files, method, request, editing);
        }
        if method == METHOD_RIBBON_LIST {
            let document = self
                .files
                .workspace()
                .active_document()
                .map(|id| self.files.document_type(id))
                .transpose()
                .map_err(|error| Error(error.to_string()))?;
            return encode(&crate::ribbon::list(crate::ribbon::key_for_document(
                document,
            )));
        }
        if method == METHOD_DOCUMENT_GET_END_OF_PART {
            return crate::end_of_part::query(self.files, request);
        }
        if method == METHOD_DOCUMENT_GET_SKETCH_SETTINGS {
            return crate::document_sketch_settings::query(
                self.files,
                crate::decode(method, request)?,
            );
        }
        if method == METHOD_PARAMETERS_DERIVED_TABLES_LIST {
            return encode(&derived::list(self.files)?);
        }
        if method != METHOD_DOCUMENTS_LIST {
            return SessionQueryHost::new(self.files.workspace()).call(method, request);
        }
        encode(&ListDocumentsResult {
            documents: Some(self.documents()?),
        })
    }
}
pub(crate) fn pending_sketch_edit<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &oblikovati_session::workspace::WorkspaceSession<O, W>,
    editing: Option<(u64, oblikovati_model::document_sketch::DocumentSketchId)>,
    focused: Option<(u64, oblikovati_model::document_sketch::DocumentSketchId)>,
) -> Option<(u64, oblikovati_model::document_sketch::DocumentSketchId)> {
    let document = workspace.active_document()?;
    let Some(navigation) = workspace
        .session(document)
        .ok()
        .and_then(|session| session.pending_sketch_navigation())
    else {
        return editing;
    };
    let current = editing
        .filter(|(owner, _)| *owner == document)
        .map(|(_, sketch)| sketch);
    let focused = focused
        .filter(|(owner, _)| *owner == document)
        .map(|(_, sketch)| sketch);
    navigation
        .target(current, focused)
        .map_or(editing, |target| target.map(|sketch| (document, sketch)))
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileQueryHost<'_, O, W, F> {
    fn sketch_datums_for_query(
        &self,
        resolver: &crate::PartSketchContinuity,
        method: &str,
        request: &[u8],
    ) -> Result<Option<&oblikovati_model::feature::work_geometry::WorkGeometryEvaluation>, Error>
    {
        let document = crate::sketches::owner::active_sketch_host(self.files)?;
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|error| Error(error.to_string()))?;
        if !crate::sketches::inspection::query_reads_work_geometry(
            session.snapshot(),
            method,
            request,
        )? {
            return Ok(None);
        }
        self.sketch_datums
            .get_or_init(|| {
                resolver
                    .datums_for_document(workspace, document)
                    .map_err(|error| error.0)
            })
            .as_ref()
            .map(Some)
            .map_err(|error| Error(error.clone()))
    }

    fn documents(&self) -> Result<Vec<DocumentInfo>, Error> {
        self.files
            .workspace()
            .documents()
            .iter()
            .map(|id| {
                self.files
                    .document_info(*id)
                    .map_err(|error| Error(error.to_string()))
            })
            .collect()
    }
}
