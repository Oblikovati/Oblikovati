// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{GuiTemplateMaterial, GuiTemplateMaterialEntry};

#[test]
fn chooser_distinguishes_material_assignment_absence_and_resolution_failure() {
    let template = GuiDocumentTemplate::default();
    let mut state = GuiCommandState::default();
    assert_eq!(
        material_label(&state, template).as_deref(),
        Some("Material: unavailable")
    );
    state.template_materials.push(GuiTemplateMaterialEntry {
        template,
        material: GuiTemplateMaterial::Unassigned,
    });
    assert_eq!(
        material_label(&state, template).as_deref(),
        Some("Material: Unassigned")
    );
    state.template_materials[0].material = GuiTemplateMaterial::Assigned {
        id: "steel".into(),
        name: "Steel".into(),
    };
    assert_eq!(
        material_label(&state, template).as_deref(),
        Some("Material: Steel")
    );
    state.template_materials[0].material = GuiTemplateMaterial::Unresolved { id: "lost".into() };
    assert_eq!(
        material_label(&state, template).as_deref(),
        Some("Material unavailable: lost")
    );
    state.template_materials[0].material = GuiTemplateMaterial::Unavailable {
        cause: "material \"fixture-steel\": fixture catalog unavailable".into(),
    };
    assert_eq!(
        material_label(&state, template).as_deref(),
        Some("Material unavailable: material \"fixture-steel\": fixture catalog unavailable")
    );
    let drawing = GuiDocumentTemplate {
        kind: GuiTemplateKind::Drawing,
        ..template
    };
    assert_eq!(material_label(&state, drawing), None);
}
