// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::addin_discovery::{NativeAddInDirectoryReader, addin_directory, discover_addins};
use crate::{HeadWork, NativeAddInDesktop, ScriptDocumentOwner};
use std::path::PathBuf;
pub(super) fn run_startup(
    window: &mut NativeHeadWindow,
    mut owner: NativeDesktopOwner,
    options: HeadOptions,
    documents: &[PathBuf],
) -> Result<HeadSummary, String> {
    let profile = startup_libraries(window)
        .and_then(|libraries| native_script_engine().map(|engine| (libraries, engine)));
    let (libraries, engine) = match profile {
        Ok(profile) => profile,
        Err(cause) => return crate::frames::retire_head_run(Err(cause), &mut owner, window),
    };
    if libraries.is_empty() {
        let mut scripts =
            crate::DesktopScripts::new(owner, engine, oblikovati_script::ScriptLimits::gui());
        return crate::run_window_with_documents(window, options, &mut scripts, documents);
    }
    let outcome = run_addin_frames(window, &mut owner, options, documents, libraries, engine);
    crate::frames::retire_head_run(outcome, &mut owner, window)
}
fn startup_libraries(window: &mut NativeHeadWindow) -> Result<Vec<PathBuf>, String> {
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
    Ok(libraries)
}
fn run_addin_frames<D: ScriptDocumentOwner>(
    window: &mut NativeHeadWindow,
    owner: &mut D,
    options: HeadOptions,
    documents: &[PathBuf],
    libraries: Vec<PathBuf>,
    engine: oblikovati_script::IsolatedLuaEngine,
) -> Result<HeadSummary, String> {
    let mut services = NativeAddInDesktop::new(
        owner,
        Default::default(),
        NativeOwnerWake(window.wake.clone()),
    )?;
    // The application trusts extensions in its selected add-in directory, as in Go startup.
    let startup = unsafe { services.begin_startup(libraries) };
    let mut scripts = native_catalog_console(services, engine);
    let outcome = startup
        .and_then(|()| crate::startup::open_startup_documents(&mut scripts, documents))
        .and_then(|()| crate::frames::run_frames(window, options, &mut scripts));
    crate::frames::combine_retirement(outcome, scripts.finish())
}

fn native_catalog_console<D: ScriptDocumentOwner>(
    services: D,
    engine: oblikovati_script::IsolatedLuaEngine,
) -> crate::DesktopScripts<D> {
    crate::DesktopScripts::with_methods(
        services,
        engine,
        oblikovati_script::ScriptLimits::gui(),
        native_catalog_methods(),
    )
}

pub(super) fn native_catalog_methods() -> Vec<String> {
    let mut methods = oblikovati_automation::file_host_methods();
    methods.extend([
        oblikovati_api::wire::METHOD_ADD_INS_LIST.into(),
        oblikovati_api::wire::METHOD_ADD_INS_GET.into(),
    ]);
    methods.sort();
    methods.dedup();
    methods
}

#[cfg(test)]
mod tests {
    #[test]
    fn native_console_inventory_adds_only_read_only_catalog_methods() {
        let mut expected = oblikovati_automation::file_host_methods();
        expected.extend([
            oblikovati_api::wire::METHOD_ADD_INS_LIST.into(),
            oblikovati_api::wire::METHOD_ADD_INS_GET.into(),
        ]);
        expected.sort();
        expected.dedup();
        assert_eq!(super::native_catalog_methods(), expected);
    }
}
