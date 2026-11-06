// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn selected_part_templates_create_real_documents_with_the_selected_units() {
    let (mut owner, _) = owner();
    for template in [
        oblikovati_gui::GuiPartTemplate::Metric,
        oblikovati_gui::GuiPartTemplate::Inch,
    ] {
        assert!(owner.create_part_template(template).unwrap());
        let client = Client::new(owner.document_host());
        let units = client.documents().get_units().unwrap();
        assert_eq!(units.length_unit, template.length_unit());
        let state = owner.gui_command_state().unwrap();
        assert_eq!(
            state.document_type,
            Some(oblikovati_api::types::DOCUMENT_PART)
        );
        assert!(state.browser.as_ref().unwrap().has_origin);
        assert!(state.browser.as_ref().unwrap().features.is_empty());
    }
}
