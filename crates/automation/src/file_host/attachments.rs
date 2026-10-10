// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{attachments, command_owner::CommandOwner};

impl<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'a, O, W, F> {
    /// Supply foreign-file access for attachment capture and freshness checks.
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
        let external = attachments::external_probe(&mut self.external);
        encode(&attachments::list(
            self.files.workspace(),
            request,
            external,
        )?)
    }

    pub(super) fn add_attachment(&mut self, request: AddAttachmentArgs) -> Result<Vec<u8>, Error> {
        let external = &mut self.external;
        let source = &mut self.source;
        let response = self
            .files
            .commands(request.document)
            .map_err(|error| Error(error.to_string()))?
            .run("Add Attachment", |snapshot, _| {
                let probe = attachments::external_probe(external);
                snapshot
                    .attachments
                    .attach(
                        &request.name,
                        request.kind,
                        &request.full_file_name,
                        probe,
                        &mut **source,
                    )
                    .map_err(|error| Error(error.to_string()))?;
                let probe = attachments::external_probe(external);
                let attachment = snapshot
                    .attachments
                    .get(&request.name)
                    .expect("captured attachment");
                encode(&attachments::info(attachment, probe)?)
            })
            .map_err(|error| Error(error.to_string()))?;
        self.files.refresh_after_command(request.document)?;
        Ok(response)
    }
}
