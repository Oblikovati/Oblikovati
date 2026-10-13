// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::error::Error;
#[test]
fn naming_refusal_keeps_sketch_and_message_without_fabricating_native_cause() {
    let sketch = oblikovati_model::document_sketch::DocumentSketchId(12);
    let message = "prism face Start: expected exact write-time reference round trip";
    let failure = naming_decline(sketch, message.into());
    assert_eq!(failure.phase(), Phase::Naming);
    assert_eq!(failure.kind(), Kind::InvalidReference);
    assert_eq!(failure.entity(), &Entity::Body);
    assert_eq!(failure.sketch(), Some(sketch));
    assert!(failure.source().is_none());
    assert_eq!(failure.to_string(), message);
}
