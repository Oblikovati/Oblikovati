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
    OpenGroup(crate::ribbon_panels::RibbonPanel),
}
impl RibbonKeyTips {
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
            } else if self.tabs() {
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
            _ => None,
        };
        if action.is_some() {
            self.dismiss();
            return action;
        }
        // Hole is disabled until the host can create it.
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
mod tests {
    use super::*;
    fn press(tips: &mut RibbonKeyTips, key: GuiKey) -> Option<KeyTipAction> {
        tips.process(
            &[GuiInputEvent::Key { key, pressed: true }],
            &[RibbonTab::Model, RibbonTab::Tools],
            false,
            false,
        )
    }
    #[test]
    fn tab_tips_ignore_unknown_keys_and_select_a_document_tab() {
        let mut tips = RibbonKeyTips::default();
        press(&mut tips, GuiKey::LeftAlt);
        assert!(tips.tabs());
        press(&mut tips, GuiKey::Z);
        assert!(tips.tabs() && tips.consumed);
        assert_eq!(
            press(&mut tips, GuiKey::M),
            Some(KeyTipAction::SelectTab(RibbonTab::Model))
        );
        assert!(tips.commands());
        press(&mut tips, GuiKey::H);
        press(&mut tips, GuiKey::Z);
        assert!(tips.matching("HO"));
        press(&mut tips, GuiKey::Escape);
        assert!(tips.tabs());
    }
    #[test]
    fn file_key_opens_menu_and_closes_ribbon_tips() {
        let mut tips = RibbonKeyTips::default();
        press(&mut tips, GuiKey::LeftAlt);
        assert_eq!(
            press(&mut tips, GuiKey::F),
            Some(KeyTipAction::OpenFileMenu)
        );
        assert!(!tips.active());
    }
    #[test]
    fn display_and_quick_access_actions_close_tips() {
        let mut tips = RibbonKeyTips::default();
        press(&mut tips, GuiKey::LeftAlt);
        press(&mut tips, GuiKey::X);
        assert_eq!(
            press(&mut tips, GuiKey::Alpha2),
            Some(KeyTipAction::CycleDisplay)
        );
        assert!(!tips.active());
        press(&mut tips, GuiKey::LeftAlt);
        assert_eq!(
            press(&mut tips, GuiKey::Alpha1),
            Some(KeyTipAction::QuickAccess(0))
        );
        assert!(!tips.active());
    }
    #[test]
    fn document_tab_sets_have_unambiguous_tips() {
        for tabs in [
            &[RibbonTab::GetStarted, RibbonTab::Tools][..],
            &[
                RibbonTab::Model,
                RibbonTab::Sketch,
                RibbonTab::Tools,
                RibbonTab::Manage,
                RibbonTab::View,
                RibbonTab::Annotate,
            ],
            &[
                RibbonTab::Assemble,
                RibbonTab::Sketch,
                RibbonTab::Tools,
                RibbonTab::Manage,
                RibbonTab::View,
            ],
            &[
                RibbonTab::PlaceViews,
                RibbonTab::Annotate,
                RibbonTab::Tools,
                RibbonTab::View,
            ],
            &[RibbonTab::Presentation, RibbonTab::Tools, RibbonTab::View],
        ] {
            let tips: std::collections::BTreeSet<_> = tabs.iter().copied().map(tab_tip).collect();
            assert_eq!(tips.len(), tabs.len(), "tabs {tabs:?}");
        }
    }
    #[test]
    fn consumed_key_text_does_not_reach_a_later_widget() {
        let mut tips = RibbonKeyTips::default();
        press(&mut tips, GuiKey::LeftAlt);
        press(&mut tips, GuiKey::X);
        press(&mut tips, GuiKey::Alpha2);
        assert!(!tips.active());
        tips.process(&[GuiInputEvent::Text("2".into())], &[], false, false);
        assert!(tips.consumed);
        tips.process(&[GuiInputEvent::Text("a".into())], &[], false, false);
        assert!(!tips.consumed);
    }
    #[test]
    fn model_commands_resolve_and_leave_the_tip_mode() {
        let mut tips = RibbonKeyTips::default();
        press(&mut tips, GuiKey::LeftAlt);
        press(&mut tips, GuiKey::M);
        press(&mut tips, GuiKey::S);
        assert_eq!(press(&mut tips, GuiKey::K), Some(KeyTipAction::StartSketch));
        assert!(!tips.active());
    }
    #[test]
    fn modify_group_tip_opens_only_for_a_grouped_model_ribbon() {
        let mut tips = RibbonKeyTips::default();
        press(&mut tips, GuiKey::LeftAlt);
        press(&mut tips, GuiKey::M);
        assert_eq!(press(&mut tips, GuiKey::Y), None);
        let action = tips.process(
            &[GuiInputEvent::Key {
                key: GuiKey::Y,
                pressed: true,
            }],
            &[RibbonTab::Model],
            false,
            true,
        );
        assert_eq!(
            action,
            Some(KeyTipAction::OpenGroup(
                crate::ribbon_panels::RibbonPanel::PartModify
            ))
        );
        assert!(tips.commands() && tips.matching("HO"));
    }
    #[test]
    fn model_command_tips_are_hidden_on_other_tabs() {
        let mut tips = RibbonKeyTips::default();
        tips.process(
            &[GuiInputEvent::Key {
                key: GuiKey::LeftAlt,
                pressed: true,
            }],
            &[RibbonTab::Sketch],
            false,
            false,
        );
        tips.process(
            &[GuiInputEvent::Key {
                key: GuiKey::S,
                pressed: true,
            }],
            &[RibbonTab::Sketch],
            false,
            false,
        );
        assert!(tips.commands());
        assert!(!tips.model_commands());
    }
    #[test]
    fn text_focus_blocks_activation_and_arrows_remain_unclaimed() {
        let mut tips = RibbonKeyTips::default();
        tips.process(
            &[GuiInputEvent::Key {
                key: GuiKey::LeftAlt,
                pressed: true,
            }],
            &[],
            true,
            false,
        );
        assert!(!tips.active());
        press(&mut tips, GuiKey::LeftAlt);
        press(&mut tips, GuiKey::LeftArrow);
        assert!(tips.tabs() && !tips.consumed);
        tips.process(&[GuiInputEvent::Focus(false)], &[], false, false);
        assert!(!tips.active());
    }
}
