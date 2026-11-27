// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_session::files::FileWorkspaceError;
use std::path::PathBuf;

pub(super) type SourceBindings = Vec<(u64, Option<PathBuf>)>;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn lifecycle_command(
        &mut self,
        method: &str,
        request: &[u8],
    ) -> Result<Vec<u8>, Error> {
        if method == METHOD_DOCUMENTS_LOAD_FULL {
            return self.load_full(decode(method, request)?);
        }
        let before = self.source_bindings()?;
        let response = self.command(method, request);
        if before == self.source_bindings()? {
            return response;
        }
        // Close-all and batch save can change ownership before a later item fails.
        let refresh = self
            .files
            .refresh_all_derived()
            .map_err(|error| error.to_string())
            .and_then(command_owner::check_refresh_report);
        combine_lifecycle_result(method, response, refresh)
    }

    pub(super) fn source_bindings(&self) -> Result<SourceBindings, Error> {
        self.files
            .workspace()
            .documents()
            .iter()
            .map(|&id| {
                let path = match self.files.path(id) {
                    Ok(path) => Some(path.to_owned()),
                    Err(FileWorkspaceError::UnboundDocument(_)) => None,
                    Err(error) => return Err(Error(error.to_string())),
                };
                Ok((id, path))
            })
            .collect()
    }
    pub(super) fn refresh_after_native_file_change(
        &mut self,
        before: &SourceBindings,
        operation: &str,
    ) -> Result<(), String> {
        if *before == self.source_bindings().map_err(|error| error.to_string())? {
            return Ok(());
        }
        self.files
            .refresh_all_derived()
            .map_err(|error| format!("{operation}: {error}"))
            .and_then(command_owner::check_refresh_report)
    }
}

pub(super) fn combine_lifecycle_result<T>(
    method: &str,
    response: Result<T, Error>,
    refresh: Result<(), String>,
) -> Result<T, Error> {
    let Err(reason) = refresh else {
        return response;
    };
    let command = match response {
        Ok(_) => "command completed".into(),
        Err(error) => format!("command failed after changing file ownership: {error}"),
    };
    Err(Error(format!(
        "{method}: {command}; derived refresh is incomplete: {reason}"
    )))
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    fn command(&mut self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        match method {
            METHOD_DOCUMENTS_ADD_ATTACHMENT => self.add_attachment(decode(method, request)?),
            METHOD_DOCUMENTS_CREATE => self.create(decode(method, request)?),
            METHOD_DOCUMENTS_OPEN => self.open(decode(method, request)?),
            METHOD_DOCUMENTS_ACTIVATE => self.activate(decode(method, request)?),
            METHOD_DOCUMENTS_BATCH_SAVE => self.batch_save(decode(method, request)?),
            METHOD_DOCUMENTS_SAVE => self.save(decode(method, request)?),
            METHOD_DOCUMENTS_SAVE_AS => self.save_as(decode(method, request)?),
            METHOD_DOCUMENTS_SAVE_COPY_AS => self.save_copy(decode(method, request)?),
            METHOD_DOCUMENTS_CLOSE_ALL => self.close_all(decode(method, request)?),
            METHOD_DOCUMENTS_CLOSE => self.close(decode(method, request)?),
            _ => Err(Error(format!(
                "file command {method:?}: expected one of {FILE_COMMAND_METHODS:?}"
            ))),
        }
    }

    fn batch_save(&mut self, request: BatchSaveArgs) -> Result<Vec<u8>, Error> {
        let backend = self.drawing_views;
        let topology = &mut self.part_topology;
        let replay = &mut self.part_update;
        let response = self
            .files
            .batch_save_with_report(request, self.version, self.source, |files, id, path| {
                let owner: Option<&mut dyn crate::PartUpdateAccess<O, W, F>> =
                    match replay.as_deref_mut() {
                        Some(owner) => Some(owner),
                        None => topology
                            .as_deref_mut()
                            .map(|owner| owner as &mut dyn crate::PartUpdateAccess<O, W, F>),
                    };
                drawing_snapshots::stage(files, backend, owner, id)?;
                let access = topology.as_deref_mut();
                Ok(assembly_snapshots::stage(files, access, id, path)?
                    .into_iter()
                    .collect())
            })
            .map_err(|error| Error(error.to_string()))?;
        encode(&response)
    }

    fn activate(&mut self, request: ActivateDocumentArgs) -> Result<Vec<u8>, Error> {
        let response = encode(&OKResult { ok: true })?;
        self.files
            .activate(request.id)
            .map_err(|error| Error(error.to_string()))?;
        Ok(response)
    }

    fn save(&mut self, request: SaveDocumentArgs) -> Result<Vec<u8>, Error> {
        let path = self
            .files
            .path(request.document)
            .map_err(|error| Error(error.to_string()))?
            .to_owned();
        let name = path
            .to_str()
            .ok_or_else(|| {
                Error(format!(
                    "save path {path:?}: expected a UTF-8 path for the canonical API response"
                ))
            })?
            .to_owned();
        self.stage_drawing_views(request.document)?;
        let warnings = self.stage_assembly_view(request.document, &path)?;
        let response = encode(&SaveDocumentResult {
            full_document_name: name,
            warnings,
        })?;
        let result = self
            .files
            .save(request.document, self.version, self.source)
            .map_err(|error| Error(error.to_string()));
        self.files.clear_assembly_view(request.document);
        result?;
        Ok(response)
    }

    fn save_as(&mut self, request: SaveDocumentAsArgs) -> Result<Vec<u8>, Error> {
        self.stage_drawing_views(request.document)?;
        let document = request.document;
        let warnings = self.stage_assembly_view(
            document,
            std::path::Path::new(&request.new_full_document_name),
        )?;
        let result = self
            .files
            .save_as_report(request, self.version, self.source)
            .map_err(|error| Error(error.to_string()));
        self.files.clear_assembly_view(document);
        let mut response = result?;
        response.warnings = warnings;
        encode(&response)
    }

    fn save_copy(&mut self, request: SaveCopyAsArgs) -> Result<Vec<u8>, Error> {
        self.stage_drawing_views(request.document)?;
        let destination = std::path::Path::new(&request.target_file_name);
        let warnings = self.stage_assembly_view(request.document, destination)?;
        let response = encode(&SaveDocumentResult {
            full_document_name: request.target_file_name.clone(),
            warnings,
        })?;
        let result = self
            .files
            .save_copy(
                request.document,
                destination,
                request.metadata.as_deref(),
                self.version,
                self.source,
            )
            .map_err(|error| Error(error.to_string()));
        self.files.clear_assembly_view(request.document);
        result?;
        Ok(response)
    }
    pub(super) fn stage_assembly_view(
        &mut self,
        document: u64,
        destination: &std::path::Path,
    ) -> Result<Option<Vec<SaveWarning>>, Error> {
        let access = self.part_topology.as_deref_mut();
        let warning =
            assembly_snapshots::stage(self.files, access, document, destination).map_err(Error)?;
        Ok(warning.map(|warning| vec![warning]))
    }
    pub(super) fn stage_drawing_views(&mut self, document: u64) -> Result<(), Error> {
        let owner: Option<&mut dyn crate::PartUpdateAccess<O, W, F>> =
            match self.part_update.as_deref_mut() {
                Some(owner) => Some(owner),
                None => self
                    .part_topology
                    .as_deref_mut()
                    .map(|owner| owner as &mut dyn crate::PartUpdateAccess<O, W, F>),
            };
        drawing_snapshots::stage(self.files, self.drawing_views, owner, document).map_err(Error)
    }
}
