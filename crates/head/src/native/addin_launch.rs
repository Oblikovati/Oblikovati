// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::addin_discovery::{NativeAddInDirectoryReader, addin_directory, discover_addins};
use crate::{HeadWork, NativeAddInDesktop, ScriptDocumentOwner};
use std::path::PathBuf;
pub(super) fn run_startup(
    window: &mut NativeHeadWindow,
    owner: &mut NativeScriptDesktop,
    options: HeadOptions,
    documents: &[PathBuf],
) -> Result<HeadSummary, String> {
    let override_path = std::env::var_os("OBK_ADDINS_DIR");
    let directory = addin_directory(override_path.as_deref(), std::env::current_exe());
    let libraries = match discover_addins(&NativeAddInDirectoryReader, &directory) {
        Ok(libraries) => libraries,
        Err(cause) => {
            window.show_command_notice(crate::DesktopCommandNotice::AddInStartupFailed {
                library: directory,
                cause,
            })?;
            Vec::new()
        }
    };
    if libraries.is_empty() {
        return crate::run_window_with_documents(window, options, owner, documents);
    }
    let outcome = run_addin_frames(window, owner, options, documents, libraries);
    crate::frames::retire_head_run(outcome, owner, window)
}
fn run_addin_frames<D: ScriptDocumentOwner>(
    window: &mut NativeHeadWindow,
    owner: &mut D,
    options: HeadOptions,
    documents: &[PathBuf],
    libraries: Vec<PathBuf>,
) -> Result<HeadSummary, String> {
    let mut services = NativeAddInDesktop::new(
        owner,
        Default::default(),
        NativeOwnerWake(window.wake.clone()),
    )?;
    // The application trusts extensions in its selected add-in directory, as in Go startup.
    let outcome = unsafe { services.begin_startup(libraries) }
        .and_then(|()| crate::startup::open_startup_documents(&mut services, documents))
        .and_then(|()| crate::frames::run_frames(window, options, &mut services));
    crate::frames::combine_retirement(outcome, services.finish())
}
