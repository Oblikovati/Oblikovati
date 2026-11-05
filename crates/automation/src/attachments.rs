// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{Error, wire::*};
use oblikovati_model::attachment::{
    AttachmentContents, DocumentAttachments, ExternalAttachmentFiles, FileAttachment,
};
use oblikovati_persistence::package::format_attachment_time;
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};

pub(super) fn info(
    attachment: &FileAttachment,
    external: Option<&mut dyn ExternalAttachmentFiles>,
) -> Result<AttachmentInfo, Error> {
    let state = attachment
        .probe(external)
        .map_err(|error| Error(error.to_string()))?;
    let (resource, stamp) = match attachment.contents() {
        AttachmentContents::Generic => (String::new(), None),
        AttachmentContents::Linked { last_known } => (String::new(), *last_known),
        AttachmentContents::Embedded { resource_id, .. } => (resource_id.to_string(), None),
    };
    Ok(AttachmentInfo {
        name: attachment.name().into(),
        kind: attachment.kind(),
        full_file_name: attachment.full_file_name().into(),
        resolved_file_name: state.resolved_file_name.unwrap_or_default(),
        resource,
        status: state.status,
        last_known_file_time: format_attachment_time(stamp)
            .map_err(|error| Error(error.to_string()))?,
        browser_visible: attachment.browser_visible(),
    })
}

pub(super) fn list<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    request: ListAttachmentsArgs,
    mut external: Option<&mut dyn ExternalAttachmentFiles>,
) -> Result<ListAttachmentsResult, Error> {
    let session = workspace
        .session(request.document)
        .map_err(|error| Error(error.to_string()))?;
    let mut attachments = Vec::new();
    for attachment in session.snapshot().attachments.entries() {
        let probe = external_probe(&mut external);
        attachments.push(info(attachment, probe)?);
    }
    Ok(ListAttachmentsResult {
        attachments: Some(attachments),
    })
}

pub(super) fn remove(
    attachments: &mut DocumentAttachments,
    request: RemoveAttachmentArgs,
) -> Result<OKResult, Error> {
    if !attachments.remove(&request.name) {
        return Err(Error(format!(
            "document {} attachment {:?}: expected an existing attachment",
            request.document, request.name
        )));
    }
    Ok(OKResult { ok: true })
}

pub(super) fn external_probe<'borrow>(
    external: &'borrow mut Option<&mut dyn ExternalAttachmentFiles>,
) -> Option<&'borrow mut dyn ExternalAttachmentFiles> {
    match external {
        Some(files) => Some(&mut **files),
        None => None,
    }
}
