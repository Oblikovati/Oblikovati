// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn unavailable_interest_warning_is_retained_and_coalesced_until_document_close() {
    let mut observations = DesktopObservations::default();
    let clone = observations.create().unwrap();
    assert!(!clone.take_changed());
    observations.unavailable_interest_clients(4, "Part", &["addon".into()]);
    assert!(clone.take_changed());
    assert!(!clone.take_changed());
    assert_eq!(clone.missing_interests()[0].clients, ["addon"]);
    observations.unavailable_interest_clients(4, "Part renamed", &["other".into()]);
    assert_eq!(clone.missing_interests().len(), 1);
    assert_eq!(clone.missing_interests()[0].display_name, "Part renamed");
    observations.closed(4, None);
    assert!(clone.missing_interests().is_empty());
    assert!(clone.take_changed());
    assert!(!observations.interest_client_present("addon"));
}
