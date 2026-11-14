// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn value_list_row() -> GuiParameterRow {
    GuiParameterRow {
        tolerance_text: String::new(),
        has_groups: false,
        parameter: 0,
        detail: oblikovati_api::wire::ParameterDetail {
            expression_list: Some(Box::new(ExpressionListInfo {
                expressions: Some(vec!["é🙂".into(), "steel".into()]),
                allow_custom_values: true,
                custom_order: true,
            })),
            ..Default::default()
        },
    }
}
#[test]
fn value_list_editor_seeds_zero_identity_custom_policy_and_keeps_authored_order() {
    let mut editor = ParameterValueListEditor::default();
    editor.open(7, &value_list_row()).unwrap();
    assert_eq!(editor.text, "é🙂\nsteel");
    assert!(editor.allow_custom);
    editor.text = "  steel,é🙂\n\n wood , steel \r\n".into();
    let edit = editor.submit().unwrap().unwrap();
    assert_eq!(edit.target.parameter, 0);
    assert_eq!(
        edit.choices.expressions,
        Some(vec![
            "steel".into(),
            "é🙂".into(),
            "wood".into(),
            "steel".into()
        ])
    );
    assert!(edit.choices.custom_order);
    assert!(edit.choices.allow_custom_values);
    assert!(editor.submit().unwrap().is_none());
    assert_eq!(editor.acknowledge(&edit, Ok(())), GuiParameterAck::Applied);
    assert!(editor.target.is_none());
}
#[test]
fn failed_value_lists_keep_text_and_stale_results_cannot_close_reopened_editors() {
    let mut editor = ParameterValueListEditor::default();
    editor.open(7, &value_list_row()).unwrap();
    let first = editor.submit().unwrap().unwrap();
    assert_eq!(
        editor.acknowledge(&first, Err("parameter 0: expected an editable row".into())),
        GuiParameterAck::Failed
    );
    assert_eq!(editor.text, "é🙂\nsteel");
    assert!(editor.request.issue.is_some());
    editor.open(8, &value_list_row()).unwrap();
    let current = editor.submit().unwrap().unwrap();
    assert!(current.request > first.request);
    assert_eq!(
        editor.acknowledge(&first, Ok(())),
        GuiParameterAck::Obsolete
    );
    let mut forged = current.clone();
    forged.choices.allow_custom_values = false;
    assert_eq!(
        editor.acknowledge(&forged, Ok(())),
        GuiParameterAck::Obsolete
    );
    assert!(editor.target.is_some());
}
#[test]
fn value_list_document_switch_and_deletion_retire_only_stale_targets() {
    let mut editor = ParameterValueListEditor::default();
    let snapshot = GuiParameterSnapshot {
        linked_tables: Vec::new(),
        link_sources: Vec::new(),
        groups: Vec::new(),
        document: 7,
        rows: vec![value_list_row()],
    };
    editor.open(7, &snapshot.rows[0]).unwrap();
    editor.synchronize(Some(&snapshot));
    assert!(editor.target.is_some());
    editor.synchronize(Some(&GuiParameterSnapshot {
        linked_tables: Vec::new(),
        link_sources: Vec::new(),
        groups: Vec::new(),
        document: 7,
        rows: Vec::new(),
    }));
    assert!(editor.target.is_none());
    editor.open(7, &snapshot.rows[0]).unwrap();
    editor.synchronize(Some(&GuiParameterSnapshot {
        linked_tables: Vec::new(),
        link_sources: Vec::new(),
        groups: Vec::new(),
        document: 8,
        rows: snapshot.rows,
    }));
    assert!(editor.target.is_none());
    editor.open(7, &value_list_row()).unwrap();
    editor.synchronize(None);
    assert!(editor.target.is_none());
}
#[test]
fn empty_value_lists_clear_the_policy_and_invalid_text_is_refused_before_pending_state() {
    let cleared = edited_choices(" ,\n\t", true);
    assert_eq!(cleared.expressions, None);
    assert!(!cleared.allow_custom_values);
    assert!(!cleared.custom_order);
    let mut editor = ParameterValueListEditor::default();
    assert!(editor.submit().is_err());
    editor.open(7, &value_list_row()).unwrap();
    editor.text = "bad\0value".into();
    assert!(editor.submit().is_err());
    assert!(editor.request.pending.is_none());
    editor.text = "steel".into();
    editor.request.last_request = u64::MAX;
    assert!(editor.submit().is_err());
    assert!(editor.request.pending.is_none());
}
#[test]
fn value_list_enqueue_reports_local_errors_and_emits_one_owned_packet() {
    let mut editor = ParameterValueListEditor::default();
    let mut actions = Vec::new();
    editor.enqueue(&mut actions);
    assert!(editor.request.issue.is_some());
    assert!(actions.is_empty());
    editor.open(7, &value_list_row()).unwrap();
    editor.enqueue(&mut actions);
    editor.enqueue(&mut actions);
    assert!(matches!(
        actions.as_slice(),
        [GuiParameterAction::Choices(_)]
    ));
}
#[test]
fn owned_value_list_packets_refuse_invalid_targets_and_unsafe_widget_text() {
    let mut editor = ParameterValueListEditor::default();
    editor.open(7, &value_list_row()).unwrap();
    let valid = editor.submit().unwrap().unwrap();
    valid.validate().unwrap();
    for (document, field, request) in [
        (0, GuiParameterTextField::Equation, 1),
        (7, GuiParameterTextField::Name, 1),
        (7, GuiParameterTextField::Equation, 0),
    ] {
        let mut edit = valid.clone();
        edit.target.document = document;
        edit.target.field = field;
        edit.request = request;
        assert!(edit.validate().is_err());
    }
    for choices in [
        vec!["bad\0value".into()],
        vec!["x".repeat(600_000), "y".repeat(600_000)],
    ] {
        let mut edit = valid.clone();
        edit.choices.expressions = Some(choices);
        assert!(edit.validate().is_err());
    }
}
struct ValueListWidgetFixture {
    context: imgui::Context,
    editor: ParameterValueListEditor,
    ok_position: [f32; 2],
}
impl ValueListWidgetFixture {
    fn new() -> Self {
        let mut context = imgui::Context::create();
        context.set_ini_filename(None);
        context.io_mut().display_size = [500., 350.];
        context.io_mut().delta_time = 1. / 60.;
        context.fonts().build_rgba32_texture();
        let mut editor = ParameterValueListEditor::default();
        let mut row = value_list_row();
        row.detail.expression_list = None;
        editor.open(7, &row).unwrap();
        Self {
            context,
            editor,
            ok_position: [0.; 2],
        }
    }
    fn frame(&mut self, focus: bool, events: &[crate::GuiInputEvent]) -> Vec<GuiParameterAction> {
        for event in events {
            crate::input::apply_input(self.context.io_mut(), event, [1.; 2]);
        }
        let spacing = self.context.style().item_spacing[0];
        let padding = self.context.style().frame_padding[0];
        let ui = self.context.frame();
        let mut actions = Vec::new();
        ui.window("Value-list widget probe")
            .position([8., 8.], Condition::Always)
            .size([400., 300.], Condition::Always)
            .build(|| {
                if focus {
                    ui.set_keyboard_focus_here();
                }
                self.editor.draw_body(ui, &mut actions);
                let minimum = ui.item_rect_min();
                let maximum = ui.item_rect_max();
                self.ok_position = [
                    minimum[0] - spacing - (ui.calc_text_size("OK")[0] + 2. * padding) / 2.,
                    (minimum[1] + maximum[1]) / 2.,
                ];
            });
        self.context.render();
        actions
    }
    fn click_ok(&mut self) -> Vec<GuiParameterAction> {
        use crate::{GuiInputEvent, GuiPointerButton};
        self.frame(false, &[GuiInputEvent::PointerMoved(self.ok_position)]);
        self.frame(
            false,
            &[GuiInputEvent::PointerButton {
                button: GuiPointerButton::Left,
                pressed: true,
            }],
        );
        self.frame(
            false,
            &[GuiInputEvent::PointerButton {
                button: GuiPointerButton::Left,
                pressed: false,
            }],
        )
    }
}
#[test]
fn real_multiline_list_widget_and_ok_button_emit_one_owned_ordered_unicode_list() {
    let mut fixture = ValueListWidgetFixture::new();
    fixture.frame(false, &[]);
    fixture.frame(true, &[]);
    fixture.frame(false, &[]);
    fixture.frame(
        false,
        &[crate::GuiInputEvent::Text(" é🙂,steel\nwood ".into())],
    );
    assert_eq!(fixture.editor.text, " é🙂,steel\nwood ");
    let actions = fixture.click_ok();
    let [GuiParameterAction::Choices(edit)] = actions.as_slice() else {
        panic!("expected one list edit: {actions:?}");
    };
    assert_eq!(
        edit.choices.expressions,
        Some(vec!["é🙂".into(), "steel".into(), "wood".into()])
    );
    assert!(fixture.click_ok().is_empty());
    assert_eq!(
        fixture
            .editor
            .acknowledge(edit, Err("parameter 0: expected an editable row".into())),
        GuiParameterAck::Failed
    );
    assert_eq!(fixture.editor.text, " é🙂,steel\nwood ");
    fixture.editor.close();
    assert!(fixture.editor.target.is_none());
}
#[test]
fn switching_list_documents_does_not_reuse_the_old_active_text_input() {
    let mut fixture = ValueListWidgetFixture::new();
    fixture.frame(false, &[]);
    fixture.frame(true, &[]);
    fixture.frame(false, &[]);
    fixture.frame(false, &[crate::GuiInputEvent::Text("old document".into())]);
    fixture.editor.open(8, &value_list_row()).unwrap();
    assert_eq!(fixture.editor.text, "é🙂\nsteel");
    fixture.frame(false, &[crate::GuiInputEvent::Text("leaked".into())]);
    assert_eq!(fixture.editor.text, "é🙂\nsteel");
}
#[test]
fn switching_list_parameters_does_not_reuse_the_old_active_text_input() {
    let mut fixture = ValueListWidgetFixture::new();
    fixture.frame(false, &[]);
    fixture.frame(true, &[]);
    fixture.frame(false, &[]);
    fixture.frame(false, &[crate::GuiInputEvent::Text("old parameter".into())]);
    let mut replacement = value_list_row();
    replacement.parameter = 11;
    fixture.editor.open(7, &replacement).unwrap();
    fixture.frame(false, &[crate::GuiInputEvent::Text("leaked".into())]);
    assert_eq!(fixture.editor.text, "é🙂\nsteel");
}
