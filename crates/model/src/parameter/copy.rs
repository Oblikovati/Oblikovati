// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Copy to a fresh caller-owned ID; for example `parameters.copy_to_user(source_id, new_id)`.
    /// Formulas keep their drivers, while the new parameter can be edited independently.
    pub fn copy_to_user(&mut self, source: Id, destination: Id) -> Result<(), ParameterError<Id>> {
        let original = self.require_record(source)?.clone();
        let name = self.unique_copy_name(&original.state.name);
        self.insert_copied_value(destination, &name, &original)?;
        let state = &mut self
            .records
            .get_mut(&destination)
            .expect("inserted copy")
            .state;
        state.annotations = copied_annotations(&original.state.annotations);
        state.tolerance = original.state.tolerance;
        state.choices = original.state.choices;
        // Preserve Go's SetExpressionList reset: copied choices become authored order.
        state.choices.custom_order = state
            .choices
            .expressions
            .as_ref()
            .is_some_and(|items| !items.is_empty());
        Ok(())
    }

    fn insert_copied_value(
        &mut self,
        id: Id,
        name: &str,
        original: &ParameterRecord<Id>,
    ) -> Result<(), ParameterError<Id>> {
        if let Some(formula) = &original.formula {
            // #3580: measured constants have diagnostic source, not parser input.
            return self.insert_compiled(
                id,
                name,
                USER_PARAM,
                formula.expression.clone(),
                original.state.expression_context.clone(),
            );
        }
        match original.state.current() {
            Some(ParameterValue::Text(text)) => self.insert_text(id, name, text),
            Some(ParameterValue::Boolean(value)) => self.insert_boolean(id, name, *value),
            found => Err(ParameterError::Value {
                id,
                found: format!("{found:?}"),
                expected: "a healthy literal or a numeric formula to copy",
            }),
        }
    }

    fn unique_copy_name(&self, source: &str) -> String {
        self.first_available_name((0..).map(|index: usize| {
            if index == 0 {
                format!("{source}_copy")
            } else {
                format!("{source}_copy{}", index + 1)
            }
        }))
    }
}

fn copied_annotations(source: &ParameterAnnotations) -> ParameterAnnotations {
    ParameterAnnotations {
        comment: source.comment.clone(),
        is_key: source.is_key,
        exposed_as_property: source.exposed_as_property,
        precision: source.precision,
        ..ParameterAnnotations::default()
    }
}
