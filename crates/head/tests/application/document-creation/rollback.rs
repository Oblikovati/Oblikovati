// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{
    DocumentInfo, METHOD_DOCUMENTS_ACTIVATE, METHOD_DOCUMENTS_CLOSE, METHOD_DOCUMENTS_CREATE,
    METHOD_DOCUMENTS_SET_UNITS,
};
use std::{cell::RefCell, rc::Rc};
/// A host whose unit change fails, recording what the rollback then asks of it.
#[derive(Clone, Default)]
struct UnitRefusingHost {
    methods: Rc<RefCell<Vec<String>>>,
}
impl Caller for UnitRefusingHost {
    fn call(&self, method: &str, _: &[u8]) -> Result<Vec<u8>, oblikovati_api::Error> {
        self.methods.borrow_mut().push(method.to_string());
        if method == METHOD_DOCUMENTS_SET_UNITS {
            return Err(oblikovati_api::Error("units refused".into()));
        }
        let created = DocumentInfo {
            id: 9,
            ..Default::default()
        };
        Ok(serde_json::to_vec(&created).unwrap())
    }
}
fn metric_part() -> GuiDocumentTemplate {
    GuiDocumentTemplate {
        kind: oblikovati_gui::GuiTemplateKind::Part,
        units: oblikovati_gui::GuiTemplateUnits::Metric,
    }
}
#[test]
fn a_refused_unit_closes_the_new_document_and_returns_to_the_previous_one() {
    let client = Client::new(UnitRefusingHost::default());
    let refusal = create_template(&client, metric_part(), "Part1", Some(4)).unwrap_err();
    assert!(refusal.contains("units refused"), "refusal {refusal:?}");
    assert!(
        refusal.contains("mm"),
        "refusal {refusal:?}: expected the unit named"
    );
}
#[test]
fn the_rollback_runs_in_order_and_skips_activation_with_no_previous_document() {
    let host = UnitRefusingHost::default();
    let client = Client::new(host.clone());
    assert!(create_template(&client, metric_part(), "Part1", Some(4)).is_err());
    assert_eq!(
        *host.methods.borrow(),
        [
            METHOD_DOCUMENTS_CREATE,
            METHOD_DOCUMENTS_SET_UNITS,
            METHOD_DOCUMENTS_CLOSE,
            METHOD_DOCUMENTS_ACTIVATE,
        ]
    );
    host.methods.borrow_mut().clear();
    assert!(create_template(&client, metric_part(), "Part2", None).is_err());
    assert_eq!(
        *host.methods.borrow(),
        [
            METHOD_DOCUMENTS_CREATE,
            METHOD_DOCUMENTS_SET_UNITS,
            METHOD_DOCUMENTS_CLOSE,
        ]
    );
}
