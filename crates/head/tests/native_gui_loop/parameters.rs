// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_viewport::ViewportPhysicalKey;

pub(super) fn create_size_parameter(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
) {
    open_parameter_ribbon(window, owner);
    click_shell_position(window, owner, [400, 43]);
    type_native_text(window, owner, "size");
    click_shell_position(window, owner, [400, 66]);
    type_native_text(window, owner, "10 mm");
    click_shell_position(window, owner, [345, 90]);
    assert_size_expression(owner, "10 mm");
    click_shell_position(window, owner, [440, 90]);
    click_shell_position(window, owner, [250, 44]);
}
fn assert_size_expression(owner: &NativeDesktopOwner, expression: &str) {
    let state = owner.gui_command_state().unwrap();
    let parameter = state
        .parameters
        .iter()
        .find(|parameter| parameter.name == "size")
        .unwrap();
    assert_eq!(parameter.expression, expression);
}
fn edit_size_parameter(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    open_parameter_ribbon(window, owner);
    click_shell_position(window, owner, [350, 137]);
    click_shell_position(window, owner, [400, 66]);
    for _ in "10 mm".chars() {
        press_shell_key(window, owner, ViewportPhysicalKey::Backspace);
    }
    type_native_text(window, owner, "20 mm");
    click_shell_position(window, owner, [380, 90]);
    assert_size_expression(owner, "20 mm");
    click_shell_position(window, owner, [440, 90]);
}
pub(super) fn edit_size_and_history(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    edit: crate::HeadSketchEdit,
) {
    edit_size_parameter(window, owner);
    verify_rectangle(owner, edit, 2.);
    for (button, expected) in [(75, 1.), (120, 2.)] {
        click_shell(window, owner, button);
        verify_rectangle(owner, edit, expected);
    }
}
pub(super) fn verify_rectangle(
    owner: &NativeDesktopOwner,
    edit: crate::HeadSketchEdit,
    expected: f64,
) {
    let session = owner
        .documents()
        .workspace()
        .session(edit.document)
        .unwrap();
    let structure = session.analyze_sketch(edit.sketch).unwrap();
    assert_eq!(structure.degrees_of_freedom, 0);
    let geometry = session
        .snapshot()
        .sketches
        .get(edit.sketch)
        .unwrap()
        .geometry();
    assert_eq!(geometry.dimensional_constraints().count(), 2);
    for (constraint, parameter) in geometry.dimensional_constraints() {
        assert_eq!(
            session.snapshot().parameters.get(parameter).unwrap().source,
            "size"
        );
        assert!(!geometry.dimension_is_reference(constraint.id).unwrap());
        assert!((geometry.measure_dimension(constraint.id).unwrap().value - expected).abs() < 1e-9);
    }
}

fn open_parameter_ribbon(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    click_shell_position(window, owner, [350, 44]);
    click_shell_position(window, owner, [50, 90]);
}
