// SPDX-License-Identifier: GPL-2.0-only
//! Desktop window lifetime and document-owned frame dispatch. Native UI and model display remain open.
#[cfg(all(test, any(target_os = "linux", target_os = "windows")))]
#[macro_use]
#[path = "../tests/document_fixture_path.rs"]
mod document_fixture_path;
#[cfg(any(target_os = "linux", target_os = "windows", test))]
mod accessibility;
#[cfg(any(
    target_os = "linux",
    target_os = "windows",
    target_arch = "wasm32",
    test
))]
mod assembly_producer;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod clock;
#[cfg(any(target_os = "linux", target_os = "windows"))]
#[allow(unsafe_code)]
mod native_services;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod part_view_display;
#[cfg(all(test, target_os = "linux"))]
#[path = "../../gui/src/test_context.rs"]
mod test_context;
#[cfg(all(test, target_os = "windows"))]
#[path = "../../gui/src/test_context_lease.rs"]
mod test_context;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod view_orientation;
#[cfg(any(target_os = "linux", target_os = "windows"))]
pub use native_services::NativeAddInDesktop;
#[cfg(any(target_os = "linux", target_os = "windows"))]
mod addin_discovery;
mod arguments;
pub use arguments::{HeadCommand, HeadOptions, parse_arguments};
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod camera_automation;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod frames;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod viewport_automation;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod window_caller;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use frames::{
    HeadGuiUpdate, HeadOrientationFitFrame, HeadPresentation, HeadSummary, HeadWindow, run_window,
    run_window_with_owner,
};
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod native;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod navigation;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use native::NativeHeadWindow;
#[cfg(any(target_os = "linux", target_os = "windows"))]
pub use native::{run_native, run_native_documents};
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod startup;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use startup::run_window_with_documents;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod capture_png;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod window_capture;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
#[macro_use]
#[path = "work_orientation.rs"]
mod orientation_work;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
#[macro_use]
mod work_datum;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_blend;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_drawing;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_hole;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_inference_scope;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_move_body;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_part_view;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_project;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_shell_feature;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_sweep;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
#[macro_use]
mod work_metadata;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
#[macro_use]
mod work_appearance;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
#[macro_use]
mod work_selection;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_extrusion;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_forwarding;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
#[path = "work_idle.rs"]
mod work_idle;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_inspect;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_sketch;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_sketch_forwarding;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_status;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod work_theme;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use capture_png::{NativeWindowCaptureFiles, WindowCaptureFiles};
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use window_capture::{DesktopWindowCaptures, WindowCaptureJob};
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use work::HeadWork;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use work_sketch::HeadSketchWork;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use work_status::{HeadInvalidation, HeadWorkStatus};
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod owner;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod selection_preferences;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use selection_preferences::SelectionPreferenceAccess;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod part_scene;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use owner::{DesktopCommandNotice, DesktopDispatchOptions, DesktopHistoryAction, DesktopOwner};
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use part_scene::{HeadSceneStatus, HeadSceneUpdate};
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod observations;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use observations::{DesktopObservations, MissingInterestNotice};
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod scripts;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use scripts::{DesktopScripts, ScriptDocumentOwner};
#[cfg(target_arch = "wasm32")]
pub mod web_host;
use std::{ffi::OsString, io::Write};
const USAGE: &str = "usage: oblikovati [--frames N] [--] [FILE ...]\n       oblikovati --help\n       oblikovati --version\n\nOpen native documents and .opj projects. The first project is active.\n0 frames runs until the window closes. A positive count bounds a smoke run.";
/// Execute the desktop entry point with injected output; example: `run(&["--help".into()], output)?;`.
pub fn run(arguments: &[OsString], output: &mut dyn Write) -> Result<(), String> {
    match parse_arguments(arguments)? {
        HeadCommand::Help => write_output(output, USAGE),
        HeadCommand::Version => {
            let identity = oblikovati_cli::BuildIdentity::parse(oblikovati_cli::BUILD_MANIFEST)?;
            write_output(
                output,
                &format!(
                    "oblikovati {} (commit {})",
                    identity.version, identity.application_commit
                ),
            )
        }
        HeadCommand::Window(options) => launch_documents(options, &[]),
        HeadCommand::OpenDocuments { options, documents } => launch_documents(options, &documents),
    }
}
#[cfg(any(target_os = "linux", target_os = "windows"))]
fn launch_documents(options: HeadOptions, documents: &[std::path::PathBuf]) -> Result<(), String> {
    run_native_documents(options, documents).map(|_| ())
}
#[cfg(not(any(target_os = "linux", target_os = "windows")))]
fn launch_documents(_: HeadOptions, _: &[std::path::PathBuf]) -> Result<(), String> {
    Err("desktop target: expected a Windows or Linux desktop".into())
}
fn write_output(output: &mut dyn Write, text: &str) -> Result<(), String> {
    writeln!(output, "{text}")
        .map_err(|error| format!("head output: expected a writable stream: {error}"))
}

#[cfg(test)]
#[path = "../tests/fixture_color.rs"]
mod fixture_color;

#[cfg(all(test, any(target_os = "linux", target_os = "windows")))]
#[path = "../tests/drawing_fixture_replay.rs"]
mod drawing_fixture_replay;

#[cfg(test)]
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod scene_fixture;

#[cfg(any(
    target_os = "linux",
    target_os = "windows",
    target_arch = "wasm32",
    test
))]
#[path = "placement/input.rs"]
mod placement_input;

#[cfg(test)]
#[path = "../tests/placement_inputs.rs"]
mod placement_test_inputs;
#[cfg(all(test, any(target_os = "linux", target_os = "windows")))]
#[path = "../tests/placement_samples.rs"]
mod placement_test_samples;

mod sketch_edit;
pub use sketch_edit::HeadSketchEdit;

#[path = "placement/tool.rs"]
mod placement_tool;
pub use placement_tool::SketchPlacementTool;

mod datum_scene;
pub use datum_scene::HeadDatumPlaneUpdate;
mod sketch_scene;
pub use sketch_scene::HeadSketchSceneUpdate;

// #3795: these private states require the DesktopOwner and placement runtime.
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod sketch_geometric;
/// Ids of extra document windows, which never collide with a document's own id.
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub(crate) const FIRST_EXTRA_WINDOW: u64 = 1 << 40;
#[cfg(any(
    target_os = "linux",
    target_os = "windows",
    target_arch = "wasm32",
    test
))]
#[path = "sketch/mod.rs"]
mod sketch_inspect;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod sketch_tool;

#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod sketch_dimension;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
mod sketch_operands;

mod dimension_annotations;
pub use dimension_annotations::{HeadDimensionAnnotation, HeadDimensionAnnotations};
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub use owner::owner_precise_input::HeadConstraintMark;
