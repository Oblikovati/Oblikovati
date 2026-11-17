// SPDX-License-Identifier: GPL-2.0-only
//! Transfer accessibility requests to the native GUI owner thread.
use super::frame::{FrameControl, control_for_id};
use accesskit::{Action, ActionHandler, ActionRequest, TreeId};
use std::sync::mpsc::Sender;

pub(crate) struct FrameActions<W> {
    pub sender: Sender<FrameControl>,
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

fn requested_control(request: &ActionRequest) -> Option<FrameControl> {
    if request.action != Action::Click || request.target_tree != TreeId::ROOT {
        return None;
    }
    control_for_id(request.target_node)
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
    fn only_known_clicks_reach_the_owner_and_wake_it() {
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
        assert_eq!(receiver.try_recv(), Ok(FrameControl::Minimize));
        request.target_node = NodeId(99);
        actions.do_action(request.clone());
        request.action = Action::Focus;
        actions.do_action(request);
        assert!(receiver.try_recv().is_err());
        assert_eq!(count.load(Ordering::Relaxed), 1);
    }
}
