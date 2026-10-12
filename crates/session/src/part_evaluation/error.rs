// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::feature::part::PartFeatureProgramError;
use std::{error::Error, fmt};
/// Rejected results never replace the currently published generation.
#[derive(Debug, PartialEq)]
pub enum PartEvaluationError {
    Replay(PartFeatureProgramError),
    Unavailable {
        document: u64,
    },
    ForeignDocument {
        supplied: u64,
        expected: u64,
    },
    ForeignHistory {
        document: u64,
        generation: u64,
    },
    Superseded {
        document: u64,
        supplied: u64,
        expected: u64,
    },
}
impl fmt::Display for PartEvaluationError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Replay(cause) => cause.fmt(output),
            Self::Unavailable { document } => write_missing_result(*document, output),
            Self::ForeignDocument { supplied, expected } => write!(
                output,
                "part result document {supplied}: expected document {expected}"
            ),
            Self::ForeignHistory {
                document,
                generation,
            } => write_history_scope(*document, *generation, output),
            Self::Superseded {
                document,
                supplied,
                expected,
            } => write_generation_scope(*document, *supplied, *expected, output),
        }
    }
}
impl Error for PartEvaluationError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Replay(cause) => Some(cause),
            _ => None,
        }
    }
}

fn write_history_scope(
    document: u64,
    generation: u64,
    output: &mut fmt::Formatter<'_>,
) -> fmt::Result {
    write!(
        output,
        "part result document {document} generation {generation}: expected the current session history owner"
    )
}
fn write_generation_scope(
    document: u64,
    supplied: u64,
    expected: u64,
    output: &mut fmt::Formatter<'_>,
) -> fmt::Result {
    write!(
        output,
        "part result document {document} generation {supplied}: expected current generation {expected}"
    )
}

fn write_missing_result(document: u64, output: &mut fmt::Formatter<'_>) -> fmt::Result {
    write!(
        output,
        "document {document} part result is absent: expected an adopted current generation"
    )
}
