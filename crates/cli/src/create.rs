// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::*;
use oblikovati_model::identity::GuidSource;
use oblikovati_persistence::{file_access::DocumentFiles, package::DocumentPackage};
use oblikovati_session::files::FileWorkspace;
use std::path::PathBuf;
mod arguments;
struct CreationRequest {
    kind: DocumentType,
    kind_name: &'static str,
    path: PathBuf,
    name: String,
    seed: bool,
}
pub(super) fn execute<F: DocumentFiles>(
    args: &[OsString],
    files: F,
    source: &mut dyn GuidSource,
    output: &mut dyn Write,
) -> Result<(), String> {
    let request = arguments::parse(args)?;
    if request.seed {
        write_line(output, "note: --seed applies only to parts; ignored")?;
    }
    let version = BuildIdentity::parse(BUILD_MANIFEST)?.version;
    save_document(&request, files, source, &version)?;
    write_line(
        output,
        &format!(
            "created {} {:?} at {}",
            request.kind_name,
            request.name,
            request.path.display()
        ),
    )
}
fn save_document<F: DocumentFiles>(
    request: &CreationRequest,
    files: F,
    source: &mut dyn GuidSource,
    version: &str,
) -> Result<(), String> {
    let mut owner = FileWorkspace::new(observations::CliObservations, files);
    let package = DocumentPackage::empty(request.kind, &request.name);
    let document = owner
        .create(
            &request.path,
            package,
            observations::CliObservations,
            version,
            source,
        )
        .map_err(|error| format!("new: {error}"))?;
    owner
        .save(document.session_id, version, source)
        .map_err(|error| format!("new: save {:?}: {error}", request.path))
}
