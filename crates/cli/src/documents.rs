// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::identity::GuidSource;
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::files::FileWorkspace;
use std::path::Path;

type CliFileWorkspace<F> =
    FileWorkspace<observations::CliObservations, observations::CliObservations, F>;

pub(super) fn execute<F: DocumentFiles>(
    command: &str,
    args: &[OsString],
    files: F,
    source: &mut dyn GuidSource,
    output: &mut dyn Write,
) -> Result<(), String> {
    let required = if command == "open" { 1 } else { 2 };
    if args.len() != required {
        return Err(format!(
            "{command}: expected {required} path operand(s), got {args:?}"
        ));
    }
    let version = BuildIdentity::parse(BUILD_MANIFEST)?.version;
    let mut owner = FileWorkspace::new(observations::CliObservations, files);
    let document = load(&mut owner, Path::new(&args[0]), &version, source, command)?;
    if command == "save-as" {
        return save_as(&mut owner, document, args, &version, source, output);
    }
    report_open(&owner, document, &args[0], output)
}
fn load<F: DocumentFiles>(
    owner: &mut CliFileWorkspace<F>,
    path: &Path,
    version: &str,
    source: &mut dyn GuidSource,
    command: &str,
) -> Result<u64, String> {
    let mut factory = observations::CliObservationFactory;
    let projection = oblikovati_drawing::DrawingProjectionService;
    owned_host::host(owner, source, version, &mut factory, &projection)
        .open_path(path, true)
        .map_err(|error| format!("{command}: open {path:?}: {error}"))
}
fn report_open<F: DocumentFiles>(
    owner: &CliFileWorkspace<F>,
    document: u64,
    requested: &OsString,
    output: &mut dyn Write,
) -> Result<(), String> {
    let info = owner
        .document_info(document)
        .map_err(|error| format!("open: {error}"))?;
    write_line(
        output,
        &format!(
            "opened {} {:?} from {}",
            info.type_,
            info.name,
            Path::new(requested).display()
        ),
    )
}
fn save_as<F: DocumentFiles>(
    owner: &mut CliFileWorkspace<F>,
    document: u64,
    args: &[OsString],
    version: &str,
    source: &mut dyn GuidSource,
    output: &mut dyn Write,
) -> Result<(), String> {
    let kind = owner
        .document_type(document)
        .map_err(|error| format!("save-as: {error}"))?;
    let extension = paths::kind_extension(kind)?;
    let destination = paths::with_extension(&args[1], extension);
    let mut factory = observations::CliObservationFactory;
    let projection = oblikovati_drawing::DrawingProjectionService;
    let warnings = owned_host::host(owner, source, version, &mut factory, &projection)
        .save_as_path(document, &destination)
        .map_err(|error| format!("save-as: {error}"))?;
    for warning in warnings {
        write_line(output, &format!("Save warning: {}", warning.detail))?;
    }
    report_saved(owner, document, &destination, output)
}
fn report_saved<F: DocumentFiles>(
    owner: &CliFileWorkspace<F>,
    document: u64,
    destination: &Path,
    output: &mut dyn Write,
) -> Result<(), String> {
    let info = owner
        .document_info(document)
        .map_err(|error| format!("save-as: {error}"))?;
    write_line(
        output,
        &format!("saved {:?} as {}", info.name, destination.display()),
    )
}
