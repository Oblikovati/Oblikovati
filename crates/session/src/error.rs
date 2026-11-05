// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::{
    history::{EditError, HistoryError},
    property::PropertyError,
};
use std::{error::Error, fmt};

/// A rejected session transition, including the observer's veto reason.
#[derive(Debug, PartialEq)]
pub enum SessionError {
    History(HistoryError),
    NoGroup,
    NavigationGroupOpen,
    HistoryPosition { supplied: i64, total: usize },
    DepthExhausted,
    Veto(String),
}
impl From<HistoryError> for SessionError {
    fn from(error: HistoryError) -> Self {
        Self::History(error)
    }
}
impl fmt::Display for SessionError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::History(error) => error.fmt(output),
            Self::NavigationGroupOpen => write!(
                output,
                "history jump with an open group: expected all transactions closed"
            ),
            Self::HistoryPosition { supplied, total } => write!(
                output,
                "history position {supplied}: expected a cursor in 0..={total}"
            ),
            Self::NoGroup => write!(
                output,
                "transaction group is absent: expected an open bounded group"
            ),
            Self::DepthExhausted => write!(
                output,
                "group depth is u64::MAX: expected space for one nested begin"
            ),
            Self::Veto(reason) => write!(
                output,
                "transaction veto {reason:?}: expected observer approval"
            ),
        }
    }
}
impl Error for SessionError {}

/// A command or its required property refresh failed before candidate adoption.
#[derive(Debug)]
pub enum CommandError<E> {
    Session(SessionError),
    Edit(EditError<MutationError<E>>),
}
/// Preserve the original command error separately from a failed projection refresh.
#[derive(Debug)]
pub enum MutationError<E> {
    Command(E),
    Property(PropertyError),
}
impl<E: fmt::Display> fmt::Display for MutationError<E> {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Command(error) => error.fmt(output),
            Self::Property(error) => error.fmt(output),
        }
    }
}
impl<E: fmt::Display> fmt::Display for CommandError<E> {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Session(error) => error.fmt(output),
            Self::Edit(error) => error.fmt(output),
        }
    }
}
impl<E: Error + 'static> Error for MutationError<E> {}
impl<E: Error + 'static> Error for CommandError<E> {}
