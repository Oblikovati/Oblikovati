// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::wire::ExpressionListInfo;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Store ordered choices without changing the value; for example `parameters.set_expression_list(id, vec!["10 mm".into()], false)`.
    pub fn set_expression_list(
        &mut self,
        id: Id,
        expressions: Vec<String>,
        allow_custom: bool,
    ) -> Result<(), ParameterError<Id>> {
        self.require_editable(id)?;
        let populated = !expressions.is_empty();
        self.records
            .get_mut(&id)
            .expect("validated parameter")
            .state
            .choices = ExpressionListInfo {
            expressions: populated.then_some(expressions),
            allow_custom_values: allow_custom && populated,
            custom_order: populated,
        };
        Ok(())
    }

    /// Return to single-value presentation; for example `parameters.clear_expression_list(id)`.
    pub fn clear_expression_list(&mut self, id: Id) -> Result<(), ParameterError<Id>> {
        self.set_expression_list(id, Vec::new(), false)
    }

    /// Enable authored order or sort current choices; for example `parameters.set_custom_order(id, false)`.
    pub fn set_custom_order(&mut self, id: Id, custom: bool) -> Result<(), ParameterError<Id>> {
        self.require_editable(id)?;
        let choices = &mut self
            .records
            .get_mut(&id)
            .expect("validated parameter")
            .state
            .choices;
        let Some(expressions) = &mut choices.expressions else {
            return Ok(());
        };
        choices.custom_order = custom;
        if !custom {
            expressions.sort();
        }
        Ok(())
    }

    /// Select a listed or permitted custom value; for example `parameters.select_value(id, "20 mm")`.
    /// Numeric choices are already-qualified expression source; text choices are literal text.
    pub fn select_value(&mut self, id: Id, source: &str) -> Result<(), ParameterError<Id>> {
        self.require_editable(id)?;
        let record = self.require_record(id)?;
        require_choice(id, source, &record.state.choices)?;
        if record.formula.is_some() {
            return self.set_expression(id, source);
        }
        match record.state.current() {
            Some(ParameterValue::Text(_)) => self.set_text(id, source),
            Some(ParameterValue::Boolean(_)) => {
                self.set_boolean(id, parse_boolean_choice(id, source)?)
            }
            _ => Err(choice_error(
                id,
                source,
                "a healthy text or Boolean parameter",
            )),
        }
    }
}

fn require_choice<Id>(
    id: Id,
    source: &str,
    choices: &ExpressionListInfo,
) -> Result<(), ParameterError<Id>> {
    let expressions = choices.expressions.as_deref().unwrap_or_default();
    if !expressions.is_empty()
        && !choices.allow_custom_values
        && !expressions.iter().any(|choice| choice == source)
    {
        return Err(choice_error(
            id,
            source,
            "one of the parameter's allowed expressions",
        ));
    }
    Ok(())
}

fn parse_boolean_choice<Id>(id: Id, source: &str) -> Result<bool, ParameterError<Id>> {
    source
        .parse()
        .map_err(|_| choice_error(id, source, "the Boolean literal true or false"))
}

fn choice_error<Id>(id: Id, source: &str, expected: &'static str) -> ParameterError<Id> {
    ParameterError::Value {
        id,
        found: source.into(),
        expected,
    }
}
