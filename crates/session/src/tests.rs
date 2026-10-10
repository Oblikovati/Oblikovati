// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::wire::TransactionEventPayload;
use oblikovati_model::{
    identity::RecipeIdAllocator, parameter::ParameterCollection, property::DocumentProperties,
    units::UnitsOfMeasure,
};

struct RejectUnexpectedObservation;
impl TransactionObserver for RejectUnexpectedObservation {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        panic!("expected no transition during depth validation")
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {
        panic!("expected no observation during depth validation")
    }
    fn changed(&mut self, _: u64, _: u64) {
        panic!("expected no change during depth validation")
    }
}

#[test]
fn nesting_overflow_preserves_group_and_is_visible() {
    let snapshot = DocumentSnapshot {
        attributes: Default::default(),
        parameters: ParameterCollection::new(UnitsOfMeasure::default()),
        properties: DocumentProperties::default(),
    };
    let history = DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).unwrap();
    let mut session = DocumentSession::new(1, history, RejectUnexpectedObservation);
    session.begin("outer").unwrap();
    session.group.as_mut().unwrap().depth = u64::MAX;
    let error = session.begin("overflow").unwrap_err();
    assert_eq!(error, SessionError::DepthExhausted);
    assert!(error.to_string().contains("u64::MAX"));
    assert_eq!(session.group.as_ref().unwrap().depth, u64::MAX);
    assert_eq!(session.group.as_ref().unwrap().label, "outer");
    assert_eq!(session.generation(), 0);
}
