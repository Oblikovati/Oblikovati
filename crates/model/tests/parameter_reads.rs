// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{MODEL_VALUE_UPPER, USER_PARAM};
use oblikovati_model::{
    parameter::{ParameterCollection, ParameterError},
    units::UnitsOfMeasure,
};

fn collection() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    for (id, name, source) in [
        (9, "width", "10 cm"),
        (3, "depth", "5 cm"),
        (1, "unused", "1 cm"),
    ] {
        parameters
            .insert_expression(id, name, USER_PARAM, source)
            .unwrap();
    }
    parameters.drain_changed();
    parameters
}

#[test]
fn read_scopes_capture_only_model_reads_in_insertion_order() {
    let parameters = collection();
    let (result, reads) = parameters.track(|scope| {
        let depth = scope.model_value(3)?;
        let width = scope.model_value(9)?;
        scope.model_value(3)?;
        Ok::<f64, ParameterError<u64>>(depth.value * width.value)
    });
    assert_eq!(result.unwrap(), 50.0);
    assert_eq!(reads, [9, 3]);
    assert_eq!(
        parameters.get(1).unwrap().current_value().unwrap().value,
        1.0
    );
    assert!(parameters.track(|_| 42).1.is_empty());
}

#[test]
fn nested_reads_bubble_to_parent_without_losing_the_inner_footprint() {
    let parameters = collection();
    let (inner_reads, outer_reads) = parameters.track(|scope| {
        scope.model_value(9).unwrap();
        let (result, reads) = scope.track(|inner| inner.model_value(3));
        assert_eq!(result.unwrap().value, 5.0);
        reads
    });
    assert_eq!(inner_reads, [3]);
    assert_eq!(outer_reads, [9, 3]);
}

#[test]
fn failed_reads_keep_existing_ids_for_recovery_invalidation() {
    let mut parameters = collection();
    parameters.set_expression(9, "missing").unwrap();
    parameters.drain_changed();
    let (result, reads) = parameters.track(|scope| scope.model_value(9));
    assert!(result.is_err());
    assert_eq!(reads, [9]);
    parameters.set_expression(9, "20 cm").unwrap();
    assert!(
        parameters
            .drain_changed()
            .iter()
            .any(|id| reads.contains(id))
    );
    let (result, reads) = parameters.track(|scope| scope.model_value(999));
    assert!(result.is_err());
    assert!(reads.is_empty());
}

#[test]
fn tolerance_edits_invalidate_consumers_without_dirtying_nominal_readers() {
    let mut parameters = collection();
    parameters
        .insert_expression(4, "twice", USER_PARAM, "2 * width")
        .unwrap();
    let (_, footprint) = parameters.track(|scope| scope.model_value(9));
    parameters.drain_changed();
    parameters.set_deviation(9, 0.2, -0.1).unwrap();
    parameters
        .set_model_value_type(9, MODEL_VALUE_UPPER)
        .unwrap();
    let changed = parameters.drain_changed();
    assert_eq!(changed, [9]);
    assert_eq!(footprint, [9]);
    assert_eq!(parameters.model_value(9).unwrap().value, 10.2);
    assert_eq!(parameters.model_value(4).unwrap().value, 20.0);
}
