// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn creation_forms_keep_each_value_family_and_reset_only_after_matching_success() {
    let mut form = ParameterCreationForm::default();
    form.synchronize(Some(7));
    for (kind, expected) in [
        (0, GuiParameterInitialValue::Numeric("4 cm".into())),
        (1, GuiParameterInitialValue::Text("4 cm".into())),
        (2, GuiParameterInitialValue::Boolean(true)),
    ] {
        form.kind = kind;
        form.name = "parameter".into();
        form.expression = "4 cm".into();
        form.boolean = true;
        let creation = form.submit().unwrap().unwrap();
        assert_eq!(creation.value, expected);
        assert_eq!(form.submit().unwrap(), None);
        assert_eq!(
            form.acknowledge(&creation, Ok(())),
            GuiParameterAck::Applied
        );
        assert!(form.name.is_empty());
        assert!(form.expression.is_empty());
        assert!(form.boolean);
    }
}
#[test]
fn creation_failures_keep_authored_values_and_stale_responses_cannot_clear_another_document() {
    let mut form = ParameterCreationForm::default();
    assert!(form.submit().is_err());
    form.synchronize(Some(7));
    form.name = "width".into();
    form.expression = "(".into();
    let failed = form.submit().unwrap().unwrap();
    assert_eq!(
        form.acknowledge(&failed, Err("formula (: expected an expression".into())),
        GuiParameterAck::Failed
    );
    assert_eq!(form.expression, "(");
    form.synchronize(Some(8));
    form.name = "material".into();
    form.kind = 1;
    let current = form.submit().unwrap().unwrap();
    assert!(current.request > failed.request);
    assert_eq!(form.acknowledge(&failed, Ok(())), GuiParameterAck::Obsolete);
    assert_eq!(form.name, "material");
    assert_eq!(form.request.pending.as_ref(), Some(&current));
    let mut forged = current.clone();
    forged.name = "forged".into();
    assert_eq!(form.acknowledge(&forged, Ok(())), GuiParameterAck::Obsolete);
    form.synchronize(None);
    assert_eq!(
        form.acknowledge(&current, Ok(())),
        GuiParameterAck::Obsolete
    );
}
#[test]
fn creation_limits_refuse_invalid_text_and_exhausted_requests_before_submission() {
    let mut form = ParameterCreationForm::default();
    form.synchronize(Some(7));
    for source in ["bad\0name".to_owned(), "x".repeat(1024 * 1024 + 1)] {
        form.name = source;
        assert!(form.submit().is_err());
        assert!(form.request.pending.is_none());
    }
    form.name = "width".into();
    form.request.last_request = u64::MAX;
    assert!(form.submit().unwrap_err().contains("18446744073709551615"));
    assert!(form.request.pending.is_none());
}
#[test]
fn creation_enqueue_keeps_local_declines_and_publishes_owned_packets_once() {
    let mut form = ParameterCreationForm::default();
    let mut actions = Vec::new();
    form.enqueue(&mut actions);
    assert!(form.request.issue.is_some());
    assert!(actions.is_empty());
    form.synchronize(Some(7));
    form.name = "material".into();
    form.kind = 1;
    form.expression = "é🙂".into();
    form.enqueue(&mut actions);
    form.enqueue(&mut actions);
    let [GuiParameterAction::Add(creation)] = actions.as_slice() else {
        panic!("expected one creation: {actions:?}");
    };
    assert_eq!(creation.value, GuiParameterInitialValue::Text("é🙂".into()));
    assert!(form.request.issue.is_none());
}
struct ParameterCreationWidgetFixture {
    context: imgui::Context,
    form: ParameterCreationForm,
    add_position: [f32; 2],
}
impl ParameterCreationWidgetFixture {
    fn new() -> Self {
        let mut context = imgui::Context::create();
        context.set_ini_filename(None);
        context.io_mut().display_size = [800., 300.];
        context.io_mut().delta_time = 1. / 60.;
        context.fonts().build_rgba32_texture();
        let mut form = ParameterCreationForm::default();
        form.synchronize(Some(7));
        Self {
            context,
            form,
            add_position: [0.; 2],
        }
    }
    fn frame(
        &mut self,
        focus: Option<usize>,
        events: &[crate::GuiInputEvent],
    ) -> Vec<GuiParameterAction> {
        for event in events {
            crate::input::apply_input(self.context.io_mut(), event, [1.; 2]);
        }
        let ui = self.context.frame();
        let mut actions = Vec::new();
        ui.window("Parameter creation probe")
            .position([8., 8.], imgui::Condition::Always)
            .size([740., 180.], imgui::Condition::Always)
            .build(|| {
                if let Some(offset) = focus {
                    ui.set_keyboard_focus_here_with_offset(imgui::FocusedWidget::Offset(
                        offset as u32,
                    ));
                }
                self.form.draw(ui, &mut actions);
                let minimum = ui.item_rect_min();
                let maximum = ui.item_rect_max();
                self.add_position =
                    std::array::from_fn(|axis| (minimum[axis] + maximum[axis]) / 2.);
            });
        self.context.render();
        actions
    }
    fn type_field(&mut self, offset: usize, value: &str) {
        self.frame(Some(offset), &[]);
        self.frame(None, &[]);
        self.frame(None, &[crate::GuiInputEvent::Text(value.into())]);
    }
    fn click_add(&mut self) -> Vec<GuiParameterAction> {
        use crate::{GuiInputEvent, GuiPointerButton};
        self.frame(None, &[GuiInputEvent::PointerMoved(self.add_position)]);
        self.frame(
            None,
            &[GuiInputEvent::PointerButton {
                button: GuiPointerButton::Left,
                pressed: true,
            }],
        );
        self.frame(
            None,
            &[GuiInputEvent::PointerButton {
                button: GuiPointerButton::Left,
                pressed: false,
            }],
        )
    }
}
#[test]
fn real_creation_fields_and_add_button_emit_owned_unicode_values_once() {
    let mut fixture = ParameterCreationWidgetFixture::new();
    fixture.form.kind = 1;
    fixture.frame(None, &[]);
    fixture.type_field(1, "material");
    fixture.type_field(2, "é🙂");
    assert_eq!(fixture.form.name, "material");
    assert_eq!(fixture.form.expression, "é🙂");
    let actions = fixture.click_add();
    let [GuiParameterAction::Add(creation)] = actions.as_slice() else {
        panic!("expected one creation: {actions:?}");
    };
    assert_eq!(creation.name, "material");
    assert_eq!(creation.value, GuiParameterInitialValue::Text("é🙂".into()));
    assert!(fixture.click_add().is_empty());
    assert_eq!(
        fixture.form.acknowledge(creation, Ok(())),
        GuiParameterAck::Applied
    );
    fixture.form.kind = 2;
    fixture.frame(None, &[]);
    fixture.type_field(1, "vented");
    let actions = fixture.click_add();
    let [GuiParameterAction::Add(creation)] = actions.as_slice() else {
        panic!("expected one Boolean creation: {actions:?}");
    };
    assert_eq!(creation.value, GuiParameterInitialValue::Boolean(false));
}
#[test]
fn switching_creation_documents_does_not_reuse_the_old_active_name_input() {
    let mut fixture = ParameterCreationWidgetFixture::new();
    fixture.frame(None, &[]);
    fixture.type_field(1, "old_name");
    fixture.form.synchronize(Some(8));
    assert!(fixture.form.name.is_empty());
    fixture.frame(None, &[crate::GuiInputEvent::Text("leaked".into())]);
    assert!(fixture.form.name.is_empty());
    assert!(fixture.form.expression.is_empty());
}
