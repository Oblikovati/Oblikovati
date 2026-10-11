// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_solve::nonlinear::{SolveOptions, SolveTermination};

#[test]
fn simultaneous_radius_solves_cover_equal_thin_and_near_equal_shapes_across_scales() {
    for scale in [1e-6, 1.0, 1e6] {
        for ratio in [1.0, 1.0 - 1e-8, 1e-6, 1e-12] {
            assert_degenerate_radius_solve(scale, ratio);
        }
    }
}
fn scaled_radius_sketch(scale: f64, ratio: f64) -> (Sketch2D, ParameterCollection<u64>) {
    let (mut sketch, mut parameters) = minor_driving();
    let mut recipe = sketch.recipe();
    let shape = recipe.ellipses.as_mut().unwrap().first_mut().unwrap();
    *shape = shape.with_radii([3.0 * scale, scale * ratio]).unwrap();
    sketch = Sketch2D::from_recipe(&recipe).unwrap();
    constrain_ellipse_frame(&mut sketch);
    parameters
        .set_expression(1, &format!("{} cm", 4.0 * scale))
        .unwrap();
    parameters
        .set_expression(2, &format!("{} cm", 4.0 * scale * ratio))
        .unwrap();
    (sketch, parameters)
}
fn assert_degenerate_radius_solve(scale: f64, ratio: f64) {
    let (sketch, parameters) = scaled_radius_sketch(scale, ratio);
    let outcome = sketch
        .solve(&parameters, radius_precision(scale, ratio), || false)
        .unwrap();
    assert_eq!(
        outcome.numerical().termination,
        SolveTermination::Converged,
        "scale={scale}, ratio={ratio}"
    );
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 0);
    assert_radius_accuracy(outcome.candidate(), scale, ratio);
    assert_solved_radius_storage(outcome.candidate());
    assert_eq!(
        sketch.ellipse(SketchEntityId(2)).unwrap().radii(),
        [3.0 * scale, scale * ratio]
    );
}
fn radius_precision(scale: f64, ratio: f64) -> SolveOptions {
    // The numerical scale includes orientation: tighten explicitly to resolve the small radius.
    SolveOptions {
        relative_tolerance: 1e-10 * scale.min(1.0) * ratio,
        ..Default::default()
    }
}
fn assert_radius_accuracy(candidate: &Sketch2D, scale: f64, ratio: f64) {
    let ellipse = candidate.ellipse(SketchEntityId(2)).unwrap();
    for (actual, expected) in ellipse
        .radii()
        .into_iter()
        .zip([4.0 * scale, 4.0 * scale * ratio])
    {
        assert!(
            (actual / expected - 1.0).abs() < 1e-8,
            "{actual}: expected {expected}"
        );
    }
    assert!(ellipse.major_axis()[1].abs() < 1e-8);
    assert_eq!(
        candidate.point(ellipse.center()).unwrap().position,
        [2.0, -3.0]
    );
    assert_eq!(ellipse.id(), SketchEntityId(2));
}

fn assert_solved_radius_storage(candidate: &Sketch2D) {
    let restored = Sketch2D::from_recipe(&candidate.recipe()).unwrap();
    assert!(candidate.same_recipe(&restored));
}
