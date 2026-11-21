// SPDX-License-Identifier: GPL-2.0-only
//! A model change within an input batch precedes new hover display (#4051).
use super::*;
use oblikovati_viewport::{ViewportButton, ViewportSketchFeedback};
use std::{cell::RefCell, rc::Rc};

type Trace = Rc<RefCell<Vec<&'static str>>>;

#[derive(Default)]
struct SceneOrderOwner {
    trace: Trace,
    pending: bool,
}
impl crate::HeadSketchWork for SceneOrderOwner {
    fn take_sketch_scene(&mut self) -> Result<Option<crate::HeadSketchSceneUpdate>, String> {
        if !std::mem::take(&mut self.pending) {
            return Ok(None);
        }
        Ok(Some(crate::HeadSketchSceneUpdate {
            source: Some((1, 2)),
            sketches: Vec::new(),
            plane_declines: Vec::new(),
            block_declines: Vec::new(),
        }))
    }
    fn sketch_tool_feedback(
        &mut self,
        _: [i32; 2],
        _: &dyn HeadWindow,
    ) -> Result<Option<ViewportSketchFeedback>, String> {
        Ok(Some(ViewportSketchFeedback::Point {
            sketch: oblikovati_model::document_sketch::DocumentSketchId(1),
            position: [0., 0.],
        }))
    }
}
impl HeadWork for SceneOrderOwner {
    fn process_inputs(&mut self, events: &[ViewportEvent]) -> Result<bool, String> {
        if events.iter().any(|event| {
            matches!(
                event,
                ViewportEvent::Input(ViewportInput::Button { pressed: true, .. })
            )
        }) {
            self.pending = true;
            self.trace.borrow_mut().push("edit");
        }
        Ok(false)
    }
    fn poll(&mut self, _: usize) -> Result<HeadWorkStatus, String> {
        Ok(Default::default())
    }
    fn close_allowed(&mut self) -> Result<bool, String> {
        Ok(true)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}
struct SceneOrderWindow {
    trace: Trace,
    fail: bool,
}
impl HeadWindow for SceneOrderWindow {
    fn submit_sketch_scene(&mut self, _: crate::HeadSketchSceneUpdate) -> Result<(), String> {
        self.trace.borrow_mut().push("scene");
        if self.fail {
            return Err("scene publication failed".into());
        }
        Ok(())
    }
    fn set_tool_feedback(
        &mut self,
        value: Option<&ViewportSketchFeedback>,
    ) -> Result<bool, String> {
        if value.is_some() {
            self.trace.borrow_mut().push("feedback");
        }
        Ok(value.is_some())
    }
    fn events(&mut self, _: usize) -> Result<Vec<ViewportEvent>, String> {
        Ok(Vec::new())
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        Ok(HeadPresentation::Completed)
    }
    fn wait(&mut self, _: Option<Duration>) -> Result<bool, String> {
        Ok(false)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}
fn events() -> Vec<ViewportEvent> {
    vec![
        ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Left,
            pressed: true,
            position: [10, 10],
            modifiers: Default::default(),
        }),
        ViewportEvent::Input(ViewportInput::Motion {
            position: [20, 20],
            modifiers: Default::default(),
        }),
    ]
}
#[test]
fn edit_during_batch_publishes_scene_before_hover_feedback() {
    let trace = Trace::default();
    let mut window = SceneOrderWindow {
        trace: trace.clone(),
        fail: false,
    };
    let mut owner = SceneOrderOwner {
        trace: trace.clone(),
        pending: false,
    };
    super::super::super::process_frame_inputs(&mut window, &mut owner, &events()).unwrap();
    assert_eq!(&*trace.borrow(), &["edit", "scene", "feedback"]);
}
#[test]
fn scene_publication_failure_stops_before_hover_feedback() {
    let trace = Trace::default();
    let mut window = SceneOrderWindow {
        trace: trace.clone(),
        fail: true,
    };
    let mut owner = SceneOrderOwner {
        trace: trace.clone(),
        pending: false,
    };
    let result = super::super::super::process_frame_inputs(&mut window, &mut owner, &events());
    assert_eq!(result.unwrap_err(), "scene publication failed");
    assert_eq!(&*trace.borrow(), &["edit", "scene"]);
}
