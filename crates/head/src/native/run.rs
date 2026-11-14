// SPDX-License-Identifier: GPL-2.0-only
//! Native desktop startup and document admission.
use super::*;

/// Open the Linux head on its owner thread; example: `run_native(HeadOptions { frames: Some(3) })?`.
pub fn run_native(options: HeadOptions) -> Result<HeadSummary, String> {
    run_native_documents(options, &[])
}
/// Open native documents before the first frame; example: `run_native_documents(options, &["part.opd".into()])?`.
pub fn run_native_documents(
    options: HeadOptions,
    documents: &[std::path::PathBuf],
) -> Result<HeadSummary, String> {
    let directory = preferences::native_config_directory()?;
    let mut window = NativeHeadWindow::open()?;
    window.ribbon_settings_directory = Some(directory.clone());
    let owner = match native_owner_at(window.wake.clone(), &directory) {
        Ok(owner) => owner,
        Err(cause) => return crate::frames::combine_retirement(Err(cause), window.finish()),
    };
    addin_launch::run_startup(&mut window, owner, options, documents)
}
