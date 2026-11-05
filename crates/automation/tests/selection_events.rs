// SPDX-License-Identifier: GPL-2.0-only
#[path = "support/selection_sink.rs"]
mod selection_sink;
use oblikovati_api::wire::{EVENT_SELECTION_CHANGED, SelectionChangedEvent};
use oblikovati_automation::EventRelay;
use selection_sink::RecordingEventSink;
#[test]
fn typed_relay_uses_the_released_event_shape_including_zero_and_large_counts() {
    let mut relay = EventRelay::new(RecordingEventSink::default());
    for count in [0, 1, 4096] {
        relay.selection_changed(count).unwrap();
    }
    let sink = relay.into_sink();
    assert_eq!(sink.attempts, 3);
    assert_eq!(
        sink.payloads[0],
        br#"{"type":"selection.changed","count":0}"#
    );
    for (payload, count) in sink.payloads.iter().zip([0, 1, 4096]) {
        let event: SelectionChangedEvent = serde_json::from_slice(payload).unwrap();
        assert_eq!(event.type_, EVENT_SELECTION_CHANGED);
        assert_eq!(event.count, count);
    }
}
#[test]
fn out_of_range_count_declines_before_transport_io() {
    let Some(count) = usize::try_from(i64::MAX)
        .ok()
        .and_then(|limit| limit.checked_add(1))
    else {
        return;
    };
    let mut relay = EventRelay::new(RecordingEventSink::default());
    let error = relay.selection_changed(count).unwrap_err().to_string();
    assert!(error.contains(&count.to_string()));
    assert!(error.contains("expected an API count in 0..="));
    let sink = relay.into_sink();
    assert_eq!(sink.attempts, 0);
    assert!(sink.payloads.is_empty());
}
#[test]
fn delivery_failure_names_the_event_and_can_retry_notification_without_replaying_input() {
    let mut relay = EventRelay::new(RecordingEventSink {
        remaining_refusals: 1,
        ..Default::default()
    });
    let error = relay.selection_changed(2).unwrap_err().to_string();
    assert!(error.contains(EVENT_SELECTION_CHANGED));
    assert!(error.contains("count: 2"));
    assert!(error.contains("outbound queue is closed"));
    relay.selection_changed(2).unwrap();
    let sink = relay.into_sink();
    assert_eq!(sink.attempts, 2);
    assert_eq!(sink.payloads.len(), 1);
}
