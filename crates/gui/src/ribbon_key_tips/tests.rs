// SPDX-License-Identifier: GPL-2.0-only
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
fn selected_face_tools_tips_retain_escape_and_open_options() {
    let mut tips = RibbonKeyTips::default();
    tips.set_tools_face_mode(true);
    press(&mut tips, GuiKey::LeftAlt);
    assert!(tips.tools_face_tip("OA") && tips.tools_face_tip("OD"));
    press(&mut tips, GuiKey::Escape);
    assert!(tips.tabs());
    press(&mut tips, GuiKey::O);
    assert_eq!(
        press(&mut tips, GuiKey::D),
        Some(KeyTipAction::DocumentSettings)
    );
    assert!(!tips.active());
    press(&mut tips, GuiKey::LeftAlt);
    press(&mut tips, GuiKey::O);
    assert_eq!(
        press(&mut tips, GuiKey::A),
        Some(KeyTipAction::ApplicationOptions)
    );
    tips.set_tools_face_mode(false);
    press(&mut tips, GuiKey::LeftAlt);
    press(&mut tips, GuiKey::Escape);
    assert!(!tips.active());
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
fn hole_tip_recovers_after_an_unmatched_letter() {
    let mut tips = RibbonKeyTips::default();
    press(&mut tips, GuiKey::LeftAlt);
    press(&mut tips, GuiKey::M);
    press(&mut tips, GuiKey::H);
    press(&mut tips, GuiKey::Z);
    assert!(tips.matching("HO"));
    assert_eq!(press(&mut tips, GuiKey::O), Some(KeyTipAction::Hole));
    assert!(!tips.active());
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
fn sketch_create_tip_opens_additional_commands() {
    let mut tips = RibbonKeyTips::default();
    let tabs = [RibbonTab::Model, RibbonTab::Sketch];
    let mut press = |key| {
        tips.process(
            &[GuiInputEvent::Key { key, pressed: true }],
            &tabs,
            false,
            false,
        )
    };
    press(GuiKey::LeftAlt);
    assert_eq!(
        press(GuiKey::S),
        Some(KeyTipAction::SelectTab(RibbonTab::Sketch))
    );
    assert_eq!(
        press(GuiKey::D),
        Some(KeyTipAction::OpenAdditional(
            crate::ribbon_panels::RibbonPanel::Create
        ))
    );
}
#[test]
fn sketch_create_tip_is_not_shown_for_grouped_ribbons() {
    let mut tips = RibbonKeyTips::default();
    let tabs = [RibbonTab::Sketch];
    for key in [GuiKey::LeftAlt, GuiKey::S] {
        tips.process(
            &[GuiInputEvent::Key { key, pressed: true }],
            &tabs,
            false,
            true,
        );
    }
    assert!(
        !tips.create_group_tip(crate::application::ribbon_display::RibbonDisplay::PanelButtons)
    );
    assert_eq!(
        tips.process(
            &[GuiInputEvent::Key {
                key: GuiKey::D,
                pressed: true
            }],
            &tabs,
            false,
            true
        ),
        None
    );
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
fn modify_tip_opens_the_matching_panel_for_each_ribbon_layout() {
    let mut tips = RibbonKeyTips::default();
    press(&mut tips, GuiKey::LeftAlt);
    press(&mut tips, GuiKey::M);
    assert_eq!(
        press(&mut tips, GuiKey::Y),
        Some(KeyTipAction::OpenAdditional(
            crate::ribbon_panels::RibbonPanel::PartModify
        ))
    );
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

#[test]
fn model_modify_tip_opens_additional_area_in_a_full_ribbon() {
    let mut tips = RibbonKeyTips::default();
    press(&mut tips, GuiKey::LeftAlt);
    press(&mut tips, GuiKey::M);
    assert_eq!(
        press(&mut tips, GuiKey::Y),
        Some(KeyTipAction::OpenAdditional(
            crate::ribbon_panels::RibbonPanel::PartModify
        ))
    );
    assert!(tips.commands());
}

#[test]
fn modify_additional_move_body_tip_opens_the_tool() {
    let mut tips = RibbonKeyTips::default();
    press(&mut tips, GuiKey::LeftAlt);
    press(&mut tips, GuiKey::M);
    assert!(matches!(
        press(&mut tips, GuiKey::Y),
        Some(KeyTipAction::OpenAdditional(_))
    ));
    assert_eq!(press(&mut tips, GuiKey::O), None);
    assert_eq!(press(&mut tips, GuiKey::D), Some(KeyTipAction::MoveBody));
    assert!(!tips.active());
}
