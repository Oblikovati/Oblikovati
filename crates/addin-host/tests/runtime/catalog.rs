// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::client::Client;
use std::cell::RefCell;
#[derive(Default)]
struct RecordedModelCaller {
    calls: RefCell<Vec<(String, Vec<u8>)>>,
}
impl Caller for RecordedModelCaller {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        self.calls
            .borrow_mut()
            .push((method.into(), request.into()));
        Ok(b"model reply".to_vec())
    }
}
#[test]
fn catalog_queries_use_canonical_dtos_and_forward_other_requests_once() {
    let identity = AddInInfo {
        id: "catalog-entry".into(),
        ..Default::default()
    };
    let caller = NativeCatalogCaller {
        catalog: vec![identity.clone()],
        model: RecordedModelCaller::default(),
    };
    verify_catalog_queries(&caller, &identity);
    assert_eq!(
        caller.call("model-query", b"owned request").unwrap(),
        b"model reply"
    );
    assert_eq!(
        *caller.model.calls.borrow(),
        [("model-query".into(), b"owned request".to_vec())]
    );
}
fn verify_catalog_queries(caller: &NativeCatalogCaller<RecordedModelCaller>, identity: &AddInInfo) {
    let client = Client::new(caller);
    assert_eq!(
        client.add_ins().list().unwrap().add_ins.unwrap().as_slice(),
        std::slice::from_ref(identity)
    );
    assert_eq!(
        client
            .add_ins()
            .get(&AddInRefArgs {
                id: identity.id.clone()
            })
            .unwrap(),
        *identity
    );
    assert!(
        client
            .add_ins()
            .get(&AddInRefArgs {
                id: "missing".into()
            })
            .unwrap_err()
            .to_string()
            .contains("missing")
    );
    assert!(caller.call(METHOD_ADD_INS_GET, b"invalid request").is_err());
    assert!(caller.model.calls.borrow().is_empty());
}
