// SPDX-License-Identifier: GPL-2.0-only
//! The Parameters window keeps nominal and tolerance-selected values distinct (#4125).
use super::*;
use oblikovati_api::types::{MODEL_VALUE_UPPER, USER_PARAM};
use oblikovati_model::units::UnitsOfMeasure;

#[test]
fn model_value_uses_selected_tolerance_and_document_units() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "1 cm")
        .unwrap();
    parameters.set_symmetric(1, 0.5).unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_UPPER)
        .unwrap();
    let row = projected_parameter_row(&parameters, 1, 1, None).unwrap();
    assert_eq!(row.detail.parameter_info.value, "10 mm");
    assert_eq!(row.model_value_text, "15 mm");
}
