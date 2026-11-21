// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{MODEL_VALUE_UPPER, TOLERANCE_LIMITS_FITS_STACKED, USER_PARAM};
use oblikovati_model::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

fn fit_parameter(scale: f64, dimension: Dimension) -> ParameterCollection<u64> {
    let units = UnitsOfMeasure::default().with_working_scale(scale).unwrap();
    let mut parameters = ParameterCollection::new(units);
    parameters
        .insert_value(
            1,
            "diameter",
            USER_PARAM,
            Quantity::new(5.0 / scale, dimension),
        )
        .unwrap();
    parameters.drain_changed();
    parameters
}

#[test]
fn fit_annotations_use_working_scale_and_hole_precedence() {
    for scale in [0.001, 0.1, 1.0, 100.0] {
        let mut parameters = fit_parameter(scale, Dimension::Length);
        parameters.set_fits(1, "H7", "g6").unwrap();
        let band = &parameters.get(1).unwrap().tolerance;
        assert_eq!(band.type_, TOLERANCE_LIMITS_FITS_STACKED);
        assert_eq!(
            (&*band.hole_tolerance, &*band.shaft_tolerance),
            ("H7", "g6")
        );
        assert_eq!(band.upper, 0.025 / 10.0 / scale);
        assert_eq!(band.lower, 0.0);
        assert_eq!(parameters.drain_changed(), vec![1]);
        parameters.set_fits(1, "H7", "g6").unwrap();
        assert!(parameters.drain_changed().is_empty());
        parameters
            .set_model_value_type(1, MODEL_VALUE_UPPER)
            .unwrap();
        assert_eq!(
            parameters.model_value(1).unwrap().value,
            5.0 / scale + 0.025 / 10.0 / scale
        );
    }
}

#[test]
fn shaft_only_and_symmetric_fits_keep_their_exact_bands() {
    let mut parameters = fit_parameter(1.0, Dimension::Length);
    parameters.set_fits(1, "", "g6").unwrap();
    assert_eq!(parameters.get(1).unwrap().tolerance.upper, -0.009 / 10.0);
    assert_eq!(parameters.get(1).unwrap().tolerance.lower, -0.025 / 10.0);
    parameters.set_fits(1, "", "js7").unwrap();
    assert_eq!(parameters.get(1).unwrap().tolerance.upper, 0.0125 / 10.0);
    assert_eq!(parameters.get(1).unwrap().tolerance.lower, -0.0125 / 10.0);
}

#[test]
fn rejected_fits_do_not_change_tolerance_or_dirty_state() {
    let mut parameters = fit_parameter(1.0, Dimension::Length);
    parameters.set_fits(1, "H7", "").unwrap();
    parameters.drain_changed();
    let before = parameters.get(1).unwrap().clone();
    for (hole, shaft) in [("", ""), ("H12", "g6"), ("H7", "invalid")] {
        assert!(parameters.set_fits(1, hole, shaft).is_err());
        assert_eq!(parameters.get(1).unwrap(), &before);
        assert!(parameters.drain_changed().is_empty());
    }
    assert!(parameters.set_fits(99, "H7", "").is_err());
    assert!(
        fit_parameter(1.0, Dimension::Angle)
            .set_fits(1, "H7", "")
            .is_err()
    );
}

#[test]
fn failed_health_and_out_of_range_nominals_refuse_fits() {
    let mut parameters = fit_parameter(1.0, Dimension::Length);
    parameters.set_expression(1, "missing").unwrap();
    assert!(parameters.set_fits(1, "H7", "").is_err());
    for nominal in [0.3, 50.1] {
        parameters
            .set_value(1, Quantity::new(nominal, Dimension::Length))
            .unwrap();
        assert!(parameters.set_fits(1, "H7", "").is_err());
    }
}
