// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::application_options::SketchPreferenceAccess;
impl<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'a, O, W, F> {
    /// Share the application's preference owner across calls and client instances.
    /// Example: `FileHost::new(files, guids, "alpha").with_sketch_preferences(&mut preferences)`.
    pub fn with_sketch_preferences(
        mut self,
        preferences: &'a mut dyn SketchPreferenceAccess,
    ) -> Self {
        self.context.get_mut().sketch_preferences = Some(preferences);
        self
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn sketch_command(
        &mut self,
        method: &str,
        request: &[u8],
    ) -> Result<Vec<u8>, Error> {
        if method == METHOD_SKETCH_ADD_ENTITY {
            return crate::sketches::add_entity::apply(
                self.files,
                crate::decode(method, request)?,
                self.sketch_construction,
            );
        }
        if method != METHOD_SKETCH_CREATE {
            let editing = self
                .sketch_edit
                .as_ref()
                .and_then(|edit| edit.current(self.files.workspace()));
            return crate::sketches::apply(self.files, method, request, editing);
        }
        let project_origin = self
            .sketch_preferences
            .as_deref()
            .map(|preferences| preferences.sketch_options().auto_project_origin)
            .unwrap_or_else(|| {
                oblikovati_model::application_options::SketchApplicationOptions::default()
                    .auto_project_origin
            });
        let args: CreateSketchArgs = crate::decode(method, request)?;
        let part_datums = self.sketch_creation_datums(&args)?;
        crate::sketches::create::apply(self.files, args, project_origin, part_datums.as_ref())
    }
    fn sketch_creation_datums(
        &mut self,
        args: &CreateSketchArgs,
    ) -> Result<Option<oblikovati_session::part_evaluation::CurrentPartDatums>, Error> {
        if args.work_plane_index.is_none() && args.orientation.is_none() {
            return Ok(None);
        }
        let document = crate::sketches::owner::active_sketch_host(self.files)?;
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|error| Error(error.to_string()))?
            .snapshot();
        if !crate::requires_part_replay(snapshot) {
            return Ok(None);
        }
        if let Some(index) = args.work_plane_index.as_deref() {
            let (placement, _) = crate::datums::sketch_plane_at_index(snapshot, *index)?;
            if matches!(
                placement,
                oblikovati_model::document_sketch::SketchPlacement::Fixed(_)
            ) && args.orientation.is_none()
            {
                return Ok(None);
            }
        }
        let owner: &mut dyn crate::PartUpdateAccess<O, W, F> = match self.part_update.as_deref_mut()
        {
            Some(owner) => owner,
            None => self.part_topology.as_deref_mut().ok_or_else(|| {
                Error(format!(
                    "document {document} sketch creation: expected a part replay owner"
                ))
            })?,
        };
        owner
            .current_part_datums(self.files, document)
            .map(Some)
            .map_err(Error)
    }
    pub(super) fn sketch_option_command(
        &mut self,
        method: &str,
        request: &[u8],
    ) -> Result<Vec<u8>, Error> {
        let preferences = self.sketch_preferences.as_deref_mut().ok_or_else(|| {
            Error(format!("application option call {method:?}: expected an injected application preference owner"))
        })?;
        crate::options::apply(preferences, method, request)
    }
}
