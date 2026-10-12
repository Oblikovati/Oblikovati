// SPDX-License-Identifier: GPL-2.0-only

//! Headless commands with injected file input and user-facing output.
#[path = "../build_identity.rs"]
mod build_identity;
pub use build_identity::BuildIdentity;
mod create;
mod documents;
mod info;
mod observations;
mod paths;
mod scripts;
use oblikovati_persistence::load::PackageFileReader;
pub use scripts::{NativeScriptSources, ScriptSources, run_script_with_files};
use std::{ffi::OsString, io::Write};

/// Complete packaged provenance, or an explicitly unstamped development identity.
pub const BUILD_MANIFEST: &str = include_str!(concat!(env!("OUT_DIR"), "/build-manifest.json"));

const USAGE: &str = "oblikovati-cli — headless document tool\n\nusage:\n  oblikovati-cli new <type> <path> [--seed]\n  oblikovati-cli info <path> [--json]\n  oblikovati-cli open <path>\n  oblikovati-cli save-as <src> <dst>\n  oblikovati-cli script run <file.lua> [--doc <path>] [--save <path>]\n  oblikovati-cli version\n  oblikovati-cli help\n\nRust port: info reads the package manifest only. Unseeded creation and recipe open/save-as are available. Kernel geometry replay and import/export are not yet available.";

/// Execute one command without exiting the process; native paths stay as OS strings.
/// Example: `run(&["info".into(), path.into()], reader, output)?;`.
pub fn run(
    arguments: &[OsString],
    reader: &mut dyn PackageFileReader,
    output: &mut dyn Write,
) -> Result<(), String> {
    let Some((command, operands)) = arguments.split_first() else {
        return Err(format!("oblikovati-cli: no command\n{USAGE}"));
    };
    match command.to_str() {
        Some("new" | "open" | "save-as" | "script") => owned_access_required(command),
        Some("info") => info::inspect(operands, reader, output),
        Some("version" | "--version") => version(operands, output),
        Some("help" | "-h" | "--help") => write_line(output, USAGE),
        Some("import" | "export" | "export-flat" | "generate-assembly" | "check-updates") => {
            Err(format!(
                "oblikovati-cli: command {command:?} is not yet ported; available commands: new, open, save-as, script, info, version, help"
            ))
        }
        _ => Err(format!(
            "oblikovati-cli: unknown command {command:?}; expected new, open, save-as, script, info, version, or help"
        )),
    }
}

fn owned_access_required(command: &OsString) -> Result<(), String> {
    Err(format!("{command:?}: expected file-owned document access"))
}

fn version(operands: &[OsString], output: &mut dyn Write) -> Result<(), String> {
    if !operands.is_empty() {
        return Err(format!("version: expected no arguments, got {operands:?}"));
    }
    let identity = BuildIdentity::parse(BUILD_MANIFEST)?;
    write_line(
        output,
        &format!(
            "oblikovati-cli {} (commit {})",
            identity.version, identity.application_commit
        ),
    )
}

fn write_line(output: &mut dyn Write, text: &str) -> Result<(), String> {
    writeln!(output, "{text}")
        .map_err(|error| format!("CLI output: expected a writable stream: {error}"))
}

/// Execute native-path commands with injected storage and identity generation.
/// Example: `run_with_files(&args, files, &mut guids, output)?` creates and saves through the document owner.
pub fn run_with_files<F: oblikovati_persistence::file_access::DocumentFiles>(
    arguments: &[OsString],
    mut files: F,
    source: &mut dyn oblikovati_model::identity::GuidSource,
    output: &mut dyn Write,
) -> Result<(), String> {
    if let Some((command, operands)) = arguments.split_first() {
        match command.to_str() {
            Some("script") => return scripts::run_native(operands, files, source, output),
            Some("new") => return create::execute(operands, files, source, output),
            Some(command @ ("open" | "save-as")) => {
                return documents::execute(command, operands, files, source, output);
            }
            _ => {}
        }
    }
    run(arguments, &mut files, output)
}
