// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[derive(Clone, Debug)]
pub(super) struct ParameterFormula<Id> {
    pub expression: Expression,
    pub binding: BoundExpression<Id>,
}

#[derive(Clone, Debug)]
pub(super) struct ParameterRecord<Id> {
    pub state: ParameterState<Id>,
    pub formula: Option<ParameterFormula<Id>>,
}

impl<Id> ParameterRecord<Id> {
    pub fn new(name: &str, kind: ParameterKind, source: String) -> Self {
        let state = ParameterState {
            name: name.into(),
            annotations: ParameterAnnotations::default(),
            kind,
            source,
            expression_context: None,
            last_good_value: None,
            renamed: false,
            built_in: false,
            disabled_actions: oblikovati_api::types::ACTION_NONE,
            custom_property: super::CustomPropertyFormat::default(),
            tolerance: oblikovati_api::types::Tolerance::default(),
            choices: oblikovati_api::wire::ExpressionListInfo::default(),
            model_value_type: oblikovati_api::types::MODEL_VALUE_NOMINAL,
            health: ParameterHealth::Healthy,
        };
        Self {
            state,
            formula: None,
        }
    }

    pub fn references(&self) -> &[String] {
        self.formula
            .as_ref()
            .map_or(&[], |formula| formula.expression.references())
    }
}
