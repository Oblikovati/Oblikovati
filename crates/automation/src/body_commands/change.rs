// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) enum BodyDisplayEdit {
    Rename(BodyRenameArgs),
    Visibility(BodySetVisibleArgs),
}
impl BodyDisplayEdit {
    pub fn parse(method: &str, encoded: &[u8]) -> Result<Self, Error> {
        match method {
            METHOD_BODY_RENAME => crate::decode(method, encoded).map(Self::Rename),
            METHOD_BODY_SET_VISIBLE => crate::decode(method, encoded).map(Self::Visibility),
            _ => Err(Error(format!(
                "body display method {method:?}: expected {BODY_COMMAND_METHODS:?}"
            ))),
        }
    }
    pub fn index(&self) -> i64 {
        match self {
            Self::Rename(request) => request.body_index,
            Self::Visibility(request) => request.body_index,
        }
    }
    pub fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        body: &mut BodyInfo,
    ) -> Result<(), Error> {
        match self {
            Self::Rename(request) => apply_rename(files, document, body, request),
            Self::Visibility(request) => {
                files
                    .commands(document)
                    .map_err(|cause| Error(cause.to_string()))?
                    .set_body_visible(&body.key, request.visible)
                    .map_err(Error)?;
                body.visible = request.visible;
                Ok(())
            }
        }
    }
}
fn apply_rename<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    body: &mut BodyInfo,
    request: BodyRenameArgs,
) -> Result<(), Error> {
    store_name(files, document, &body.key, &request.name)?;
    body.name = if request.name.is_empty() {
        format!("Solid{}", request.body_index + 1)
    } else {
        request.name
    };
    Ok(())
}
