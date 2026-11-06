// SPDX-License-Identifier: GPL-2.0-only
use std::{
    io,
    path::{Path, PathBuf},
};

/// Custom file creation must never replace an existing theme, even when names collide.
pub trait ThemeLifecycleFiles {
    /// Publish complete XML at a new path; example: `files.create_theme(directory, xml)?`.
    fn create_theme(&mut self, directory: &Path, xml: &[u8]) -> io::Result<PathBuf>;
    /// Remove one known custom source; example: `files.remove_theme(source)?`.
    fn remove_theme(&mut self, source: &Path) -> io::Result<()>;
}

impl ThemeLifecycleFiles for crate::theme_preferences::NativeThemePreferenceFiles {
    fn create_theme(&mut self, directory: &Path, xml: &[u8]) -> io::Result<PathBuf> {
        use std::io::Write;
        std::fs::create_dir_all(directory)?;
        let mut stage = tempfile::Builder::new()
            .prefix(".oblikovati-theme-")
            .tempfile_in(directory)?;
        stage.write_all(xml)?;
        stage.as_file().sync_all()?;
        // The random source identity is independent of display names and their slug collisions.
        let destination = stage.path().with_extension("xml");
        stage
            .persist_noclobber(&destination)
            .map_err(|failure| failure.error)?;
        Ok(destination)
    }
    fn remove_theme(&mut self, source: &Path) -> io::Result<()> {
        std::fs::remove_file(source)
    }
}
