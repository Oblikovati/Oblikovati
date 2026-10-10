// SPDX-License-Identifier: GPL-2.0-only

use super::PackageCodecError;
use std::time::{Duration, SystemTime};
use time::{OffsetDateTime, format_description::well_known::Rfc3339};

// Go time.Time{} is 0001-01-01 UTC, the unknown-time sentinel (#3690).
const GO_ZERO_UNIX_NANOS: i128 = -62_135_596_800_000_000_000;

pub(super) fn decode(text: &str) -> Result<Option<SystemTime>, PackageCodecError> {
    if text.is_empty() {
        return Ok(None);
    }
    reject_lossy_time(text)?;
    let parsed = OffsetDateTime::parse(text, &Rfc3339).map_err(|cause| error(text, cause))?;
    let nanos = parsed.unix_timestamp_nanos();
    if nanos == GO_ZERO_UNIX_NANOS {
        return Ok(None);
    }
    let magnitude = nanos.unsigned_abs();
    let seconds = u64::try_from(magnitude / 1_000_000_000).map_err(|cause| error(text, cause))?;
    let duration = Duration::new(seconds, (magnitude % 1_000_000_000) as u32);
    let stamp = if nanos < 0 {
        SystemTime::UNIX_EPOCH.checked_sub(duration)
    } else {
        SystemTime::UNIX_EPOCH.checked_add(duration)
    };
    stamp
        .map(Some)
        .ok_or_else(|| error(text, "timestamp is outside SystemTime range"))
}

fn reject_lossy_time(text: &str) -> Result<(), PackageCodecError> {
    let digits = text
        .get(19..)
        .and_then(|tail| tail.strip_prefix('.'))
        .map_or(0, |tail| {
            tail.bytes().take_while(u8::is_ascii_digit).count()
        });
    // time 0.3.55 maps leap seconds to the preceding nanosecond and truncates
    // excess fractional digits; file metadata must not silently change (#3690).
    if text.get(17..19) == Some("60") || digits > 9 {
        return Err(error(
            text,
            "expected seconds 00..59 and at most nine fractional digits",
        ));
    }
    Ok(())
}

/// Format captured file time for storage and API output without losing nanoseconds.
/// Example: `format_attachment_time(None)` returns an empty string.
pub fn format_attachment_time(stamp: Option<SystemTime>) -> Result<String, PackageCodecError> {
    let Some(stamp) = stamp else {
        return Ok(String::new());
    };
    let (duration, sign) = match stamp.duration_since(SystemTime::UNIX_EPOCH) {
        Ok(duration) => (duration, 1),
        Err(error) => (error.duration(), -1),
    };
    let nanos = i128::try_from(duration.as_nanos())
        .map_err(|cause| error(&format!("{stamp:?}"), cause))?
        * sign;
    if nanos == GO_ZERO_UNIX_NANOS {
        return Err(error(
            &format!("{stamp:?}"),
            "Go zero-time sentinel: expected None for an unknown attachment time",
        ));
    }
    let parsed = OffsetDateTime::from_unix_timestamp_nanos(nanos)
        .map_err(|cause| error(&format!("{stamp:?}"), cause))?;
    parsed
        .format(&Rfc3339)
        .map_err(|cause| error(&format!("{stamp:?}"), cause))
}

fn error(value: &str, cause: impl std::fmt::Display) -> PackageCodecError {
    PackageCodecError(format!(
        "attachment timestamp {value:?}: expected an exact RFC 3339 nanosecond timestamp: {cause}"
    ))
}
