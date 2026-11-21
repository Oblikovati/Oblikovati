// SPDX-License-Identifier: GPL-2.0-only
//! Repeatable GUI-only timing for dense sketch dimension labels.
use oblikovati_gui::{
    GuiAuthoringContext, GuiCommandState, GuiDimensionEdit, GuiDimensionLabel,
    GuiDimensionLabelRole, GuiFrameEngine, GuiFrameRequest, GuiInputEvent,
};

fn request(count: usize, overlap: bool) -> GuiFrameRequest {
    let labels = (0..count)
        .map(|index| GuiDimensionLabel {
            pixel: if overlap {
                [720, 450]
            } else {
                [
                    390 + (index % 20) as i32 * 50,
                    240 + (index / 20) as i32 * 50,
                ]
            },
            line: None,
            text: format!("{} mm", index + 1),
            role: GuiDimensionLabelRole::Driving,
            edit: Some(GuiDimensionEdit {
                sketch: 1,
                constraint: index as u64 + 1,
                name: format!("d{index}"),
                expression: format!("{} mm", index + 1),
                driven: false,
            }),
        })
        .collect();
    GuiFrameRequest {
        extent: [1440, 900],
        events: Vec::new(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1., 1.],
        commands: GuiCommandState {
            document: Some(1),
            active_sketch: Some(oblikovati_model::document_sketch::DocumentSketchId(1)),
            authoring: GuiAuthoringContext::Sketch,
            sketch_at_ready: true,
            dimension_labels: labels,
            ..Default::default()
        },
    }
}
fn measure(count: usize, overlap: bool) {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = request(count, overlap);
    request.events = vec![GuiInputEvent::PointerMoved([720., 450.])];
    engine.frame(&request).unwrap();
    request.events.clear();
    for _ in 0..10 {
        engine.frame(&request).unwrap();
    }
    let mut times = Vec::new();
    for _ in 0..60 {
        let start = std::time::Instant::now();
        engine.frame(&request).unwrap();
        times.push(start.elapsed());
    }
    times.sort();
    eprintln!(
        "dimension labels={count}, overlap={overlap}, p50={:?}, p95={:?}, max={:?}",
        times[30], times[57], times[59]
    );
}
#[test]
#[ignore = "manual GUI frame timing; run on an idle machine"]
fn dense_dimension_layout_frame_timing() {
    measure(200, false);
    measure(1000, true);
}
