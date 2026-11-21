// SPDX-License-Identifier: GPL-2.0-only
//! Explain Bend Part prerequisites without claiming the feature is available (#4109).
use crate::{GuiAuthoringContext, GuiCommandState};
use imgui::Ui;

const TITLE: &str = "Bend Part##prerequisite";
const NEED_SKETCH: &str = "A visible unadaptive sketch is needed to bend the part.";
const UNAVAILABLE: &str = "Bend Part is not available in this build.";

#[derive(Clone, Copy, Debug, Default)]
pub(super) struct BendNotice {
    message: Option<&'static str>,
    opening: bool,
}

impl BendNotice {
    pub(super) fn available(state: &GuiCommandState) -> bool {
        state.document.is_some()
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open
            && state.browser.is_some()
    }

    pub(super) fn request(&mut self, state: &GuiCommandState) {
        let Some(browser) = state.browser.as_ref() else {
            return;
        };
        self.message = Some(if browser.sketches.iter().any(|sketch| sketch.visible) {
            UNAVAILABLE
        } else {
            NEED_SKETCH
        });
        self.opening = true;
    }

    pub(super) fn active(&self) -> bool {
        self.message.is_some()
    }

    pub(super) fn draw(&mut self, ui: &Ui) -> bool {
        if std::mem::take(&mut self.opening) {
            ui.open_popup(TITLE);
        }
        let Some(message) = self.message else {
            return false;
        };
        let size = [430., 130.];
        let display = ui.io().display_size;
        let origin = std::array::from_fn(|axis| ((display[axis] - size[axis]) * 0.5).max(0.));
        crate::widget_abi::fit_dialog(ui, TITLE, origin, size);
        let Some(_notice) = ui.begin_modal_popup(TITLE) else {
            return false;
        };
        ui.text_wrapped(message);
        if !ui.button("OK") && !ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
            return false;
        }
        ui.close_current_popup();
        self.message = None;
        true
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{GuiBrowserSketch, GuiModelBrowser};
    use oblikovati_model::document_sketch::DocumentSketchId;

    fn state() -> GuiCommandState {
        GuiCommandState {
            document: Some(1),
            authoring: GuiAuthoringContext::Part,
            browser: Some(GuiModelBrowser::default()),
            ..Default::default()
        }
    }
    #[test]
    fn empty_part_reports_the_missing_prerequisite() {
        let state = state();
        assert!(BendNotice::available(&state));
        let mut notice = BendNotice::default();
        notice.request(&state);
        assert_eq!(notice.message, Some(NEED_SKETCH));
        assert!(notice.active());
    }
    #[test]
    fn visible_sketch_does_not_claim_that_it_is_unsuitable() {
        let mut state = state();
        state
            .browser
            .as_mut()
            .unwrap()
            .sketches
            .push(GuiBrowserSketch {
                id: DocumentSketchId(1),
                inspection: Err("not inspected".into()),
                name: "Sketch1".into(),
                visible: true,
                consumed_by: None,
                line_style: Default::default(),
                dimensions_shown: false,
            });
        let mut notice = BendNotice::default();
        notice.request(&state);
        assert_eq!(notice.message, Some(UNAVAILABLE));
        assert!(!BendNotice::available(&GuiCommandState::default()));
        state.transaction_open = true;
        assert!(!BendNotice::available(&state));
    }
}
