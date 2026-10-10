// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::wire::*;
use oblikovati_model::{history::DocumentSnapshot, identity::IdentityRegistration};
use std::path::Path;

struct RejectUnexpectedCalls;
impl TransactionObserver for RejectUnexpectedCalls {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        panic!("unexpected transaction")
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {
        panic!("unexpected transaction")
    }
    fn changed(&mut self, _: u64, _: u64) {
        panic!("unexpected transaction")
    }
}
impl WorkspaceObserver for RejectUnexpectedCalls {
    fn created(&mut self, _: u64, _: IdentityRegistration) {
        panic!("unexpected document")
    }
    fn path_changed(&mut self, _: u64, _: &Path, _: &Path) {
        panic!("unexpected path change")
    }
    fn before_open(&mut self, _: &Path) -> Result<(), String> {
        panic!("unexpected open")
    }
    fn before_save(&mut self, _: u64, _: &DocumentSnapshot) -> Result<(), String> {
        panic!("unexpected save")
    }
    fn saved(&mut self, _: u64) {
        panic!("unexpected save")
    }
    fn opened(&mut self, _: u64, _: IdentityRegistration) {
        panic!("unexpected open")
    }
    fn activated(&mut self, _: u64) {
        panic!("unexpected activation")
    }
    fn before_close(&mut self, _: u64, _: &DocumentSnapshot) -> Result<(), String> {
        panic!("unexpected close")
    }
    fn closed(&mut self, _: u64, _: Option<u64>) {
        panic!("unexpected close")
    }
}

#[test]
fn overlapping_query_and_edit_calls_fail_without_poisoning_the_host() {
    let mut workspace = WorkspaceSession::<RejectUnexpectedCalls, _>::new(RejectUnexpectedCalls);
    let host = SessionHost::new(&mut workspace);
    {
        let _active_call = host.workspace.borrow_mut();
        for method in [
            METHOD_PARAMETERS_GET_SETTINGS,
            METHOD_PARAMETERS_SET_SETTINGS,
            METHOD_TRANSACTION_BEGIN,
        ] {
            let error = host.call(method, b"{}").unwrap_err().to_string();
            assert!(error.contains(method));
            assert!(error.contains("reentry is refused"));
        }
    }
    let result = host
        .call(
            METHOD_UNITS_CONVERT,
            br#"{"value":1,"from":"in","to":"mm"}"#,
        )
        .unwrap();
    let result: ConvertUnitsResult = serde_json::from_slice(&result).unwrap();
    assert_eq!(result.value, 25.4);
}
