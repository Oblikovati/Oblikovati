// SPDX-License-Identifier: GPL-2.0-only
//! Transfer accessibility requests to the native GUI owner thread.
use super::frame::{FrameControl, control_for_id};
use accesskit::{Action, ActionHandler, ActionRequest, TreeId};
use std::sync::mpsc::Sender;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum FrameRequest {
    Click(FrameControl),
    Focus(FrameControl),
}

pub(crate) struct FrameActions<W> {
    pub sender: Sender<FrameRequest>,
    pub wake: W,
}

impl<W: Fn() + Send> ActionHandler for FrameActions<W> {
    fn do_action(&mut self, request: ActionRequest) {
        let Some(control) = requested_control(&request) else {
            return;
        };
        if self.sender.send(control).is_ok() {
            (self.wake)();
        }
    }
}

fn requested_control(request: &ActionRequest) -> Option<FrameRequest> {
    if request.target_tree != TreeId::ROOT {
        return None;
    }
    let control = control_for_id(request.target_node)?;
    match request.action {
        Action::Click => Some(FrameRequest::Click(control)),
        Action::Focus => Some(FrameRequest::Focus(control)),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use accesskit::NodeId;
    use std::sync::{
        Arc,
        atomic::{AtomicUsize, Ordering},
        mpsc,
    };

    #[test]
    fn known_clicks_and_focus_requests_reach_the_owner_and_wake_it() {
        let (sender, receiver) = mpsc::channel();
        let count = Arc::new(AtomicUsize::new(0));
        let observed = count.clone();
        let mut actions = FrameActions {
            sender,
            wake: move || {
                observed.fetch_add(1, Ordering::Relaxed);
            },
        };
        let mut request = ActionRequest {
            action: Action::Click,
            target_tree: TreeId::ROOT,
            target_node: NodeId(2),
            data: None,
        };
        actions.do_action(request.clone());
        assert_eq!(
            receiver.try_recv(),
            Ok(FrameRequest::Click(FrameControl::Minimize))
        );
        request.target_node = NodeId(99);
        actions.do_action(request.clone());
        request.target_node = NodeId(3);
        request.action = Action::Focus;
        actions.do_action(request);
        assert_eq!(
            receiver.try_recv(),
            Ok(FrameRequest::Focus(FrameControl::Maximize))
        );
        assert!(receiver.try_recv().is_err());
        assert_eq!(count.load(Ordering::Relaxed), 2);
    }
}
