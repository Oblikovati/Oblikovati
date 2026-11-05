// SPDX-License-Identifier: GPL-2.0-only
use super::{
    CompletionCandidate, CompletionEngine, EditorCompletion, EditorPosition, ScriptEditor,
};
use crate::help::{MethodHelp, MethodHelpCatalog};
const POPUP_ROWS: usize = 8;
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum CompletionKey {
    Escape,
    Up,
    Down,
    Enter,
    Tab,
    Other,
}
/// Borrowed choices for one frame; the native view maps the source anchor to one line below the caret.
pub struct CompletionView<'frame> {
    pub candidates: &'frame [CompletionCandidate],
    pub selected: usize,
    pub first: usize,
    pub shown: usize,
    pub width_columns: usize,
    pub anchor: EditorPosition,
}
/// Headless popup controls; the editor remains the sole source/history owner.
pub struct CompletionPopup {
    engine: CompletionEngine,
    result: Option<EditorCompletion>,
    selected: usize,
    active: bool,
}
impl CompletionPopup {
    /// Inject the host method engine; example: `CompletionPopup::new(engine)`.
    pub fn new(engine: CompletionEngine) -> Self {
        Self {
            engine,
            result: None,
            selected: 0,
            active: false,
        }
    }
    /// Replace host capabilities and discard old suggestions; example: `popup.replace_engine(engine)`.
    pub fn replace_engine(&mut self, engine: CompletionEngine) {
        self.engine = engine;
        self.result = None;
        self.selected = 0;
        self.dismiss();
    }
    /// Refresh after typing, or force empty-prefix suggestions; example: `popup.refresh(&editor, true)` for Ctrl+Space.
    pub fn refresh(&mut self, editor: &ScriptEditor, trigger: bool) {
        let result = editor.complete(&self.engine);
        let total = result.candidates.len();
        if self.selected >= total {
            self.selected = 0;
        }
        let prefix = result.caret.column - result.start.column;
        self.active = (trigger || prefix > 0) && total > 0;
        self.result = Some(result);
    }
    /// Hide suggestions without editing source; example: `popup.dismiss()` on focus loss.
    pub fn dismiss(&mut self) {
        self.active = false;
    }
    /// Check source/caret freshness before drawing or consuming keys; example: `popup.visible(&editor)`.
    pub fn visible(&self, editor: &ScriptEditor) -> bool {
        self.active
            && self
                .result
                .as_ref()
                .is_some_and(|result| result.is_current(editor) && !result.candidates.is_empty())
    }
    /// Borrow layout in character cells, with at most eight rows; example: `popup.view(&editor)`.
    pub fn view(&self, editor: &ScriptEditor) -> Option<CompletionView<'_>> {
        if !self.visible(editor) {
            return None;
        }
        let result = self.result.as_ref()?;
        let width = result
            .candidates
            .iter()
            .map(|candidate| candidate.text.chars().count())
            .max()
            .unwrap_or(0);
        Some(CompletionView {
            candidates: &result.candidates,
            selected: self.selected,
            first: self.selected.saturating_sub(POPUP_ROWS - 1),
            shown: result.candidates.len().min(POPUP_ROWS),
            width_columns: width + 4,
            anchor: result.start,
        })
    }
    /// Consume popup keys; return false so normal editor keys can proceed when hidden or unrelated.
    /// Example: `popup.handle_key(&mut editor, CompletionKey::Tab)?` accepts the selected prefix.
    pub fn handle_key(
        &mut self,
        editor: &mut ScriptEditor,
        key: CompletionKey,
    ) -> Result<bool, String> {
        if !self.visible(editor) {
            return Ok(false);
        }
        match key {
            CompletionKey::Escape => self.dismiss(),
            CompletionKey::Up => self.selected = self.selected.saturating_sub(1),
            CompletionKey::Down => self.move_down(),
            CompletionKey::Enter | CompletionKey::Tab => self.accept(editor)?,
            CompletionKey::Other => return Ok(false),
        }
        Ok(true)
    }
    fn move_down(&mut self) {
        let total = self
            .result
            .as_ref()
            .map_or(0, |result| result.candidates.len());
        self.selected = (self.selected + 1).min(total.saturating_sub(1));
    }
    /// Accept only a current visible selection and close the popup; example: `popup.accept(&mut editor)?`.
    pub fn accept(&mut self, editor: &mut ScriptEditor) -> Result<(), String> {
        if !self.visible(editor) {
            self.dismiss();
            return Err(format!(
                "completion at {:?}: expected a visible current popup",
                editor.selection().caret
            ));
        }
        let result = self
            .result
            .as_ref()
            .expect("visible popup has current choices");
        let accepted = editor.accept_completion(result, self.selected);
        self.dismiss();
        accepted
    }
    /// Suppress call hints while suggestions overlap them; example: `popup.signature_help(&editor, &catalog)`.
    pub fn signature_help<'help>(
        &self,
        editor: &ScriptEditor,
        catalog: &'help MethodHelpCatalog,
    ) -> Option<&'help MethodHelp> {
        if self.visible(editor) {
            return None;
        }
        catalog.signature_help(editor)
    }
    /// Suppress hover hints while suggestions are visible; example: `popup.hover_help(&editor, position, &catalog)`.
    pub fn hover_help<'help>(
        &self,
        editor: &ScriptEditor,
        position: EditorPosition,
        catalog: &'help MethodHelpCatalog,
    ) -> Option<&'help MethodHelp> {
        if self.visible(editor) {
            return None;
        }
        catalog.hover_help(editor, position)
    }
}
