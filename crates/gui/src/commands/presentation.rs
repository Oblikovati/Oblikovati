// SPDX-License-Identifier: GPL-2.0-only
/// A small picture of the window: enough cells for a preview area, drawn without a texture.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiPreviewImage {
    pub extent: [u32; 2],
    pub rgb: std::sync::Arc<Vec<[u8; 3]>>,
}
/// One loaded project, with the settings its dialog lists.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiProject {
    pub name: String,
    pub location: String,
    pub active: bool,
    pub settings: Vec<(String, String)>,
    /// What the tree folds away until it is asked for; example: `Folder Options` and its rows.
    pub groups: Vec<GuiProjectGroup>,
}
/// One project file request from Projects.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiProjectAction {
    Create {
        request: u64,
        workspace: std::path::PathBuf,
        name: String,
    },
    Open {
        request: u64,
        path: std::path::PathBuf,
    },
}
/// One foldable branch of a project's settings.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiProjectGroup {
    pub label: String,
    pub rows: Vec<String>,
}
/// What the owner knows about one open document's file beyond its wire row.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiDocumentStorage {
    pub path: Option<String>,
    /// Closing without saving would lose work; a new, untouched part has none to lose.
    pub unsaved_edits: bool,
}
