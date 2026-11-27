// SPDX-License-Identifier: GPL-2.0-only
use super::GuiFrameEngine;
use std::path::{Path, PathBuf};

impl GuiFrameEngine {
    /// Remember a saved project template before the chooser opens.
    /// Example: `engine.remember_project_template(saved_path)`.
    pub fn remember_project_template(&mut self, path: Option<PathBuf>) {
        self.document_files
            .templates
            .remember_project_template(path);
    }

    /// Read the named selection for the next desktop session.
    /// Example: `let selection = engine.project_template_path();`.
    pub fn project_template_path(&self) -> Option<&Path> {
        self.document_files.templates.project_template_path()
    }
}
