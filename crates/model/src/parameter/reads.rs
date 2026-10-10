// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::quantity::Quantity;

/// An immutable model-value read scope with an explicit dependency footprint.
pub struct ParameterReads<'a, Id> {
    parameters: &'a ParameterCollection<Id>,
    reads: BTreeSet<Id>,
}

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Capture model reads without global state; for example `parameters.track(|scope| scope.model_value(id))`.
    /// Error results retain their read footprint so recovery can invalidate the consumer.
    pub fn track<ResultValue>(
        &self,
        work: impl FnOnce(&mut ParameterReads<'_, Id>) -> ResultValue,
    ) -> (ResultValue, Vec<Id>) {
        let mut scope = ParameterReads {
            parameters: self,
            reads: BTreeSet::new(),
        };
        let result = work(&mut scope);
        (result, self.ordered_reads(&scope.reads))
    }

    fn ordered_reads(&self, reads: &BTreeSet<Id>) -> Vec<Id> {
        self.order
            .iter()
            .copied()
            .filter(|id| reads.contains(id))
            .collect()
    }
}

impl<Id: Copy + Ord + Debug> ParameterReads<'_, Id> {
    /// Read an existing parameter and record its identity; for example `scope.model_value(depth)`.
    pub fn model_value(&mut self, id: Id) -> Result<Quantity, ParameterError<Id>> {
        if self.parameters.get(id).is_some() {
            self.reads.insert(id);
        }
        self.parameters.model_value(id)
    }

    /// Capture a nested consumer; for example `scope.track(|inner| inner.model_value(radius))`.
    /// Every nested read also belongs to the enclosing footprint.
    pub fn track<ResultValue>(
        &mut self,
        work: impl FnOnce(&mut ParameterReads<'_, Id>) -> ResultValue,
    ) -> (ResultValue, Vec<Id>) {
        let (result, reads) = self.parameters.track(work);
        self.reads.extend(reads.iter().copied());
        (result, reads)
    }
}
