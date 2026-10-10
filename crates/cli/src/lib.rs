// SPDX-License-Identifier: GPL-2.0-only

//! Headless commands with injected file input and user-facing output.
#[path = "../build_identity.rs"]
mod build_identity;
pub use build_identity::BuildIdentity;
mod info;
use oblikovati_persistence::load::PackageFileReader;
use std::{ffi::OsString, io::Write};

/// Complete packaged provenance, or an explicitly unstamped development identity.
pub const BUILD_MANIFEST: &str = include_str!(concat!(env!("OUT_DIR"), "/build-manifest.json"));

const USAGE: &str = "oblikovati-cli — headless document tool\n\nusage:\n  oblikovati-cli info <path> [--json]\n  oblikovati-cli version\n  oblikovati-cli help\n\nRust port: info reads the package manifest only. Geometry reconstruction, edits, and scripts are not yet available.";

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
        Some("info") => info::inspect(operands, reader, output),
        Some("version" | "--version") => version(operands, output),
        Some("help" | "-h" | "--help") => write_line(output, USAGE),
        Some(
            "new" | "open" | "save-as" | "import" | "export" | "export-flat" | "script"
            | "generate-assembly" | "check-updates",
        ) => Err(format!(
            "oblikovati-cli: command {command:?} is not yet ported; available commands: info, version, help"
        )),
        _ => Err(format!(
            "oblikovati-cli: unknown command {command:?}; expected info, version, or help"
        )),
    }
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
