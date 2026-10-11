// SPDX-License-Identifier: GPL-2.0-only

use super::TransactionToken;
use std::{error::Error, fmt};

/// A refused lifecycle operation leaves history and recipe state unchanged.
#[derive(Clone, Debug, PartialEq)]
pub enum HistoryError {
    DatumDeletion(super::DocumentDatumDeleteError),
    Identity(crate::identity::RecipeIdError),
    ScalarOwnership(crate::feature::FeatureScalarOwnershipError),
    DatumMetadata(crate::feature::work_geometry::WorkGeometryError),
    UnreservedIdentity(u64),
    UnreservedFeatureId(u64),
    UnreservedSketchId(u64),
    UnreservedSketchEntityId(u64),
    UnreservedSketchConstraintId {
        sketch: u64,
        constraint: u64,
    },
    UnreservedDerivedIdentity(i64),
    DerivedIdentityExhausted,
    ForeignTransaction,
    Inactive {
        supplied: TransactionToken,
        active: Option<TransactionToken>,
    },
    Open(TransactionToken),
    Empty(&'static str),
    Exhausted(&'static str),
}

impl fmt::Display for HistoryError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::DatumDeletion(error) => error.fmt(output),
            Self::Identity(error) => error.fmt(output),
            Self::ScalarOwnership(error) => error.fmt(output),
            Self::DatumMetadata(error) => error.fmt(output),
            Self::UnreservedFeatureId(id) => write!(
                output,
                "feature ID {id}: expected an existing identity or one reserved by this edit"
            ),
            Self::UnreservedSketchId(id) => write!(
                output,
                "sketch ID {id}: expected an existing identity or one reserved by this edit"
            ),
            Self::UnreservedSketchEntityId(id) => write!(
                output,
                "sketch entity ID {id}: expected an existing identity or one reserved by this edit"
            ),
            Self::UnreservedSketchConstraintId { sketch, constraint } => write!(
                output,
                "sketch {sketch} constraint ID {constraint}: expected an existing local identity or one reserved by this edit"
            ),
            Self::UnreservedDerivedIdentity(id) => write!(
                output,
                "derived table ID {id}: expected an existing identity or one reserved by this edit"
            ),
            Self::DerivedIdentityExhausted => write!(
                output,
                "derived table high-water mark {}: expected a value below i64::MAX",
                i64::MAX
            ),
            Self::UnreservedIdentity(id) => write!(
                output,
                "recipe ID {id}: expected an existing identity or one reserved by this edit"
            ),
            Self::ForeignTransaction => write!(
                output,
                "foreign transaction token: expected a token issued by this history owner"
            ),
            Self::Inactive { supplied, active } => write!(
                output,
                "transaction {supplied:?}: expected active top frame {active:?}"
            ),
            Self::Open(token) => write!(
                output,
                "transaction {token:?} is open: expected all transactions closed before history navigation"
            ),
            Self::Empty(action) => write!(
                output,
                "{action} stack is empty: expected an available history step"
            ),
            Self::Exhausted(namespace) => write!(
                output,
                "{namespace} reached u64::MAX: expected an unused monotonic value"
            ),
        }
    }
}
impl Error for HistoryError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::DatumDeletion(error) => Some(error),
            Self::Identity(error) => Some(error),
            Self::ScalarOwnership(error) => Some(error),
            Self::DatumMetadata(error) => Some(error),
            _ => None,
        }
    }
}
impl From<crate::identity::RecipeIdError> for HistoryError {
    fn from(error: crate::identity::RecipeIdError) -> Self {
        Self::Identity(error)
    }
}

/// Distinguish ownership errors from a rejected command's own typed error.
#[derive(Debug)]
pub enum EditError<E> {
    History(HistoryError),
    Mutation(E),
}
impl<E: fmt::Display> fmt::Display for EditError<E> {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::History(error) => error.fmt(output),
            Self::Mutation(error) => error.fmt(output),
        }
    }
}
impl<E: Error + 'static> Error for EditError<E> {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::History(error) => Some(error),
            Self::Mutation(error) => Some(error),
        }
    }
}
