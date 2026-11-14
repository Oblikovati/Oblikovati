// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn group_identity_is_scoped_and_survives_only_its_own_nested_levels() {
    let mut owner = session(None);
    assert!(owner.transaction_identity().is_none());
    owner.begin("Line").unwrap();
    let line = owner.transaction_identity().unwrap();
    owner.begin("Nested").unwrap();
    assert_eq!(owner.transaction_identity(), Some(line.clone()));
    owner.end().unwrap();
    assert_eq!(owner.transaction_identity(), Some(line.clone()));
    owner.end().unwrap();
    assert!(owner.transaction_identity().is_none());
    owner.begin("Caller").unwrap();
    assert_ne!(owner.transaction_identity(), Some(line));
    let mut other = session(None);
    other.begin("Caller").unwrap();
    assert_ne!(owner.transaction_identity(), other.transaction_identity());
    owner.abort().unwrap();
    assert!(owner.transaction_identity().is_none());
}
