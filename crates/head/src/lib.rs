// SPDX-License-Identifier: GPL-2.0-only
//! Desktop window lifetime and document-owned frame dispatch. Native UI and model display remain open.
mod arguments;
pub use arguments::{HeadCommand, HeadOptions, parse_arguments};
#[cfg(target_os = "linux")]
mod frames;
#[cfg(target_os = "linux")]
pub use frames::{HeadPresentation, HeadSummary, HeadWindow, run_window, run_window_with_owner};
#[cfg(target_os = "linux")]
mod native;
#[cfg(target_os = "linux")]
pub use native::run_native;
#[cfg(target_os = "linux")]
mod work;
#[cfg(target_os = "linux")]
pub use work::{HeadInvalidation, HeadWork, HeadWorkStatus};
#[cfg(target_os = "linux")]
mod owner;
#[cfg(target_os = "linux")]
pub use owner::{DesktopDispatchOptions, DesktopOwner};
#[cfg(target_os = "linux")]
mod observations;
#[cfg(target_os = "linux")]
pub use observations::{DesktopObservations, MissingInterestNotice};
use std::{ffi::OsString, io::Write};
const USAGE: &str = "usage: oblikovati-head [--frames N]\n       oblikovati-head --help\n       oblikovati-head --version\n\n0 frames runs until the window closes. A positive count bounds a smoke run.";
/// Execute the desktop entry point with injected output; example: `run(&["--help".into()], output)?;`.
pub fn run(arguments: &[OsString], output: &mut dyn Write) -> Result<(), String> {
    match parse_arguments(arguments)? {
        HeadCommand::Help => write_output(output, USAGE),
        HeadCommand::Version => {
            let identity = oblikovati_cli::BuildIdentity::parse(oblikovati_cli::BUILD_MANIFEST)?;
            write_output(
                output,
                &format!(
                    "oblikovati-head {} (commit {})",
                    identity.version, identity.application_commit
                ),
            )
        }
        HeadCommand::Window(options) => launch(options),
    }
}
#[cfg(target_os = "linux")]
fn launch(options: HeadOptions) -> Result<(), String> {
    run_native(options).map(|_| ())
}
#[cfg(not(target_os = "linux"))]
fn launch(_: HeadOptions) -> Result<(), String> {
    Err("desktop target: expected the supported local Linux profile; other platforms remain deferred".into())
}
fn write_output(output: &mut dyn Write, text: &str) -> Result<(), String> {
    writeln!(output, "{text}")
        .map_err(|error| format!("head output: expected a writable stream: {error}"))
}
