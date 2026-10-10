// SPDX-License-Identifier: GPL-2.0-only

use std::fmt::{Debug, Display, Formatter};

/// A rejected dependency edit. Rejection preserves all existing graph edges.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum DependencyError<Id> {
    Missing(Id),
    Duplicate(Id),
    Cycle { id: Id, driver: Id },
    InUse { id: Id, readers: Vec<Id> },
}

impl<Id: Debug> Display for DependencyError<Id> {
    fn fmt(&self, formatter: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Missing(id) => write!(
                formatter,
                "dependency ID {id:?}: expected a registered node"
            ),
            Self::Duplicate(id) => write!(formatter, "dependency ID {id:?}: expected an unused ID"),
            Self::Cycle { id, driver } => write!(
                formatter,
                "dependency {id:?} reading {driver:?}: expected an acyclic graph"
            ),
            Self::InUse { id, readers } => write!(
                formatter,
                "dependency ID {id:?} has readers {readers:?}: expected no readers before removal"
            ),
        }
    }
}

impl<Id: Debug> std::error::Error for DependencyError<Id> {}
