// SPDX-License-Identifier: GPL-2.0-only

use super::{Expression, ExpressionError, functions, node::Node};
use crate::quantity::Quantity;

enum Step {
    Visit(usize),
    Finish(usize, usize),
}

impl Expression {
    pub(super) fn evaluate_with(
        &self,
        mut reference: impl FnMut(usize) -> Result<Quantity, ExpressionError>,
    ) -> Result<Quantity, ExpressionError> {
        let mut pending = vec![Step::Visit(self.root)];
        let mut values = Vec::new();
        while let Some(step) = pending.pop() {
            match step {
                Step::Visit(index) => {
                    self.visit(index, &mut pending, &mut values, &mut reference)?
                }
                Step::Finish(index, count) => self.finish(index, count, &mut values)?,
            }
        }
        Ok(values[0])
    }

    fn visit(
        &self,
        index: usize,
        pending: &mut Vec<Step>,
        values: &mut Vec<Quantity>,
        reference: &mut impl FnMut(usize) -> Result<Quantity, ExpressionError>,
    ) -> Result<(), ExpressionError> {
        match &self.nodes[index] {
            Node::Literal(quantity) => values.push(*quantity),
            Node::Reference(index) => values.push(reference(*index)?),
            node => schedule(index, node, pending)?,
        }
        Ok(())
    }

    fn finish(
        &self,
        index: usize,
        count: usize,
        values: &mut Vec<Quantity>,
    ) -> Result<(), ExpressionError> {
        let start = values.len() - count;
        let result = self.nodes[index].compute(&values[start..])?;
        values.truncate(start);
        values.push(result);
        Ok(())
    }
}

fn schedule(index: usize, node: &Node, pending: &mut Vec<Step>) -> Result<(), ExpressionError> {
    // Unknown functions fail before evaluating their arguments, as in the Go host.
    if let Node::Call(name, _) = node {
        functions::require_known(name)?;
    }
    let children = node.children();
    pending.push(Step::Finish(index, children.len()));
    pending.extend(children.into_iter().rev().map(Step::Visit));
    Ok(())
}
