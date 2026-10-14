// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn editor_requests_require_one_current_identity_and_the_exact_owned_packet() {
    let mut state = ParameterEditRequest::<String>::default();
    assert_eq!(state.prepare().unwrap(), 1);
    assert!(state.publish(2, "wrong".into()).is_err());
    assert!(state.pending.is_none());
    state.publish(1, "owned".into()).unwrap();
    assert!(state.prepare().is_err());
    assert_eq!(
        state.acknowledge(&"different".into(), Ok(())),
        GuiParameterAck::Obsolete
    );
    assert_eq!(state.pending.as_deref(), Some("owned"));
    assert_eq!(
        state.acknowledge(&"owned".into(), Err("declined".into())),
        GuiParameterAck::Failed
    );
    assert_eq!(state.issue.as_deref(), Some("declined"));
    state.publish(2, "retry".into()).unwrap();
    assert!(state.issue.is_none());
    assert_eq!(
        state.acknowledge(&"retry".into(), Ok(())),
        GuiParameterAck::Applied
    );
    assert!(state.pending.is_none());
}
#[test]
fn retirement_keeps_the_counter_and_exhaustion_is_a_named_decline() {
    let mut state = ParameterEditRequest::<String>::default();
    state.publish(1, "owned".into()).unwrap();
    state.retire();
    assert!(state.pending.is_none());
    assert_eq!(state.prepare().unwrap(), 2);
    assert_eq!(
        state.acknowledge(&"owned".into(), Ok(())),
        GuiParameterAck::Obsolete
    );
    state.last_request = u64::MAX;
    assert!(
        state
            .prepare()
            .unwrap_err()
            .contains("18446744073709551615")
    );
}
