// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Rename by stable identity; for example `parameters.rename(width_id, "span")`.
    /// Unrelated units, functions, whitespace, and references retain their authored text.
    pub fn rename(&mut self, id: Id, name: &str) -> Result<(), ParameterError<Id>> {
        let old = self.require_record(id)?.state.name.clone();
        self.require_available_name(name, Some(id))?;
        if old == name {
            return Ok(());
        }
        let replacements = self.rename_expressions(id, name)?;
        self.names.remove(&old);
        self.names.insert(name.into(), id);
        self.update_display_name(id, name);
        self.install_renamed_expressions(replacements);
        self.refresh_renamed_bindings(name)
    }

    pub(super) fn require_available_name(
        &self,
        name: &str,
        owner: Option<Id>,
    ) -> Result<(), ParameterError<Id>> {
        let occupied = self.find(name).is_some_and(|id| Some(id) != owner);
        if occupied || !crate::expression::reference_name(name) {
            return Err(ParameterError::Name {
                name: name.into(),
                expected: "a unique expression identifier without surrounding whitespace",
            });
        }
        Ok(())
    }

    fn rename_expressions(
        &self,
        id: Id,
        name: &str,
    ) -> Result<Vec<(Id, Expression)>, ParameterError<Id>> {
        self.order
            .iter()
            .filter_map(|reader| {
                self.records[reader]
                    .formula
                    .as_ref()?
                    .binding
                    .renamed_source(id, name)
                    .map(|source| (*reader, source))
            })
            .map(|(reader, source)| {
                super::expression_context::compile_context_source(
                    &source,
                    &self.units,
                    self.records[&reader].state.expression_context.as_ref(),
                )
                .map(|expression| (reader, expression))
                .map_err(Into::into)
            })
            .collect()
    }

    fn update_display_name(&mut self, id: Id, name: &str) {
        let state = &mut self
            .records
            .get_mut(&id)
            .expect("validated parameter")
            .state;
        state.name = name.into();
        state.renamed |= state.kind == MODEL_PARAM;
    }

    fn install_renamed_expressions(&mut self, replacements: Vec<(Id, Expression)>) {
        for (reader, expression) in replacements {
            let record = self.records.get_mut(&reader).expect("registered reader");
            record.state.source = expression.source().into();
            record.formula.as_mut().expect("numeric reader").expression = expression;
        }
    }

    fn refresh_renamed_bindings(&mut self, name: &str) -> Result<(), ParameterError<Id>> {
        let candidates: Vec<Id> = self
            .order
            .iter()
            .copied()
            .filter(|id| {
                self.records[id]
                    .references()
                    .iter()
                    .any(|reference| reference == name)
            })
            .collect();
        let changed: Vec<Id> = candidates
            .into_iter()
            .filter(|id| self.refresh_name_binding(*id))
            .collect();
        // Install all new bindings before evaluating any shared dependent tail.
        for id in changed {
            self.recompute(id)?;
        }
        Ok(())
    }

    fn refresh_name_binding(&mut self, id: Id) -> bool {
        let formula = self.records[&id].formula.as_ref().expect("numeric reader");
        let next = formula.expression.bind(|name| self.find(name));
        let previous = &formula.binding;
        let changed =
            previous.bound_ids() != next.bound_ids() || previous.unresolved() != next.unresolved();
        if changed {
            let _ = self.bind_expression(id);
            return true;
        }
        self.records
            .get_mut(&id)
            .expect("registered reader")
            .formula
            .as_mut()
            .expect("numeric reader")
            .binding = next;
        false
    }
}
