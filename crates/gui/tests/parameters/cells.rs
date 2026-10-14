// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{GuiInputEvent, input::apply_input};
use imgui::Context;
struct ParameterCellWidgetFixture {
    context: Context,
    drafts: GuiParameterDrafts,
    target: GuiParameterTextTarget,
}
impl ParameterCellWidgetFixture {
    fn new() -> Self {
        let mut context = Context::create();
        context.set_ini_filename(None);
        context.io_mut().display_size = [400., 240.];
        context.io_mut().delta_time = 1. / 60.;
        context.fonts().build_rgba32_texture();
        let snapshot = GuiParameterSnapshot {
            linked_tables: Vec::new(),
            link_sources: Vec::new(),
            document: 7,
            rows: vec![GuiParameterRow {
                tolerance_text: String::new(),
                has_groups: false,
                parameter: 11,
                detail: Default::default(),
            }],
        };
        let mut drafts = GuiParameterDrafts::new(128, 8).unwrap();
        drafts.synchronize(Some(&snapshot)).unwrap();
        Self {
            context,
            drafts,
            target: GuiParameterTextTarget {
                document: 7,
                parameter: 11,
                field: GuiParameterTextField::Equation,
            },
        }
    }
    fn frame(
        &mut self,
        events: &[GuiInputEvent],
        focus: bool,
        blur: bool,
    ) -> Vec<GuiParameterAction> {
        for event in events {
            apply_input(self.context.io_mut(), event, [1.; 2]);
        }
        let mut actions = Vec::new();
        let ui = self.context.frame();
        ui.window("Parameter cell probe")
            .size([300., 160.], Condition::Always)
            .build(|| {
                if focus {
                    ui.set_keyboard_focus_here();
                }
                draw_text_editor(
                    ui,
                    ("##equation", ""),
                    self.target,
                    None,
                    &mut self.drafts,
                    &mut actions,
                );
                if blur {
                    ui.set_keyboard_focus_here();
                }
                let mut other = String::new();
                ui.input_text("Other", &mut other).build();
            });
        self.context.render();
        actions
    }
    fn focus(&mut self) {
        self.frame(&[], false, false);
        self.frame(&[], true, false);
        self.frame(&[], false, false);
    }
}
#[test]
fn real_parameter_text_widget_commits_on_focus_loss_once_and_keeps_failed_text() {
    let mut fixture = ParameterCellWidgetFixture::new();
    fixture.focus();
    assert!(
        fixture
            .frame(&[GuiInputEvent::Text("6 cm".into())], false, false)
            .is_empty()
    );
    fixture.frame(&[], false, true);
    let actions = fixture.frame(&[], false, false);
    let [GuiParameterAction::Text(edit)] = actions.as_slice() else {
        panic!("expected one owned parameter edit, received {actions:?}");
    };
    assert_eq!(edit.value, "6 cm");
    assert_eq!(edit.target, fixture.target);
    assert!(fixture.frame(&[], false, false).is_empty());
    assert_eq!(
        fixture
            .drafts
            .acknowledge(edit, Err("Rejected edit".into())),
        GuiParameterAck::Failed
    );
    assert_eq!(fixture.drafts.text(fixture.target), Some("6 cm"));
    fixture.focus();
    fixture.frame(&[], false, true);
    assert!(fixture.frame(&[], false, false).is_empty());
}
#[test]
fn panel_response_forwarding_retires_only_the_matching_text_request() {
    let mut panel = ParameterPanel::new().unwrap();
    let view = GuiParameterSnapshot {
        linked_tables: Vec::new(),
        link_sources: Vec::new(),
        document: 7,
        rows: vec![GuiParameterRow {
            tolerance_text: String::new(),
            has_groups: false,
            parameter: 11,
            detail: Default::default(),
        }],
    };
    panel.replace_snapshot(Some(view)).unwrap();
    let target = GuiParameterTextTarget {
        document: 7,
        parameter: 11,
        field: GuiParameterTextField::Comment,
    };
    *panel.drafts.text_buffer(target, "").unwrap() = "é🙂".into();
    let edit = panel.drafts.submit(target).unwrap().unwrap();
    assert_eq!(panel.acknowledge(&edit, Ok(())), GuiParameterAck::Applied);
    assert_eq!(panel.acknowledge(&edit, Ok(())), GuiParameterAck::Obsolete);
}
#[test]
fn choice_labels_retain_the_current_custom_expression_after_a_list_changes() {
    let mut row = GuiParameterRow {
        parameter: 1,
        tolerance_text: String::new(),
        has_groups: false,
        detail: Default::default(),
    };
    row.detail.parameter_info.expression = "custom".into();
    row.detail.expression_list = Some(Box::new(oblikovati_api::wire::ExpressionListInfo {
        expressions: Some(vec!["one".into(), "two".into()]),
        ..Default::default()
    }));
    assert_eq!(choice_labels(&row), vec!["one", "two", "custom"]);
}
struct ParameterRowMenuWidgetFixture {
    context: Context,
    last_item: [f32; 2],
    menu_stride: f32,
    value_row: Option<GuiParameterRow>,
    tolerance_menu: bool,
    group_menu: bool,
}
impl ParameterRowMenuWidgetFixture {
    fn new() -> Self {
        let mut context = Context::create();
        context.set_ini_filename(None);
        context.io_mut().display_size = [500., 300.];
        context.io_mut().delta_time = 1. / 60.;
        context.fonts().build_rgba32_texture();
        Self {
            context,
            last_item: [0.; 2],
            menu_stride: 0.,
            value_row: None,
            tolerance_menu: false,
            group_menu: false,
        }
    }
    fn frame(
        &mut self,
        open: bool,
        editable: bool,
        events: &[GuiInputEvent],
    ) -> Vec<GuiParameterAction> {
        for event in events {
            apply_input(self.context.io_mut(), event, [1.; 2]);
        }
        let ui = self.context.frame();
        let mut actions = Vec::new();
        ui.window("Row menu probe")
            .size([350., 180.], Condition::Always)
            .build(|| {
                ui.text("width");
                if open {
                    ui.open_popup("##parameter-row-menu");
                }
                if let Some(_popup) = ui.begin_popup("##parameter-row-menu") {
                    match &self.value_row {
                        Some(row) if self.group_menu => {
                            draw_parameter_group_menu(ui, 7, row, &mut actions)
                        }
                        Some(row) if self.tolerance_menu => {
                            draw_parameter_tolerance_menu(ui, 7, row, editable, &mut actions)
                        }
                        Some(row) => draw_parameter_value_menu(ui, 7, row, editable, &mut actions),
                        None => draw_parameter_row_menu_items(ui, 7, 11, editable, &mut actions),
                    }
                    self.menu_stride = ui.text_line_height_with_spacing();
                    let minimum = ui.item_rect_min();
                    let maximum = ui.item_rect_max();
                    self.last_item =
                        std::array::from_fn(|axis| (minimum[axis] + maximum[axis]) / 2.);
                }
            });
        self.context.render();
        actions
    }
    fn choose(&mut self, editable: bool, position: [f32; 2]) -> Vec<GuiParameterAction> {
        use crate::GuiPointerButton;
        self.frame(false, editable, &[GuiInputEvent::PointerMoved(position)]);
        self.frame(
            false,
            editable,
            &[GuiInputEvent::PointerButton {
                button: GuiPointerButton::Left,
                pressed: true,
            }],
        );
        self.frame(
            false,
            editable,
            &[GuiInputEvent::PointerButton {
                button: GuiPointerButton::Left,
                pressed: false,
            }],
        )
    }
}
#[test]
fn row_menu_delete_emits_the_model_identity_and_is_disabled_for_read_only_rows() {
    for editable in [true, false] {
        let mut fixture = ParameterRowMenuWidgetFixture::new();
        fixture.frame(
            false,
            editable,
            &[GuiInputEvent::PointerMoved([120., 120.])],
        );
        fixture.frame(true, editable, &[]);
        fixture.frame(false, editable, &[]);
        let position = fixture.last_item;
        let actions = fixture.choose(editable, position);
        if !editable {
            assert!(actions.is_empty());
            continue;
        }
        assert_eq!(
            actions,
            vec![GuiParameterAction::Row {
                document: 7,
                parameter: 11,
                operation: GuiParameterRowOperation::Delete
            }],
            "last menu item {position:?}"
        );
    }
}

#[test]
fn row_menu_copy_is_available_for_editable_and_read_only_parameters() {
    for editable in [true, false] {
        let mut fixture = ParameterRowMenuWidgetFixture::new();
        fixture.frame(
            false,
            editable,
            &[GuiInputEvent::PointerMoved([120., 120.])],
        );
        fixture.frame(true, editable, &[]);
        fixture.frame(false, editable, &[]);
        let position = [
            fixture.last_item[0],
            fixture.last_item[1] - fixture.menu_stride,
        ];
        assert_eq!(
            fixture.choose(editable, position),
            vec![GuiParameterAction::Row {
                document: 7,
                parameter: 11,
                operation: GuiParameterRowOperation::CopyToUser
            }]
        );
    }
}

#[test]
fn value_list_menu_opens_or_clears_lists_and_hides_commands_on_boolean_and_read_only_rows() {
    for (units, editable, listed) in [
        ("text", true, false),
        ("mm", true, true),
        ("boolean", true, true),
        ("text", false, true),
    ] {
        let mut fixture = ParameterRowMenuWidgetFixture::new();
        let mut detail = oblikovati_api::wire::ParameterDetail {
            units: units.into(),
            ..Default::default()
        };
        detail.expression_list =
            listed.then(|| Box::new(oblikovati_api::wire::ExpressionListInfo::default()));
        fixture.value_row = Some(GuiParameterRow {
            tolerance_text: String::new(),
            has_groups: false,
            parameter: 11,
            detail,
        });
        fixture.frame(
            false,
            editable,
            &[GuiInputEvent::PointerMoved([120., 120.])],
        );
        fixture.frame(true, editable, &[]);
        fixture.frame(false, editable, &[]);
        let actions = fixture.choose(editable, fixture.last_item);
        if !editable || units == "boolean" {
            assert!(actions.is_empty());
            continue;
        }
        let operation = if listed {
            GuiParameterRowOperation::ClearValueList
        } else {
            GuiParameterRowOperation::EditValueList
        };
        assert_eq!(
            actions,
            vec![GuiParameterAction::Row {
                document: 7,
                parameter: 11,
                operation
            }]
        );
    }
}

#[test]
fn listed_equation_choices_keep_failed_source_and_allow_only_a_new_idle_attempt() {
    let mut fixture = ParameterCellWidgetFixture::new();
    let target = fixture.target;
    let first = choose_parameter_expression(&mut fixture.drafts, target, "4 cm", "(")
        .unwrap()
        .unwrap();
    assert!(fixture.drafts.is_pending(target));
    assert!(choose_parameter_expression(&mut fixture.drafts, target, "4 cm", "8 cm").is_err());
    fixture
        .drafts
        .acknowledge(&first, Err("formula (: expected a valid expression".into()));
    assert_eq!(fixture.drafts.text(target), Some("("));
    assert!(fixture.drafts.issue(target).is_some());
    assert!(
        choose_parameter_expression(&mut fixture.drafts, target, "4 cm", "(")
            .unwrap()
            .is_none()
    );
    let next = choose_parameter_expression(&mut fixture.drafts, target, "4 cm", "8 cm")
        .unwrap()
        .unwrap();
    assert!(next.request > first.request);
    assert_eq!(next.value, "8 cm");
}

#[test]
fn tolerance_menu_is_available_only_for_editable_numeric_rows() {
    for (units, editable) in [
        ("mm", true),
        ("unitless", true),
        ("text", true),
        ("boolean", true),
        ("mm", false),
    ] {
        let mut fixture = ParameterRowMenuWidgetFixture::new();
        fixture.tolerance_menu = true;
        fixture.value_row = Some(GuiParameterRow {
            tolerance_text: String::new(),
            has_groups: false,
            parameter: 11,
            detail: oblikovati_api::wire::ParameterDetail {
                units: units.into(),
                ..Default::default()
            },
        });
        fixture.frame(
            false,
            editable,
            &[GuiInputEvent::PointerMoved([120., 120.])],
        );
        fixture.frame(true, editable, &[]);
        fixture.frame(false, editable, &[]);
        let actions = fixture.choose(editable, fixture.last_item);
        if !editable || matches!(units, "text" | "boolean") {
            assert!(actions.is_empty());
            continue;
        }
        assert_eq!(
            actions,
            vec![GuiParameterAction::Row {
                document: 7,
                parameter: 11,
                operation: GuiParameterRowOperation::EditTolerance
            }]
        );
    }
}

#[test]
fn group_menu_offers_add_on_read_only_rows_and_remove_only_when_memberships_exist() {
    for grouped in [false, true] {
        let mut fixture = ParameterRowMenuWidgetFixture::new();
        fixture.group_menu = true;
        fixture.value_row = Some(GuiParameterRow {
            parameter: 11,
            detail: Default::default(),
            tolerance_text: String::new(),
            has_groups: grouped,
        });
        fixture.frame(false, false, &[GuiInputEvent::PointerMoved([120., 120.])]);
        fixture.frame(true, false, &[]);
        fixture.frame(false, false, &[]);
        let actions = fixture.choose(false, fixture.last_item);
        let operation = if grouped {
            GuiParameterRowOperation::RemoveFromGroups
        } else {
            GuiParameterRowOperation::AddToGroup
        };
        assert_eq!(
            actions,
            vec![GuiParameterAction::Row {
                document: 7,
                parameter: 11,
                operation
            }]
        );
    }
}

#[test]
fn failed_parameter_values_keep_last_good_text_and_show_an_explicit_warning() {
    let mut row = GuiParameterRow {
        parameter: 0,
        detail: Default::default(),
        tolerance_text: String::new(),
        has_groups: false,
    };
    row.detail.parameter_info.value = "80 mm".into();
    assert_eq!(parameter_value_label(&row), "80 mm");
    row.detail.parameter_info.health = "unknown parameter span".into();
    assert_eq!(parameter_value_label(&row), "! 80 mm");
}
