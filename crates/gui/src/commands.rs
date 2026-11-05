// SPDX-License-Identifier: GPL-2.0-only
#[derive(Clone, Debug, PartialEq)]
pub enum GuiShellAction {
    SetTheme(String),
    EditThemeColor {
        name: String,
        token: oblikovati_api::types::ThemeToken,
        value: oblikovati_api::types::Rgba,
    },
    DuplicateTheme {
        base: String,
        name: String,
    },
    DeleteTheme {
        name: String,
    },
    SaveTheme {
        name: String,
    },
    SetColorScheme(oblikovati_api::wire::SetColorSchemeArgs),
    ActivateDocument(oblikovati_api::wire::ActivateDocumentArgs),
    CloseDocument(oblikovati_api::wire::CloseDocumentArgs),
    OpenDocument(oblikovati_api::wire::OpenDocumentArgs),
    SaveDocumentAs(oblikovati_api::wire::SaveDocumentAsArgs),
    EditSketch {
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
    },
    Parameter {
        mutation: GuiParameterMutation,
        request: oblikovati_api::wire::ParameterSetArgs,
    },
    NewPart,
    StartSketchXY,
    FinishSketch,
    PlaceSketchTool(crate::GuiSketchTool),
    Save,
    Undo,
    Redo,
    Home,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiParameterMutation {
    Add,
    Set,
}
/// A bounded owner snapshot; the GUI holds no document history or transaction token.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiAuthoringContext {
    #[default]
    NoDocument,
    OtherDocument,
    Part,
    Sketch,
}
/// Command availability derived from the active document owner.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiCommandState {
    pub color_scheme: Option<crate::GuiColorSchemeState>,
    pub theme: Option<crate::GuiThemeState>,
    pub browser: Option<crate::GuiModelBrowser>,
    pub documents: Vec<oblikovati_api::wire::DocumentInfo>,
    pub document: Option<u64>,
    pub document_type: Option<oblikovati_api::types::DocumentType>,
    pub parameters: Vec<oblikovati_api::wire::ParameterInfo>,
    pub authoring: GuiAuthoringContext,
    pub undo_label: Option<String>,
    pub redo_label: Option<String>,
    pub transaction_open: bool,
}
impl GuiCommandState {
    pub(crate) fn validate(&self) -> Result<(), String> {
        crate::document_tabs::validate_documents(&self.documents)?;
        if self.browser.is_some() && self.document.is_none() {
            return Err("GUI browser document None: expected an active document identity".into());
        }
        if let Some(browser) = &self.browser {
            browser.validate()?;
        }
        for label in [&self.undo_label, &self.redo_label].into_iter().flatten() {
            if label.contains('\0') {
                return Err(format!(
                    "GUI history label {label:?}: expected text without NUL"
                ));
            }
        }
        for parameter in &self.parameters {
            validate_parameter_row(parameter)?;
        }
        Ok(())
    }
}
impl GuiShellAction {
    /// Preserve Go empty-stack enablement; example: `GuiShellAction::Undo.enabled(&state)`.
    pub fn enabled(&self, state: &GuiCommandState) -> bool {
        match self {
            Self::ActivateDocument(request) => {
                !state.transaction_open
                    && state.authoring != GuiAuthoringContext::Sketch
                    && state
                        .documents
                        .iter()
                        .any(|row| row.id == request.id && row.visible)
            }
            Self::EditThemeColor { name, .. } => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.editable() && theme.active_name() == name),
            Self::DuplicateTheme { base, .. } => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.creation_available() && theme.active_name() == base),
            Self::DeleteTheme { name } => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.deletion_available() && theme.active_name() == name),
            Self::SaveTheme { name } => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.save_available() && theme.active_name() == name),
            Self::SetColorScheme(request) => state
                .color_scheme
                .as_ref()
                .is_some_and(|palette| palette.contains(&request.name)),
            Self::SetTheme(name) => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.contains(name)),
            Self::OpenDocument(_) => !state.transaction_open,
            Self::CloseDocument(request) => {
                (state.document == Some(request.id)
                    || state.documents.iter().any(|row| row.id == request.id))
                    && !state.transaction_open
            }
            Self::SaveDocumentAs(request) => {
                state.document == Some(request.document) && !state.transaction_open
            }
            Self::EditSketch { document, sketch } => {
                state.document == Some(*document)
                    && state.authoring != GuiAuthoringContext::Sketch
                    && state
                        .browser
                        .as_ref()
                        .is_some_and(|browser| browser.sketches.iter().any(|row| row.id == *sketch))
            }
            Self::Parameter { .. } => state.document.is_some(),
            Self::Undo => state.undo_label.is_some(),
            Self::Redo => state.redo_label.is_some(),
            Self::StartSketchXY => state.authoring == GuiAuthoringContext::Part,
            Self::PlaceSketchTool(_) | Self::FinishSketch => {
                state.authoring == GuiAuthoringContext::Sketch
            }
            Self::NewPart | Self::Save | Self::Home => true,
        }
    }
    /// Keep available history controls inert during an edit; example: `action.dispatchable(&state)` after a click.
    pub fn dispatchable(&self, state: &GuiCommandState) -> bool {
        let history = matches!(self, Self::Undo | Self::Redo | Self::EditSketch { .. });
        self.enabled(state) && !(history && state.transaction_open)
    }
    pub(crate) fn next_label<'a>(&self, state: &'a GuiCommandState) -> Option<&'a str> {
        match self {
            Self::Undo => state.undo_label.as_deref(),
            Self::Redo => state.redo_label.as_deref(),
            _ => None,
        }
    }
}
fn validate_parameter_row(parameter: &oblikovati_api::wire::ParameterInfo) -> Result<(), String> {
    for value in [
        &parameter.name,
        &parameter.kind,
        &parameter.expression,
        &parameter.value,
        &parameter.health,
    ] {
        if value.contains('\0') {
            return Err(format!(
                "GUI parameter value {value:?}: expected text without NUL"
            ));
        }
    }
    Ok(())
}
#[cfg(test)]
#[path = "../tests/commands/unit.rs"]
mod tests;
