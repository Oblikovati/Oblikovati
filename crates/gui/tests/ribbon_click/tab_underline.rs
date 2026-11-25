// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn clicked_ribbon_tab_moves_its_underline_with_its_command_body() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [1920, 900];
    enter_part(&mut request);
    request.commands.color_scheme =
        Some(GuiColorSchemeState::from_registry(&Default::default()).unwrap());
    let model = ribbon_caption(&mut engine, &mut request, "3D Model", false);
    let view = ribbon_caption(&mut engine, &mut request, "View", false);
    let initial = engine.frame(&request).unwrap();
    assert!(underline_at(&initial.draws, model));
    request.events = vec![GuiInputEvent::PointerMoved(view)];
    engine.frame(&request).unwrap();

    for event in [Some(true), Some(false), None, None] {
        request.events = event
            .map(|pressed| GuiInputEvent::PointerButton {
                button: GuiPointerButton::Left,
                pressed,
            })
            .into_iter()
            .collect();
        let frame = engine.frame(&request).unwrap();
        let body = drawn_text::rendered_frame();
        if body
            .runs
            .iter()
            .any(|run| run.text.replace(' ', "").contains("Colourscheme"))
        {
            assert!(underline_at(&frame.draws, view));
            assert!(!underline_at(&frame.draws, model));
            return;
        }
    }
    panic!("View commands did not appear after clicking the tab");
}

fn underline_at(frame: &GuiFrame, tab: [f32; 2]) -> bool {
    let color = [129, 114, 250, 255];
    let [x, y] = tab;
    frame.batches.iter().any(|batch| {
        batch.vertices.windows(4).any(|quad| {
            if quad.iter().any(|vertex| vertex.srgb != color) {
                return false;
            }
            let bound = |axis, order: fn(&f32, &f32) -> std::cmp::Ordering| {
                quad.iter()
                    .map(|vertex| vertex.position[axis])
                    .min_by(order)
                    .unwrap()
            };
            let left = bound(0, f32::total_cmp);
            let right = bound(0, |a, b| b.total_cmp(a));
            let top = bound(1, f32::total_cmp);
            let bottom = bound(1, |a, b| b.total_cmp(a));
            (left..right).contains(&x) && (y..y + 10.).contains(&top) && bottom - top == 2.
        })
    })
}
