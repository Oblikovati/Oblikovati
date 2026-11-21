// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_solve::ad::{DualNumber, DualVector};

fn constants<const N: usize>(values: [f64; N]) -> DualVector<N> {
    DualVector::new(values.map(|value| DualNumber::constant(value).unwrap())).unwrap()
}
fn seeded<const N: usize>(values: [f64; N]) -> DualVector<N> {
    let variables =
        std::array::from_fn(|index| DualNumber::variable(values[index], index, N).unwrap());
    DualVector::new(variables).unwrap()
}
fn values<const N: usize>(vector: &DualVector<N>) -> [f64; N] {
    std::array::from_fn(|index| vector.components()[index].value())
}
fn close(actual: f64, expected: f64) {
    assert!(
        (actual - expected).abs() < 1e-9,
        "actual {actual:?}, expected {expected:?}"
    );
}

#[test]
fn planar_arithmetic_dot_cross_and_lengths_have_analytic_derivatives() {
    let a = seeded([1.0, 2.0]);
    let b = constants([3.0, 5.0]);
    assert_eq!(values(&a.plus(&b).unwrap()), [4.0, 7.0]);
    assert_eq!(values(&a.minus(&b).unwrap()), [-2.0, -3.0]);
    assert_eq!(values(&a.scaled(2.0).unwrap()), [2.0, 4.0]);
    let dot = a.dot(&b).unwrap();
    assert_eq!(dot.value(), 13.0);
    assert_eq!(dot.gradient(), [3.0, 5.0]);
    let cross = a.cross(&b).unwrap();
    assert_eq!(cross.value(), -1.0);
    assert_eq!(cross.gradient(), [5.0, -3.0]);
    let length = seeded([3.0, 4.0]).length().unwrap();
    assert_eq!(length.value(), 5.0);
    assert_eq!(length.gradient(), [0.6, 0.8]);
    assert_eq!(a.components()[0].gradient(), [1.0, 0.0]);
}

#[test]
fn spatial_operations_share_arithmetic_and_match_go_cross_product() {
    let a = seeded([1.0, 2.0, 3.0]);
    let b = constants([4.0, 5.0, 6.0]);
    assert_eq!(values(&a.plus(&b).unwrap()), [5.0, 7.0, 9.0]);
    assert_eq!(values(&b.minus(&a).unwrap()), [3.0, 3.0, 3.0]);
    assert_eq!(values(&b.scaled(0.5).unwrap()), [2.0, 2.5, 3.0]);
    let cross = a.cross(&b).unwrap();
    assert_eq!(values(&cross), [-3.0, 6.0, -3.0]);
    for (component, gradient) in
        cross
            .components()
            .iter()
            .zip([[0.0, 6.0, -5.0], [-6.0, 0.0, 4.0], [5.0, -4.0, 0.0]])
    {
        assert_eq!(component.gradient(), gradient);
    }
    assert_eq!(a.dot(&b).unwrap().gradient(), [4.0, 5.0, 6.0]);
    let scaled = a.product(&a.components()[0]).unwrap();
    assert_eq!(values(&scaled), [1.0, 2.0, 3.0]);
    assert_eq!(scaled.components()[1].gradient(), [2.0, 1.0, 0.0]);
}

#[test]
fn cross_product_partials_match_independent_central_differences() {
    let coordinates = [1.2, -2.1, 0.7, 3.4, 0.2, -1.5];
    let variables = DualNumber::seed(&coordinates).unwrap();
    let a = DualVector::new(std::array::from_fn::<_, 3, _>(|i| variables[i].clone())).unwrap();
    let b = DualVector::new(std::array::from_fn::<_, 3, _>(|i| variables[i + 3].clone())).unwrap();
    let result = a.cross(&b).unwrap();
    for column in 0..6 {
        let mut plus = coordinates;
        let mut minus = coordinates;
        plus[column] += 1e-5;
        minus[column] -= 1e-5;
        let high = scalar_cross(plus);
        let low = scalar_cross(minus);
        for row in 0..3 {
            close(
                result.components()[row].gradient()[column],
                (high[row] - low[row]) / 2e-5,
            );
        }
    }
}
fn scalar_cross([x, y, z, u, v, w]: [f64; 6]) -> [f64; 3] {
    [y * w - z * v, z * u - x * w, x * v - y * u]
}

#[test]
fn norms_retain_representable_values_and_gradients_across_scales() {
    for scale in [1e-200, 1.0, 1e200] {
        let vector = seeded([2.0 * scale, -3.0 * scale, 6.0 * scale]);
        let norm = vector.length().unwrap();
        close(norm.value() / scale, 7.0);
        for (&actual, expected) in norm
            .gradient()
            .iter()
            .zip([2.0 / 7.0, -3.0 / 7.0, 6.0 / 7.0])
        {
            close(actual, expected);
        }
    }
    let origin = seeded([0.0, 0.0, 0.0]).length().unwrap();
    assert_eq!(origin.value(), 0.0);
    assert_eq!(origin.gradient(), [0.0, 0.0, 0.0]);
    assert!(
        constants([3.0, 4.0])
            .length()
            .unwrap()
            .gradient()
            .is_empty()
    );
}

#[test]
fn incompatible_dimensions_and_overflow_fail_without_mutating_inputs() {
    let a = DualNumber::variable(1.0, 0, 2).unwrap();
    let b = DualNumber::variable(1.0, 0, 3).unwrap();
    assert!(
        DualVector::new([a.clone(), b.clone()])
            .unwrap_err()
            .to_string()
            .contains("dimensions [2, 3]")
    );
    let left = DualVector::new([a.clone(), DualNumber::constant(1.0).unwrap()]).unwrap();
    let right = DualVector::new([DualNumber::constant(1.0).unwrap(), b.clone()]).unwrap();
    assert!(left.plus(&right).is_err());
    assert!(left.dot(&right).is_err());
    assert!(left.cross(&right).is_err());
    assert!(left.product(&b).is_err());
    assert!(left.scaled(f64::INFINITY).is_err());
    assert_eq!(left.components()[0], a);
    assert!(constants([f64::MAX, f64::MAX]).length().is_err());
}
