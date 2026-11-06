// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_martensite::probe_box_release;

#[test]
fn released_toolkits_construct_validate_and_integrate_in_working_units() {
    for extents in [[2.0, 3.0, 4.0], [20.0, 30.0, 40.0], [0.2, 0.3, 0.4]] {
        let properties = probe_box_release(extents).unwrap();
        let expected = extents.into_iter().product::<f64>();
        assert!((properties.volume - expected).abs() <= expected * 1e-12);
        for (coordinate, extent) in properties.centroid.into_iter().zip(extents) {
            assert!((coordinate - extent / 2.0).abs() <= extent * 1e-12);
        }
    }
}

#[test]
fn separate_evaluations_do_not_change_prior_results() {
    let first = probe_box_release([2.0, 3.0, 4.0]).unwrap();
    let second = probe_box_release([7.0, 8.0, 9.0]).unwrap();
    assert_ne!(first, second);
    assert_eq!(first, probe_box_release([2.0, 3.0, 4.0]).unwrap());
}

#[test]
fn malformed_extents_are_refused_before_kernel_construction() {
    for invalid in [0.0, -1.0, f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        for index in 0..3 {
            let mut extents = [1.0; 3];
            extents[index] = invalid;
            let message = probe_box_release(extents).unwrap_err();
            assert!(message.contains(&format!("{extents:?}")));
            assert!(message.contains("finite positive"));
        }
    }
}

#[test]
fn kernel_declines_below_resolution_without_a_panic() {
    let extents = [f64::MIN_POSITIVE; 3];
    let message = probe_box_release(extents).unwrap_err();
    assert!(message.contains(&format!("{extents:?}")));
    assert!(message.contains("solid"));
}
