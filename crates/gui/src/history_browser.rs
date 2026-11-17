// SPDX-License-Identifier: GPL-2.0-only
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_api::wire::TransactionHistory;

#[derive(Default)]
pub(crate) struct HistoryBrowser {
    open: crate::dockable_lifecycle::DockableState,
    selected: Vec<u64>,
    initialized: bool,
}
impl DockableWindow for HistoryBrowser {}
impl HistoryBrowser {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("History Browser", &self.open)
    }
    pub(crate) fn set_open(&mut self, open: bool) {
        self.open.set_open(open);
    }
    pub(crate) fn is_open(&self) -> bool {
        self.open.is_open()
    }
    pub(crate) fn open(&mut self) {
        self.open.set_open(true);
    }
    pub(crate) fn draw(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !self.open.is_open() {
            return;
        }
        synchronize_documents(&state.histories, &mut self.selected, &mut self.initialized);
        let mut open = self.open.is_open();
        Self::dockable_window(ui, "History Browser", &mut open)
            .size([560., 480.], imgui::Condition::FirstUseEver)
            .build(|| self.contents(ui, state, actions));
        self.open.set_open(open);
    }
    fn contents(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        for history in &state.histories {
            let mut shown = self.selected.contains(&history.document);
            if ui.checkbox(
                format!("{}##history-document-{}", history.name, history.document),
                &mut shown,
            ) {
                select_document(&mut self.selected, history.document, shown);
            }
        }
        ui.separator();
        for history in state
            .histories
            .iter()
            .filter(|history| self.selected.contains(&history.document))
        {
            draw_timeline(ui, history, state.transaction_open, actions);
        }
    }
}
pub(crate) fn validate_histories(histories: &[TransactionHistory]) -> Result<(), String> {
    let mut documents = std::collections::BTreeSet::new();
    for history in histories {
        let count = history.entries.as_ref().map_or(0, Vec::len);
        if history.document == 0
            || !documents.insert(history.document)
            || history.name.contains('\0')
            || usize::try_from(history.position)
                .ok()
                .is_none_or(|position| position > count)
            || history
                .entries
                .as_deref()
                .unwrap_or_default()
                .iter()
                .any(|entry| entry.label.contains('\0'))
        {
            return Err(format!(
                "history {history:?}: expected a unique nonzero document, valid cursor, and NUL-free labels"
            ));
        }
    }
    Ok(())
}
fn synchronize_documents(
    histories: &[TransactionHistory],
    selected: &mut Vec<u64>,
    initialized: &mut bool,
) {
    selected.retain(|id| histories.iter().any(|history| history.document == *id));
    if histories.is_empty() {
        *initialized = false;
        return;
    }
    if !*initialized {
        selected.push(histories[0].document);
        *initialized = true;
    }
}
fn select_document(selected: &mut Vec<u64>, document: u64, shown: bool) {
    selected.retain(|id| *id != document);
    if shown {
        selected.push(document);
    }
}
fn draw_timeline(
    ui: &Ui,
    history: &TransactionHistory,
    locked: bool,
    actions: &mut Vec<GuiShellAction>,
) {
    ui.text(&history.name);
    let disabled = ui.begin_disabled(locked);
    draw_position(ui, history, "Open", 0, actions);
    for (index, entry) in history
        .entries
        .as_deref()
        .unwrap_or_default()
        .iter()
        .enumerate()
    {
        let Ok(position) = i64::try_from(index + 1) else {
            break;
        };
        let prefix = if entry.saved { "* " } else { "  " };
        draw_position(
            ui,
            history,
            &format!("{prefix}{}", entry.label),
            position,
            actions,
        );
    }
    drop(disabled);
    ui.separator();
}
fn draw_position(
    ui: &Ui,
    history: &TransactionHistory,
    label: &str,
    position: i64,
    actions: &mut Vec<GuiShellAction>,
) {
    let clicked = ui
        .selectable_config(format!("{label}##history-{}-{position}", history.document))
        .selected(history.position == position)
        .build();
    if clicked {
        actions.push(GuiShellAction::JumpHistoryTo {
            document: history.document,
            position,
        });
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    struct HistoryPanelFixture {
        context: crate::test_context::TestContext,
        browser: HistoryBrowser,
        state: GuiCommandState,
    }
    impl HistoryPanelFixture {
        fn new() -> Self {
            let mut context = crate::test_context::create();
            context.set_ini_filename(None);
            context.io_mut().display_size = [800., 600.];
            context.io_mut().delta_time = 1. / 60.;
            context.fonts().build_rgba32_texture();
            let mut browser = HistoryBrowser::default();
            browser.open();
            let history = TransactionHistory {
                document: 3,
                name: "Part".into(),
                position: 1,
                entries: Some(vec![oblikovati_api::wire::TransactionHistoryEntry {
                    label: "Edit".into(),
                    ..Default::default()
                }]),
            };
            Self {
                context,
                browser,
                state: GuiCommandState {
                    histories: vec![history],
                    ..Default::default()
                },
            }
        }
        fn frame(&mut self, events: &[crate::GuiInputEvent]) -> Vec<GuiShellAction> {
            for event in events {
                crate::input::apply_input(self.context.io_mut(), event, [1.; 2]);
            }
            let ui = self.context.frame();
            let mut actions = Vec::new();
            self.browser.draw(ui, &self.state, &mut actions);
            self.context.render();
            actions
        }
    }
    #[test]
    fn clicking_the_open_position_requests_a_history_jump() {
        let mut fixture = HistoryPanelFixture::new();
        fixture.frame(&[]);
        use crate::{GuiInputEvent, GuiPointerButton};
        fixture.frame(&[GuiInputEvent::PointerMoved([80., 137.])]);
        fixture.frame(&[GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed: true,
        }]);
        let actions = fixture.frame(&[GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed: false,
        }]);
        assert_eq!(
            actions,
            vec![GuiShellAction::JumpHistoryTo {
                document: 3,
                position: 0
            }]
        );
    }
    #[test]
    fn history_validation_rejects_stale_cursors_duplicates_and_unsafe_labels() {
        let valid = TransactionHistory {
            document: 3,
            name: "Part".into(),
            ..Default::default()
        };
        assert!(validate_histories(std::slice::from_ref(&valid)).is_ok());
        assert!(validate_histories(&[valid.clone(), valid.clone()]).is_err());
        let mut bad = valid.clone();
        bad.position = 1;
        assert!(validate_histories(&[bad]).is_err());
        bad = valid;
        bad.name = "bad\0name".into();
        assert!(validate_histories(&[bad]).is_err());
    }
    #[test]
    fn history_selection_drops_closed_documents_and_defaults_to_an_open_one() {
        let histories = vec![TransactionHistory {
            document: 3,
            ..Default::default()
        }];
        let mut selected = vec![2];
        let mut initialized = false;
        synchronize_documents(&histories, &mut selected, &mut initialized);
        assert_eq!(selected, vec![3]);
        select_document(&mut selected, 3, false);
        synchronize_documents(&histories, &mut selected, &mut initialized);
        assert!(selected.is_empty());
        synchronize_documents(&[], &mut selected, &mut initialized);
        synchronize_documents(&histories, &mut selected, &mut initialized);
        assert_eq!(selected, vec![3]);
    }
}
