// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_viewport::ViewportPhysicalKey;

pub(super) fn create_size_parameter(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
) {
    open_parameter_ribbon(window, owner);
    click_parameter_input(window, owner, [235., 140.]);
    type_native_text(window, owner, "size");
    click_parameter_input(window, owner, [380., 140.]);
    type_native_text(window, owner, "10 mm");
    click_parameter_input(window, owner, [475., 140.]);
    assert_size_expression(owner, "10 mm");
    click_parameter_label(window, owner, "Done");
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
pub(super) fn edit_size_parameter(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    open_parameter_ribbon(window, owner);
    let (position, size) = application_bdd::window_geometry("Parameters").unwrap();
    let name = application_bdd::rendered_label_in_rect(
        "size",
        position,
        [position[0] + 200., position[1] + size[1]],
    );
    let equation_x = application_bdd::pointer_pixels([position[0] + 290., position[1]])[0];
    click_shell_position(window, owner, [equation_x, name[1]]);
    press_shell_key(window, owner, ViewportPhysicalKey::End);
    for _ in "10 mm".chars() {
        press_shell_key(window, owner, ViewportPhysicalKey::Backspace);
    }
    type_native_text(window, owner, "20 mm");
    click_parameter_label(window, owner, "Done");
    assert_size_expression(owner, "20 mm");
}
fn click_parameter_input(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    offset: [f32; 2],
) {
    let (position, _) = application_bdd::window_geometry("Parameters").unwrap();
    let pixels =
        application_bdd::pointer_pixels([position[0] + offset[0], position[1] + offset[1]]);
    click_shell_position(window, owner, pixels);
}
fn click_parameter_label(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    label: &str,
) {
    click_shell_position(window, owner, application_bdd::rendered_label_center(label));
}
pub(super) fn edit_size_and_history(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    edit: crate::HeadSketchEdit,
) {
    let keys = cube::topology_keys(owner);
    edit_size_parameter(window, owner);
    assert_eq!(cube::topology_keys(owner), keys);
    verify_rectangle(owner, edit, 2.);
    cube::verify(owner, edit.document, 2.);
    for (label, expected) in [("Undo", 1.), ("Redo", 2.)] {
        click_parameter_label(window, owner, label);
        verify_rectangle(owner, edit, expected);
        cube::verify(owner, edit.document, expected);
        assert_eq!(cube::topology_keys(owner), keys);
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
    use oblikovati_model::sketch::PlanarConstraint;
    let constraints = geometry.constraints();
    assert!(
        constraints
            .iter()
            .any(|constraint| matches!(constraint.relation, PlanarConstraint::Coincident { .. }))
    );
    assert_eq!(
        constraints
            .iter()
            .filter(|constraint| matches!(
                constraint.relation,
                PlanarConstraint::LineHorizontal { .. }
            ))
            .count(),
        2
    );
    assert_eq!(
        constraints
            .iter()
            .filter(|constraint| matches!(
                constraint.relation,
                PlanarConstraint::LineVertical { .. }
            ))
            .count(),
        2
    );
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
    let parameters = application_bdd::toolbar_icon_center("Parameters");
    click_shell_position(window, owner, parameters);
    assert!(
        application_bdd::window_geometry("Parameters").is_some(),
        "Parameters toolbar command must open its window"
    );
}
