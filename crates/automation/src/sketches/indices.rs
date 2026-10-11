// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::Error;
pub(super) fn geometric_wire_index(index: usize) -> Result<i64, Error> {
    constraint_wire_index(index, "geometric constraint")
}
pub(super) fn constraint_wire_index(index: usize, kind: &str) -> Result<i64, Error> {
    i64::try_from(index).map_err(|_| {
        Error(format!(
            "{kind} index {index}: expected a nonnegative i64 wire index"
        ))
    })
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn largest_local_index_is_checked_before_constraint_encoding() {
        let actual = geometric_wire_index(usize::MAX);
        let expected = i64::try_from(usize::MAX);
        match expected {
            Ok(index) => assert_eq!(actual.unwrap(), index),
            Err(_) => {
                let cause = actual.unwrap_err().to_string();
                assert!(cause.contains(&usize::MAX.to_string()));
                assert!(cause.contains("expected a nonnegative i64"));
            }
        }
    }
}
