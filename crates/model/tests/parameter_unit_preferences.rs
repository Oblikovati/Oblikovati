// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::{
    parameter::ParameterCollection, quantity::Dimension, units::UnitsOfMeasure,
};

#[test]
fn preference_replacement_preserves_scale_and_refuses_rescaling() {
    let mut parameters = ParameterCollection::<u64>::new(UnitsOfMeasure::default());
    let mut preferences = parameters.units().clone();
    preferences.set_preferred(Dimension::Length, "in").unwrap();
    parameters
        .set_unit_preferences(preferences.clone())
        .unwrap();
    assert_eq!(parameters.units(), &preferences);
    let changed_scale = preferences.clone().with_working_scale(0.1).unwrap();
    assert!(
        parameters
            .set_unit_preferences(changed_scale)
            .unwrap_err()
            .to_string()
            .contains("unchanged document working scale")
    );
    assert_eq!(parameters.units(), &preferences);
}
