// SPDX-License-Identifier: GPL-2.0-only

use super::BoundExpression;

impl<Id: Copy + Ord> BoundExpression<Id> {
    pub(crate) fn renamed_source(&self, id: Id, name: &str) -> Option<String> {
        let mut rewritten = String::new();
        let mut cursor = 0;
        for span in &self.expression.spans {
            if self.bindings[span.reference] != Some(id) {
                continue;
            }
            rewritten.push_str(&self.expression.source[cursor..span.start]);
            rewritten.push_str(name);
            cursor = span.end;
        }
        if cursor == 0 {
            return None;
        }
        rewritten.push_str(&self.expression.source[cursor..]);
        Some(rewritten)
    }
}
