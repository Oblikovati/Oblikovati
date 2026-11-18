// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard navigation state shared by the ribbon and its controls.
use crate::{GuiInputEvent, GuiKey, ribbon::RibbonTab};

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
enum Level {
    #[default]
    Off,
    Tabs,
    Commands,
}
#[derive(Default)]
pub(crate) struct RibbonKeyTips {
    level: Level,
    prefix: String,
    pub consumed: bool,
    expected_text: Option<char>,
    command_tab: Option<RibbonTab>,
    tools_face_mode: bool,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum KeyTipAction {
    SelectTab(RibbonTab),
    CycleDisplay,
    OpenDisplayMenu,
    OpenFileMenu,
    QuickAccess(usize),
    StartSketch,
    Extrude,
    Hole,
    ApplicationOptions,
    DocumentSettings,
    OpenGroup(crate::ribbon_panels::RibbonPanel),
}
impl RibbonKeyTips {
    pub(crate) fn set_tools_face_mode(&mut self, enabled: bool) {
        self.tools_face_mode = enabled;
    }
    pub(crate) fn tools_face_tip(&self, tip: &str) -> bool {
        self.tools_face_mode && self.tabs() && self.matching(tip)
    }
    pub(crate) fn active(&self) -> bool {
        self.level != Level::Off
    }
    pub(crate) fn tabs(&self) -> bool {
        self.level == Level::Tabs
    }
    pub(crate) fn commands(&self) -> bool {
        self.level == Level::Commands
    }
    pub(crate) fn model_commands(&self) -> bool {
        self.commands() && self.command_tab == Some(RibbonTab::Model)
    }
    pub(crate) fn matching(&self, tip: &str) -> bool {
        self.active() && tip.starts_with(&self.prefix)
    }
    pub(crate) fn dismiss(&mut self) {
        self.level = Level::Off;
        self.prefix.clear();
        self.command_tab = None;
        self.expected_text = None;
    }
    pub(crate) fn process(
        &mut self,
        events: &[GuiInputEvent],
        tabs: &[RibbonTab],
        text_focused: bool,
        grouped: bool,
    ) -> Option<KeyTipAction> {
        self.consumed = false;
        let mut action = None;
        for event in events {
            let Some(key) = self.input_key(event, text_focused) else {
                continue;
            };
            self.expected_text = None;
            if let Some(selected) = self.key(key, tabs, grouped) {
                action = Some(selected);
            }
            if self.consumed {
                self.expected_text = key_character(key);
            }
        }
        action
    }
    fn input_key(&mut self, event: &GuiInputEvent, text_focused: bool) -> Option<GuiKey> {
        match event {
            GuiInputEvent::Focus(false) => {
                self.dismiss();
                None
            }
            GuiInputEvent::PointerButton { pressed: true, .. } => {
                self.dismiss();
                None
            }
            GuiInputEvent::Text(text) => {
                let expected = self.expected_text.take();
                if expected
                    .is_some_and(|character| text.eq_ignore_ascii_case(&character.to_string()))
                {
                    self.consumed = true;
                }
                None
            }
            GuiInputEvent::Key { key, pressed: true } if !text_focused => Some(*key),
            _ => None,
        }
    }
    fn key(&mut self, key: GuiKey, tabs: &[RibbonTab], grouped: bool) -> Option<KeyTipAction> {
        if matches!(key, GuiKey::LeftAlt | GuiKey::RightAlt) {
            self.consumed = true;
            if self.active() {
                self.dismiss();
            } else {
                self.level = Level::Tabs;
            }
            return None;
        }
        if !self.active() {
            return None;
        }
        if key == GuiKey::Escape {
            self.consumed = true;
            if self.commands() && !self.prefix.is_empty() {
                self.level = Level::Tabs;
                self.prefix.clear();
            } else if self.tabs() && !self.tools_face_mode {
                self.dismiss();
            }
            return None;
        }
        let character = key_character(key)?;
        self.consumed = true;
        if self.tabs() {
            return self.tab_key(character, tabs);
        }
        self.command_key(character, grouped)
    }
    fn tab_key(&mut self, character: char, tabs: &[RibbonTab]) -> Option<KeyTipAction> {
        if self.tools_face_mode && (self.prefix == "O" || character == 'O') {
            return self.tools_option_key(character);
        }
        if self.prefix == "X" {
            let action = match character {
                '2' => Some(KeyTipAction::CycleDisplay),
                '3' => Some(KeyTipAction::OpenDisplayMenu),
                _ => None,
            };
            if action.is_some() {
                self.dismiss();
            }
            return action;
        }
        if character == 'F' && self.prefix.is_empty() {
            self.dismiss();
            return Some(KeyTipAction::OpenFileMenu);
        }
        if character == 'X' {
            self.prefix.push(character);
            return None;
        }
        if let Some(number) = character.to_digit(10).filter(|number| *number > 0) {
            self.dismiss();
            return Some(KeyTipAction::QuickAccess(number as usize - 1));
        }
        let tab = tabs
            .iter()
            .copied()
            .find(|tab| tab_tip(*tab) == character)?;
        self.level = Level::Commands;
        self.command_tab = Some(tab);
        self.prefix.clear();
        Some(KeyTipAction::SelectTab(tab))
    }
    fn tools_option_key(&mut self, character: char) -> Option<KeyTipAction> {
        if self.prefix.is_empty() && character == 'O' {
            self.prefix.push(character);
            return None;
        }
        let action = match character {
            'A' => Some(KeyTipAction::ApplicationOptions),
            'D' => Some(KeyTipAction::DocumentSettings),
            _ => None,
        };
        if action.is_some() {
            self.dismiss();
        }
        action
    }
    fn command_key(&mut self, character: char, grouped: bool) -> Option<KeyTipAction> {
        if self.command_tab != Some(RibbonTab::Model) {
            return None;
        }
        if grouped && self.prefix.is_empty() {
            let panel = match character {
                'Y' => Some(crate::ribbon_panels::RibbonPanel::PartModify),
                'K' => Some(crate::ribbon_panels::RibbonPanel::Sketch),
                'C' => Some(crate::ribbon_panels::RibbonPanel::PartCreate),
                _ => None,
            };
            if let Some(panel) = panel {
                return Some(KeyTipAction::OpenGroup(panel));
            }
        }
        let candidate = format!("{}{character}", self.prefix);
        let action = match candidate.as_str() {
            "SK" => Some(KeyTipAction::StartSketch),
            "EX" => Some(KeyTipAction::Extrude),
            "HO" => Some(KeyTipAction::Hole),
            _ => None,
        };
        if action.is_some() {
            self.dismiss();
            return action;
        }
        if ["HO", "SK", "EX"]
            .iter()
            .any(|tip| tip.starts_with(&candidate))
        {
            self.prefix = candidate;
        }
        None
    }
}
pub(crate) fn tab_tip(tab: RibbonTab) -> char {
    match tab {
        RibbonTab::GetStarted => 'H',
        RibbonTab::Model | RibbonTab::SheetMetal => 'M',
        RibbonTab::Assemble | RibbonTab::Annotate => 'A',
        RibbonTab::PlaceViews | RibbonTab::Presentation => 'P',
        RibbonTab::Sketch => 'S',
        RibbonTab::Tools => 'T',
        RibbonTab::Manage => 'G',
        RibbonTab::View => 'V',
    }
}
fn key_character(key: GuiKey) -> Option<char> {
    use GuiKey::*;
    const LETTERS: [(GuiKey, char); 36] = [
        (A, 'A'),
        (B, 'B'),
        (C, 'C'),
        (D, 'D'),
        (E, 'E'),
        (F, 'F'),
        (G, 'G'),
        (H, 'H'),
        (I, 'I'),
        (J, 'J'),
        (K, 'K'),
        (L, 'L'),
        (M, 'M'),
        (N, 'N'),
        (O, 'O'),
        (P, 'P'),
        (Q, 'Q'),
        (R, 'R'),
        (S, 'S'),
        (T, 'T'),
        (U, 'U'),
        (V, 'V'),
        (W, 'W'),
        (X, 'X'),
        (Y, 'Y'),
        (Z, 'Z'),
        (Alpha0, '0'),
        (Alpha1, '1'),
        (Alpha2, '2'),
        (Alpha3, '3'),
        (Alpha4, '4'),
        (Alpha5, '5'),
        (Alpha6, '6'),
        (Alpha7, '7'),
        (Alpha8, '8'),
        (Alpha9, '9'),
    ];
    LETTERS
        .iter()
        .find_map(|(known, value)| (*known == key).then_some(*value))
}
/// Draw a key tip beside the last control; example: after an ImGui tab item.
pub(crate) fn badge(ui: &imgui::Ui, label: &str) {
    let [left, top] = ui.item_rect_min();
    let size = ui.calc_text_size(label);
    let from = [left + 2., top + 3.];
    let to = [from[0] + size[0] + 8., from[1] + size[1] + 4.];
    let draw = ui.get_foreground_draw_list();
    draw.add_rect(from, to, [0.98, 0.96, 0.83, 1.])
        .filled(true)
        .build();
    draw.add_rect(from, to, [0.2, 0.2, 0.2, 1.]).build();
    draw.add_text([from[0] + 4., from[1] + 2.], [0.1, 0.1, 0.1, 1.], label);
}

#[cfg(test)]
#[path = "../tests/ribbon_key_tips.rs"]
mod tests;
