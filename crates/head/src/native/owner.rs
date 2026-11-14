// SPDX-License-Identifier: GPL-2.0-only
//! Build the native desktop owner and its script worker.
use super::{ViewportWake, preferences};

#[derive(Clone)]
pub(super) struct NativeOwnerWake(pub(super) ViewportWake);
impl oblikovati_automation::dispatch::DispatchWakeup for NativeOwnerWake {
    fn wake(&self) -> Result<(), String> {
        self.0.wake()
    }
}
pub(super) type NativeDesktopOwner = crate::DesktopOwner<
    crate::DesktopObservations,
    crate::DesktopObservations,
    oblikovati_persistence::file_access::NativeDocumentFiles,
    oblikovati_model::identity::SystemGuidSource,
    crate::DesktopObservations,
    NativeOwnerWake,
>;
#[cfg(test)]
pub(super) fn native_owner(wake: ViewportWake) -> Result<NativeDesktopOwner, String> {
    native_owner_at(wake, &preferences::native_config_directory()?)
}
pub(super) fn native_owner_at(
    wake: ViewportWake,
    directory: &std::path::Path,
) -> Result<NativeDesktopOwner, String> {
    let themes = preferences::load_native_themes(directory)?;
    let observations = crate::DesktopObservations::default();
    let files = oblikovati_session::files::FileWorkspace::new(
        observations.clone(),
        oblikovati_persistence::file_access::NativeDocumentFiles,
    );
    let version = oblikovati_cli::BuildIdentity::parse(oblikovati_cli::BUILD_MANIFEST)?.version;
    Ok(crate::DesktopOwner::new(
        files,
        oblikovati_model::identity::SystemGuidSource,
        observations,
        version,
        NativeOwnerWake(wake),
        crate::DesktopDispatchOptions::default(),
    )
    .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())?
    .with_selection_preferences(Box::new(preferences::load_native_selection_filter(
        directory,
    )?))
    .with_keymap_preferences(Box::new(preferences::load_native_keymap(directory)?))?
    .with_marking_menu_preferences(Box::new(preferences::load_native_marking_menu(directory)?))?
    .with_ui_scale_preferences(Box::new(preferences::load_native_application_options(
        directory,
    )?))
    .with_prompt_preferences(Box::new(preferences::load_native_application_options(
        directory,
    )?))
    .with_sketch_preferences(Box::new(preferences::load_native_application_options(
        directory,
    )?))
    .with_recent_documents(Box::new(preferences::load_native_application_options(
        directory,
    )?))
    .with_theme_store(themes.library, Box::new(themes.preferences))
    .with_theme_file_declines(themes.declines)
    .with_window_captures(crate::DesktopWindowCaptures::new(
        std::env::temp_dir(),
        u64::from(std::process::id()),
    )))
}
#[cfg(test)]
pub(super) type NativeScriptDesktop = crate::DesktopScripts<NativeDesktopOwner>;
#[cfg(test)]
pub(super) fn native_script_owner(wake: ViewportWake) -> Result<NativeScriptDesktop, String> {
    let owner = native_owner(wake)?;
    Ok(crate::DesktopScripts::new(
        owner,
        native_script_engine()?,
        oblikovati_script::ScriptLimits::gui(),
    ))
}
pub(super) fn native_script_engine() -> Result<oblikovati_script::IsolatedLuaEngine, String> {
    let executable = std::env::current_exe()
        .map_err(|cause| format!("desktop script worker: expected current executable: {cause}"))?;
    Ok(oblikovati_script::IsolatedLuaEngine::new(
        &executable,
        vec![oblikovati_script::WORKER_ARGUMENT.into()],
    ))
}
