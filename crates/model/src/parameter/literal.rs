// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Add a text user parameter; for example `parameters.insert_text(id, "material", "steel")`.
    pub fn insert_text(
        &mut self,
        id: Id,
        name: &str,
        text: &str,
    ) -> Result<(), ParameterError<Id>> {
        self.insert_literal(
            id,
            name,
            ParameterValue::Text(text.into()),
            quote_text(text),
        )
    }

    /// Add a Boolean user parameter; for example `parameters.insert_boolean(id, "vented", true)`.
    pub fn insert_boolean(
        &mut self,
        id: Id,
        name: &str,
        value: bool,
    ) -> Result<(), ParameterError<Id>> {
        self.insert_literal(id, name, ParameterValue::Boolean(value), value.to_string())
    }

    /// Edit text without parsing it as a formula; for example `parameters.set_text(id, "aluminium")`.
    pub fn set_text(&mut self, id: Id, text: &str) -> Result<(), ParameterError<Id>> {
        self.set_literal(id, ParameterValue::Text(text.into()), quote_text(text))
    }

    /// Edit a Boolean literal; for example `parameters.set_boolean(id, false)`.
    pub fn set_boolean(&mut self, id: Id, value: bool) -> Result<(), ParameterError<Id>> {
        self.set_literal(id, ParameterValue::Boolean(value), value.to_string())
    }

    /// Read healthy text; for example `parameters.text_value(material_id)`.
    pub fn text_value(&self, id: Id) -> Result<&str, ParameterError<Id>> {
        match self.require_record(id)?.state.current() {
            Some(ParameterValue::Text(text)) => Ok(text),
            found => Err(value_kind_error(id, found, "a healthy text parameter")),
        }
    }

    /// Read a healthy Boolean; for example `parameters.boolean_value(vented_id)`.
    pub fn boolean_value(&self, id: Id) -> Result<bool, ParameterError<Id>> {
        match self.require_record(id)?.state.current() {
            Some(ParameterValue::Boolean(value)) => Ok(*value),
            found => Err(value_kind_error(id, found, "a healthy Boolean parameter")),
        }
    }

    pub(super) fn require_formula(&self, id: Id) -> Result<(), ParameterError<Id>> {
        let record = self.require_record(id)?;
        if record.formula.is_none() {
            return Err(value_kind_error(
                id,
                record.state.current(),
                "a numeric parameter",
            ));
        }
        Ok(())
    }

    fn insert_literal(
        &mut self,
        id: Id,
        name: &str,
        value: ParameterValue,
        source: String,
    ) -> Result<(), ParameterError<Id>> {
        let mut record = ParameterRecord::new(name, USER_PARAM, source);
        record.state.last_good_value = Some(value);
        self.insert_record(id, record)?;
        self.changed.insert(id);
        self.resolve_forward_references(name, id);
        Ok(())
    }

    fn set_literal(
        &mut self,
        id: Id,
        value: ParameterValue,
        source: String,
    ) -> Result<(), ParameterError<Id>> {
        self.require_editable(id)?;
        let previous = self.require_record(id)?.state.current();
        if !previous
            .is_some_and(|old| std::mem::discriminant(old) == std::mem::discriminant(&value))
        {
            return Err(value_kind_error(id, previous, literal_kind(&value)));
        }
        if previous == Some(&value) {
            return Ok(());
        }
        let record = self.records.get_mut(&id).expect("validated literal");
        record.state.source = source;
        record.state.last_good_value = Some(value);
        self.recompute(id)
    }
}

fn literal_kind(value: &ParameterValue) -> &'static str {
    match value {
        ParameterValue::Text(_) => "a healthy text parameter",
        ParameterValue::Boolean(_) => "a healthy Boolean parameter",
        ParameterValue::Numeric(_) => "a healthy numeric parameter",
    }
}

fn value_kind_error<Id>(
    id: Id,
    found: Option<&ParameterValue>,
    expected: &'static str,
) -> ParameterError<Id> {
    ParameterError::Value {
        id,
        found: format!("{found:?}"),
        expected,
    }
}

fn quote_text(text: &str) -> String {
    // #3575: JSON literals preserve Unicode and controls without interpreting formula tokens.
    serde_json::to_string(text).expect("serializing a string cannot fail")
}
