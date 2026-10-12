// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadOptions, HeadSummary, HeadWindow, ScriptDocumentOwner};
use oblikovati_api::{client::Client, wire::OpenDocumentArgs};
use std::path::{Path, PathBuf};
/// Open startup files through the canonical typed client, then run and retire the owner.
/// Example: `run_window_with_documents(window, options, owner, &["part.opd".into()])?`.
pub fn run_window_with_documents<D: ScriptDocumentOwner>(
    window: &mut dyn HeadWindow,
    options: HeadOptions,
    owner: &mut D,
    documents: &[PathBuf],
) -> Result<HeadSummary, String> {
    let outcome = open_startup_documents(owner, documents)
        .and_then(|()| crate::frames::run_frames(window, options, owner));
    crate::frames::retire_head_run(outcome, owner, window)
}
fn open_startup_documents<D: ScriptDocumentOwner>(
    owner: &mut D,
    documents: &[PathBuf],
) -> Result<(), String> {
    for path in documents {
        let request = startup_open_request(path)?;
        Client::new(owner.script_host())
            .documents()
            .open(&request)
            .map_err(|cause| {
                format!("startup document {path:?}: expected a readable native document: {cause}")
            })?;
    }
    Ok(())
}
fn startup_open_request(path: &Path) -> Result<OpenDocumentArgs, String> {
    let name = path
        .to_str()
        .filter(|value| !value.is_empty())
        .ok_or_else(|| {
            format!(
                "startup document {path:?}: expected a nonempty UTF-8 path for the document API"
            )
        })?;
    Ok(OpenDocumentArgs {
        full_document_name: name.into(),
        visible: true,
        ..Default::default()
    })
}
#[cfg(test)]
#[path = "../tests/startup.rs"]
mod tests;
