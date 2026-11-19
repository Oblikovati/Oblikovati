// SPDX-License-Identifier: GPL-2.0-only
/// What a press on a browser row other than a feature or sketch asks for.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiBrowserChoice {
    /// Copy a stored part design view.
    CopyPartView(u64),
    /// Rename an authored part design view.
    RenamePartView {
        id: u64,
        name: String,
    },
    /// Apply a part view to the active display.
    ActivatePartView(u64),
    /// Change the lock state of a part view.
    LockPartView {
        id: u64,
        locked: bool,
    },
    /// Store the current camera in the active part view.
    SavePartViewCamera(u64),
    /// Restore a view's saved camera to the viewport.
    RestorePartViewCamera(u64),
    /// Change automatic camera capture for a part view.
    PartViewAutosave {
        id: u64,
        enabled: bool,
    },
    /// Set whether a part view is published.
    PublishPartView {
        id: u64,
        published: bool,
    },
    /// Show or hide every resolved body in the active part view.
    PartViewBodyVisibility {
        id: u64,
        visible: bool,
    },
    /// Create and activate a new model-state member.
    CreateModelState,
    /// Copy a member without activating the copy.
    CopyModelState(u64),
    /// Rename one document-owned member.
    RenameModelState {
        id: u64,
        name: String,
    },
    /// Activate a document-owned model state.
    ActivateModelState(u64),
    /// Change whether a feature edit affects one member or all members.
    ToggleModelStateEditScope,
    /// No row stays chosen, as after a click on empty space.
    Nothing,
    /// Show this feature's dimensions in the view, or none.
    ShowFeatureDimensions(Option<oblikovati_model::feature::FeatureId>),
    /// Whether a sketch shows its dimensions while it is visible.
    SketchDimensionsShown {
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        shown: bool,
    },
    Root,
    /// Choose a placed component by its generation-scoped owner path.
    Occurrence(crate::GuiOccurrencePath),
    /// Change the suppression state of an assembly-owned occurrence.
    SetOccurrenceSuppressed {
        path: crate::GuiOccurrencePath,
        suppressed: bool,
    },
    Body {
        key: String,
        feature: oblikovati_model::feature::FeatureId,
        toggle: bool,
    },
    BodyVisible {
        key: String,
        visible: bool,
    },
}
