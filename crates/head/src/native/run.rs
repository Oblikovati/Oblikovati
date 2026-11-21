// SPDX-License-Identifier: GPL-2.0-only
//! Native desktop startup and document admission.
use super::*;

/// Open the desktop head on its owner thread; example: `run_native(HeadOptions { frames: Some(3) })?`.
pub fn run_native(options: HeadOptions) -> Result<HeadSummary, String> {
    run_native_documents(options, &[])
}
/// Load projects before opening documents. Example: `run_native_documents(options, &["shop.opj".into(), "part.opd".into()])?`.
pub fn run_native_documents(
    options: HeadOptions,
    documents: &[std::path::PathBuf],
) -> Result<HeadSummary, String> {
    let directory = preferences::native_config_directory()?;
    let mut window = NativeHeadWindow::open()?;
    window.ribbon_settings_directory = Some(directory.clone());
    let mut owner = match native_owner_at(window.wake.clone(), &directory) {
        Ok(owner) => owner,
        Err(cause) => return crate::frames::combine_retirement(Err(cause), window.finish()),
    };
    let (projects, documents): (Vec<_>, Vec<_>) = documents
        .iter()
        .cloned()
        .partition(|path| is_native_project_path(path));
    for path in projects {
        if let Err(cause) = owner.load_desktop_project(&path) {
            return crate::frames::retire_head_run(Err(cause), &mut owner, &mut window);
        }
    }
    addin_launch::run_startup(&mut window, owner, options, &documents)
}

fn is_native_project_path(path: &std::path::Path) -> bool {
    let expected = oblikovati_api::types::PROJECT_FILE_EXTENSION.trim_start_matches('.');
    path.extension()
        .is_some_and(|extension| extension.eq_ignore_ascii_case(expected))
}

#[cfg(test)]
mod tests {
    use super::is_native_project_path;
    use std::path::Path;

    #[test]
    fn desktop_distinguishes_native_projects_from_documents() {
        for name in ["machine.opj", "shop.OPJ"] {
            assert!(is_native_project_path(Path::new(name)));
        }
        for name in ["part.opd", "machine.opj.opd", "shop"] {
            assert!(!is_native_project_path(Path::new(name)));
        }
    }
}
