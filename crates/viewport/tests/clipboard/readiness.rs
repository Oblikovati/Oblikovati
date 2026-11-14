// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn owner_wait_uses_the_earliest_deadline_without_inventing_a_tick() {
    let now = Instant::now();
    let expiry = now + Duration::from_secs(2);
    assert_eq!(bounded_wait_timeout(None, None, now), None);
    assert_eq!(
        bounded_wait_timeout(None, Some(expiry), now),
        Some(Duration::from_secs(2))
    );
    assert_eq!(
        bounded_wait_timeout(Some(Duration::from_secs(1)), Some(expiry), now),
        Some(Duration::from_secs(1))
    );
    assert_eq!(
        bounded_wait_timeout(Some(Duration::from_secs(3)), Some(expiry), now),
        Some(Duration::from_secs(2))
    );
    assert_eq!(
        bounded_wait_timeout(None, Some(now), expiry),
        Some(Duration::ZERO)
    );
}
