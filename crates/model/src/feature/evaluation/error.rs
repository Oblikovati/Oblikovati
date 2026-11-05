// SPDX-License-Identifier: GPL-2.0-only

use crate::feature::{FeatureId, FeatureReplayRange};
use std::fmt::{Display, Formatter};

/// Typed offending and required values for a rejected replay prefix.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct FeatureReplayMismatch<Value> {
    pub actual: Value,
    pub expected: Value,
}

/// Protocol errors never advance replay or replace recorded health.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum FeatureReplayError {
    Range {
        range: FeatureReplayRange,
        cutoff: usize,
    },
    PrefixLength {
        length: usize,
        start: usize,
    },
    PrefixIdentity {
        index: usize,
        ids: FeatureReplayMismatch<FeatureId>,
    },
    PrefixStatus {
        feature: FeatureId,
        statuses: FeatureReplayMismatch<&'static str>,
    },
    PrefixDrivers {
        feature: FeatureId,
        drivers: FeatureReplayMismatch<Vec<FeatureId>>,
    },
    Pending(FeatureId),
    ParametersRequired(FeatureId),
    Footprint {
        feature: FeatureId,
        missing: Vec<crate::feature::FeatureReadKey>,
    },
    ResultIdentity {
        feature: FeatureId,
        expected: Option<FeatureId>,
    },
    Incomplete {
        next: usize,
        end: usize,
        pending: Option<FeatureId>,
    },
}

impl Display for FeatureReplayError {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Range { range, cutoff } => range_message(f, *range, *cutoff),
            Self::PrefixLength { length, start } => length_message(f, *length, *start),
            Self::PrefixIdentity { index, ids } => identity_message(f, *index, ids),
            Self::PrefixStatus { feature, statuses } => status_message(f, *feature, statuses),
            Self::PrefixDrivers { feature, drivers } => drivers_message(f, *feature, drivers),
            Self::Pending(feature) => pending_message(f, *feature),
            Self::ParametersRequired(feature) => write!(
                f,
                "conditional feature {feature:?}: expected immutable parameter model inputs"
            ),
            Self::Footprint { feature, missing } => footprint_message(f, *feature, missing),
            Self::ResultIdentity { feature, expected } => result_message(f, *feature, *expected),
            Self::Incomplete { next, end, pending } => {
                incomplete_message(f, (*next, *end, *pending))
            }
        }
    }
}

fn range_message(
    f: &mut Formatter<'_>,
    range: FeatureReplayRange,
    cutoff: usize,
) -> std::fmt::Result {
    write!(
        f,
        "replay range {range:?}: expected start <= end and end = cutoff {cutoff}"
    )
}

fn length_message(f: &mut Formatter<'_>, length: usize, start: usize) -> std::fmt::Result {
    write!(
        f,
        "prefix health length {length}: expected replay start {start}"
    )
}

fn identity_message(
    f: &mut Formatter<'_>,
    index: usize,
    ids: &FeatureReplayMismatch<FeatureId>,
) -> std::fmt::Result {
    let FeatureReplayMismatch {
        actual: feature,
        expected,
    } = ids;
    write!(
        f,
        "prefix index {index} feature {feature:?}: expected {expected:?}"
    )
}

fn status_message(
    f: &mut Formatter<'_>,
    feature: FeatureId,
    statuses: &FeatureReplayMismatch<&str>,
) -> std::fmt::Result {
    let FeatureReplayMismatch {
        actual: observed,
        expected,
    } = statuses;
    write!(
        f,
        "prefix feature {feature:?} status {observed}: expected {expected}"
    )
}

fn drivers_message(
    f: &mut Formatter<'_>,
    feature: FeatureId,
    causes: &FeatureReplayMismatch<Vec<FeatureId>>,
) -> std::fmt::Result {
    let FeatureReplayMismatch {
        actual: drivers,
        expected,
    } = causes;
    write!(
        f,
        "prefix feature {feature:?} blocked drivers {drivers:?}: expected {expected:?}"
    )
}

fn pending_message(f: &mut Formatter<'_>, feature: FeatureId) -> std::fmt::Result {
    write!(
        f,
        "pending feature {feature:?}: expected its validated result before advancing"
    )
}

fn result_message(
    f: &mut Formatter<'_>,
    feature: FeatureId,
    expected: Option<FeatureId>,
) -> std::fmt::Result {
    write!(
        f,
        "result feature {feature:?}: expected pending feature {expected:?}"
    )
}

fn incomplete_message(
    f: &mut Formatter<'_>,
    (next, end, pending): (usize, usize, Option<FeatureId>),
) -> std::fmt::Result {
    write!(
        f,
        "replay next {next}, pending {pending:?}: expected completed cutoff {end} with no pending feature"
    )
}

impl std::error::Error for FeatureReplayError {}

fn footprint_message(
    f: &mut Formatter<'_>,
    feature: FeatureId,
    missing: &[crate::feature::FeatureReadKey],
) -> std::fmt::Result {
    write!(
        f,
        "feature {feature:?} missing reads {missing:?}: expected every definition read in its footprint"
    )
}
