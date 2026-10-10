// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn capture(
    kind: AttachmentKind,
    path: &str,
    files: Option<&mut dyn ExternalAttachmentFiles>,
    source: &mut dyn GuidSource,
) -> Result<AttachmentContents, AttachmentError> {
    match kind {
        ATTACHMENT_GENERIC => Ok(AttachmentContents::Generic),
        ATTACHMENT_LINKED => Ok(AttachmentContents::Linked {
            last_known: match files {
                Some(files) => files
                    .inspect_source(path)
                    .map(|source| source.map(|source| source.modified))
                    .map_err(|cause| file_error(path, cause))?,
                None => None,
            },
        }),
        ATTACHMENT_EMBEDDED => embed(path, files, source),
        other => Err(AttachmentError::Invalid(format!(
            "attachment kind {}: expected generic, linked, or embedded",
            other.0
        ))),
    }
}

fn embed(
    path: &str,
    files: Option<&mut dyn ExternalAttachmentFiles>,
    source: &mut dyn GuidSource,
) -> Result<AttachmentContents, AttachmentError> {
    let files = files.ok_or_else(|| {
        AttachmentError::Invalid(format!(
            "embedded attachment {path:?}: expected an external-file provider"
        ))
    })?;
    let bytes = files.read(path).map_err(|cause| file_error(path, cause))?;
    let resource_id = source.generate().map_err(AttachmentError::Identity)?;
    Ok(AttachmentContents::Embedded { resource_id, bytes })
}
