// SPDX-License-Identifier: GPL-2.0-only
//! Desktop window lifetime and document-owned frame dispatch. Native UI and model display remain open.
#[cfg(target_os = "linux")]
#[allow(unsafe_code)]
mod native_services;
#[cfg(target_os = "linux")]
pub use native_services::NativeAddInDesktop;
#[cfg(target_os = "linux")]
mod addin_discovery;
mod arguments;
pub use arguments::{HeadCommand, HeadOptions, parse_arguments};
#[cfg(target_os = "linux")]
mod frames;
#[cfg(target_os = "linux")]
pub use frames::{HeadPresentation, HeadSummary, HeadWindow, run_window, run_window_with_owner};
#[cfg(target_os = "linux")]
mod native;
#[cfg(target_os = "linux")]
mod navigation;
#[cfg(target_os = "linux")]
pub use native::{NativeHeadWindow, run_native, run_native_documents};
#[cfg(target_os = "linux")]
mod startup;
#[cfg(target_os = "linux")]
pub use startup::run_window_with_documents;
#[cfg(target_os = "linux")]
mod work;
#[cfg(target_os = "linux")]
pub use work::{HeadInvalidation, HeadWork, HeadWorkStatus};
#[cfg(target_os = "linux")]
mod owner;
#[cfg(target_os = "linux")]
mod part_scene;
#[cfg(target_os = "linux")]
pub use owner::{DesktopCommandNotice, DesktopDispatchOptions, DesktopHistoryAction, DesktopOwner};
#[cfg(target_os = "linux")]
pub use part_scene::{HeadSceneStatus, HeadSceneUpdate};
#[cfg(target_os = "linux")]
mod observations;
#[cfg(target_os = "linux")]
pub use observations::{DesktopObservations, MissingInterestNotice};
#[cfg(target_os = "linux")]
mod scripts;
#[cfg(target_os = "linux")]
pub use scripts::{DesktopScripts, ScriptDocumentOwner};
use std::{ffi::OsString, io::Write};
const USAGE: &str = "usage: oblikovati-head [--frames N] [--] [DOCUMENT ...]\n       oblikovati-head --help\n       oblikovati-head --version\n\n0 frames runs until the window closes. A positive count bounds a smoke run.";
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
        HeadCommand::Window(options) => launch_documents(options, &[]),
        HeadCommand::OpenDocuments { options, documents } => launch_documents(options, &documents),
    }
}
#[cfg(target_os = "linux")]
fn launch_documents(options: HeadOptions, documents: &[std::path::PathBuf]) -> Result<(), String> {
    run_native_documents(options, documents).map(|_| ())
}
#[cfg(not(target_os = "linux"))]
fn launch_documents(_: HeadOptions, _: &[std::path::PathBuf]) -> Result<(), String> {
    Err("desktop target: expected the supported local Linux profile; other platforms remain deferred".into())
}
fn write_output(output: &mut dyn Write, text: &str) -> Result<(), String> {
    writeln!(output, "{text}")
        .map_err(|error| format!("head output: expected a writable stream: {error}"))
}

#[cfg(test)]
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod scene_fixture;

#[path = "placement/input.rs"]
mod placement_input;

#[cfg(test)]
#[path = "../tests/placement_inputs.rs"]
mod placement_test_inputs;
#[cfg(test)]
#[path = "../tests/placement_samples.rs"]
mod placement_test_samples;

mod sketch_edit;
pub use sketch_edit::HeadSketchEdit;

#[path = "placement/tool.rs"]
mod placement_tool;
pub use placement_tool::SketchPlacementTool;

mod sketch_scene;
pub use sketch_scene::HeadSketchSceneUpdate;
