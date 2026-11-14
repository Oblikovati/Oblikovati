// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{CertifiedNamedPrismEvaluator, PrismResolution};
use oblikovati_model::{
    history::DocumentEditHistory,
    identity::{DocumentGuid, RecipeIdAllocator},
};

fn resolution() -> PrismResolution {
    PrismResolution {
        version: 1,
        weld: 1e-6,
        sew: 1e-7,
        angular: 1e-9,
        display_chordal: 1e-3,
        display_angular: 0.1,
        maximum_tolerance: 1e-5,
    }
}
fn history() -> DocumentEditHistory {
    let snapshot = fixture::selected_prism_document();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    DocumentEditHistory::new(snapshot, ids).unwrap()
}
fn scene(history: &DocumentEditHistory) -> crate::HeadSceneUpdate {
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let mut evaluator = CertifiedNamedPrismEvaluator::new(history, document, resolution());
    let evaluation = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    crate::part_scene::render_part_scene(&evaluation, (1, 1), |_| Ok(Some([0., 1., 0., 1.])))
        .unwrap()
}
#[test]
fn certified_replay_draws_in_native_window_after_edit_and_undo() {
    let mut window = NativeHeadWindow::open().unwrap();
    let mut history = history();
    window.submit_scene(scene(&history)).unwrap();
    let initial = capture_native_scene(&mut window);
    assert!(green_pixels(&initial) > 0);
    let token = history.begin("Width").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(1, "6 cm")?;
            fixture::solve_selected_sketches(snapshot);
            Ok::<(), oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    window.submit_scene(scene(&history)).unwrap();
    let edited = capture_native_scene(&mut window);
    assert!(green_pixels(&edited) > green_pixels(&initial));
    history.undo().unwrap();
    window.submit_scene(scene(&history)).unwrap();
    assert_eq!(capture_native_scene(&mut window).rgba, initial.rgba);
    window.finish().unwrap();
}

fn green_pixels(capture: &ViewportCapture) -> usize {
    capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .filter(|pixel| pixel[0] == 0 && pixel[1] > 0 && pixel[2] == 0 && pixel[3] == 255)
        .count()
}
