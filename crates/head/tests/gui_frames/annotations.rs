// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn annotation_projection_preserves_other_labels_and_names_local_declines() {
    use oblikovati_gui::GuiDimensionLabelRole::Driving;
    let mut owner = FakeGuiOwner::default();
    owner.annotations.declines.push("unavailable plane".into());
    for position in [[3., 5.], [-1., 2.]] {
        owner
            .annotations
            .labels
            .push(crate::HeadDimensionAnnotation {
                plane: oblikovati_model::sketch::SketchPlane::xy(),
                position,
                line: Some([[1., 2.], [4., 5.]]),
                text: "d0 = 10 mm".into(),
                role: Driving,
                edit: None,
            });
    }
    let mut window = FakeGuiWindow::default();
    advance_gui(&mut window, &mut owner, None).unwrap();
    assert_eq!(
        window.states[0].dimension_labels,
        [oblikovati_gui::GuiDimensionLabel {
            pixel: [6, 15],
            line: Some([[2, 6], [8, 15]]),
            text: "d0 = 10 mm".into(),
            role: Driving,
            edit: None,
        }]
    );
    assert_eq!(window.notices.len(), 1);
    assert!(window.notices[0].message().contains("unavailable plane"));
    assert!(window.notices[0].message().contains("[-1.0, 2.0]"));
    assert!(owner.inputs.is_empty());
}

#[test]
fn unprojectable_dimension_line_reports_its_failure_without_hiding_the_label() {
    let mut owner = FakeGuiOwner::default();
    owner
        .annotations
        .labels
        .push(crate::HeadDimensionAnnotation {
            plane: oblikovati_model::sketch::SketchPlane::xy(),
            position: [3., 5.],
            line: Some([[-1., 2.], [4., 5.]]),
            text: "10".into(),
            role: oblikovati_gui::GuiDimensionLabelRole::Driving,
            edit: None,
        });
    let mut window = FakeGuiWindow::default();
    advance_gui(&mut window, &mut owner, None).unwrap();
    assert_eq!(window.states[0].dimension_labels[0].line, None);
    assert!(window.notices[0].message().contains("dimension line"));
}
