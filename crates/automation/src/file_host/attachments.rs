// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{attachments, command_owner::CommandOwner};

impl<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'a, O, W, F> {
    /// Supply foreign-file access for attachment capture and freshness checks.
    /// When a project is selected, its search roots wrap this provider.
    /// Example: `FileHost::new(files, guids, "alpha").with_attachment_files(external)`.
    pub fn with_attachment_files(mut self, external: &'a mut dyn ExternalAttachmentFiles) -> Self {
        self.context.get_mut().external = Some(external);
        self
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn list_attachments(
        &mut self,
        request: ListAttachmentsArgs,
    ) -> Result<Vec<u8>, Error> {
        let lookup = self
            .files
            .active_project()
            .map(|project| &project.bound.lookup);
        let external = attachments::external_probe(&mut self.external);
        with_project_files(lookup, external, |probe| {
            encode(&attachments::list(self.files.workspace(), request, probe)?)
        })
    }

    pub(super) fn add_attachment(&mut self, request: AddAttachmentArgs) -> Result<Vec<u8>, Error> {
        let lookup = self
            .files
            .active_project()
            .map(|project| project.bound.lookup.clone());
        let external = attachments::external_probe(&mut self.external);
        let source = &mut self.source;
        let mut commands = self
            .files
            .commands(request.document)
            .map_err(|error| Error(error.to_string()))?;
        let response = with_project_files(lookup.as_ref(), external, |mut probe| {
            commands
                .run("Add Attachment", |snapshot, _| {
                    capture_attachment(snapshot, &request, &mut **source, &mut probe)
                })
                .map_err(|error| Error(error.to_string()))
        })?;
        self.files.refresh_after_command(request.document)?;
        Ok(response)
    }
}

fn capture_attachment(
    snapshot: &mut oblikovati_model::history::DocumentSnapshot,
    request: &AddAttachmentArgs,
    source: &mut dyn GuidSource,
    external: &mut Option<&mut dyn ExternalAttachmentFiles>,
) -> Result<Vec<u8>, Error> {
    snapshot
        .attachments
        .attach(
            &request.name,
            request.kind,
            &request.full_file_name,
            attachments::external_probe(external),
            source,
        )
        .map_err(|error| Error(error.to_string()))?;
    let attachment = snapshot
        .attachments
        .get(&request.name)
        .expect("captured attachment");
    encode(&attachments::info(
        attachment,
        attachments::external_probe(external),
    )?)
}

fn with_project_files<R>(
    lookup: Option<&oblikovati_persistence::project::ProjectFileLookup>,
    external: Option<&mut dyn ExternalAttachmentFiles>,
    action: impl FnOnce(Option<&mut dyn ExternalAttachmentFiles>) -> R,
) -> R {
    match (lookup, external) {
        (Some(lookup), Some(files)) => {
            let mut project =
                oblikovati_persistence::project::ProjectAttachmentFiles::new(lookup, files);
            action(Some(&mut project))
        }
        (_, files) => action(files),
    }
}
