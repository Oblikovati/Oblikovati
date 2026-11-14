// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn group_row() -> GuiParameterRow {
    GuiParameterRow {
        parameter: 0,
        detail: Default::default(),
        tolerance_text: String::new(),
        has_groups: false,
    }
}
#[test]
fn group_editor_trims_names_keeps_zero_identity_and_publishes_one_owned_change() {
    let mut editor = ParameterGroupEditor::default();
    editor.open(7, &group_row());
    editor.name = "  ratios é🙂  ".into();
    let edit = editor.submit().unwrap().unwrap();
    assert_eq!(edit.target.parameter, 0);
    assert_eq!(&*edit.key.0, "ratios é🙂");
    assert!(editor.submit().unwrap().is_none());
    assert_eq!(editor.acknowledge(&edit, Ok(())), GuiParameterAck::Applied);
    assert!(editor.target.is_none());
}
#[test]
fn group_failures_keep_names_and_stale_results_cannot_close_a_different_target() {
    let mut editor = ParameterGroupEditor::default();
    editor.open(7, &group_row());
    editor.name = "ratios".into();
    let failed = editor.submit().unwrap().unwrap();
    assert_eq!(
        editor.acknowledge(
            &failed,
            Err("parameter 0: expected a registered parameter".into())
        ),
        GuiParameterAck::Failed
    );
    assert_eq!(editor.name, "ratios");
    assert!(editor.request.issue.is_some());
    editor.open(8, &group_row());
    editor.name = "new".into();
    let current = editor.submit().unwrap().unwrap();
    assert!(current.request > failed.request);
    assert_eq!(
        editor.acknowledge(&failed, Ok(())),
        GuiParameterAck::Obsolete
    );
    assert!(editor.target.is_some());
}
#[test]
fn group_inputs_and_packet_shapes_are_checked_before_request_admission() {
    let mut editor = ParameterGroupEditor::default();
    assert!(editor.submit().is_err());
    editor.open(7, &group_row());
    for name in ["  ", "bad\0group"] {
        editor.name = name.into();
        assert!(editor.submit().is_err());
        assert!(editor.request.pending.is_none());
    }
    editor.name = "ratios".into();
    let valid = editor.submit().unwrap().unwrap();
    let mut invalid = valid.clone();
    invalid.request = 0;
    assert!(invalid.validate().is_err());
    invalid = valid.clone();
    invalid.target.document = 0;
    assert!(invalid.validate().is_err());
    invalid = valid;
    invalid.target.field = GuiParameterTextField::Equation;
    assert!(invalid.validate().is_err());
}
#[test]
fn group_editors_retire_targets_for_deleted_parameters_and_closed_documents() {
    let mut editor = ParameterGroupEditor::default();
    let snapshot = GuiParameterSnapshot {
        linked_tables: Vec::new(),
        link_sources: Vec::new(),
        groups: Vec::new(),
        document: 7,
        rows: vec![group_row()],
    };
    editor.open(7, &snapshot.rows[0]);
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
    editor.open(7, &group_row());
    editor.synchronize(None);
    assert!(editor.target.is_none());
}
#[test]
fn group_enqueue_keeps_local_errors_and_uses_the_common_request_lifecycle() {
    let mut editor = ParameterGroupEditor::default();
    let mut actions = Vec::new();
    editor.enqueue(&mut actions);
    assert!(editor.request.issue.is_some());
    editor.open(7, &group_row());
    editor.name = "ratios".into();
    editor.enqueue(&mut actions);
    editor.enqueue(&mut actions);
    assert!(matches!(actions.as_slice(), [GuiParameterAction::Group(_)]));
}
struct GroupWidgetFixture {
    context: imgui::Context,
    editor: ParameterGroupEditor,
    ok_position: [f32; 2],
}
impl GroupWidgetFixture {
    fn new() -> Self {
        let mut context = imgui::Context::create();
        context.set_ini_filename(None);
        context.io_mut().display_size = [500., 300.];
        context.io_mut().delta_time = 1. / 60.;
        context.fonts().build_rgba32_texture();
        let mut editor = ParameterGroupEditor::default();
        editor.open(7, &group_row());
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
        ui.window("Group widget probe")
            .position([8., 8.], Condition::Always)
            .size([400., 230.], Condition::Always)
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
    fn type_name(&mut self, name: &str) {
        self.frame(false, &[]);
        self.frame(true, &[]);
        self.frame(false, &[]);
        self.frame(false, &[crate::GuiInputEvent::Text(name.into())]);
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
fn real_group_name_and_ok_button_emit_one_owned_trimmed_unicode_key() {
    let mut fixture = GroupWidgetFixture::new();
    fixture.type_name("  ratios é🙂  ");
    let actions = fixture.click_ok();
    let [GuiParameterAction::Group(edit)] = actions.as_slice() else {
        panic!("expected one group change: {actions:?}");
    };
    assert_eq!(&*edit.key.0, "ratios é🙂");
    assert!(fixture.click_ok().is_empty());
}
#[test]
fn group_document_and_parameter_switches_do_not_reuse_old_active_input() {
    for (document, parameter) in [(8, 0), (7, 11)] {
        let mut fixture = GroupWidgetFixture::new();
        fixture.type_name("old group");
        let mut row = group_row();
        row.parameter = parameter;
        fixture.editor.open(document, &row);
        fixture.frame(false, &[crate::GuiInputEvent::Text("leaked".into())]);
        assert!(fixture.editor.name.is_empty());
    }
}
