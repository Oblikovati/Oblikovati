// SPDX-License-Identifier: GPL-2.0-only
//! Schedule ViewCube motion on the native owner without blocking input.
use super::NativeHeadWindow;
use oblikovati_viewport::{ViewportCamera, ViewportCameraTransition, ViewportNavigation};
use std::time::{Duration, Instant};

// Match viewCubeSnapSecs and the frame cadence in the Go desktop.
const SWING_TIME: Duration = Duration::from_millis(350);
const SWING_TICK: Duration = Duration::from_nanos(1_000_000_000 / 60);

pub(super) struct OrientationTransition {
    cameras: ViewportCameraTransition,
    elapsed: Duration,
    last_sample: Instant,
    next_frame: Instant,
    awaiting_presentation: bool,
    finish: Option<ViewportNavigation>,
}

impl NativeHeadWindow {
    pub(super) fn schedule_orientation(
        &mut self,
        target: ViewportCamera,
        finish: Option<ViewportNavigation>,
    ) -> Result<bool, String> {
        let current = self.view.camera_state()?;
        if current == target {
            self.orientation_transition = None;
            return finish.map_or(Ok(false), |command| self.navigate_view_in_document(command));
        }
        let cameras = ViewportCameraTransition::new(current, target)?;
        let started = Instant::now();
        self.orientation_transition = Some(OrientationTransition {
            cameras,
            elapsed: Duration::ZERO,
            last_sample: started,
            next_frame: started + SWING_TICK,
            awaiting_presentation: false,
            finish,
        });
        Ok(true)
    }

    pub(super) fn advance_orientation(&mut self) -> Result<(bool, Option<Duration>), String> {
        let Some(mut motion) = self.orientation_transition.take() else {
            return Ok((false, None));
        };
        if motion.awaiting_presentation {
            self.orientation_transition = Some(motion);
            return Ok((false, None));
        }
        let now = Instant::now();
        if now < motion.next_frame {
            let wake_after = motion.next_frame - now;
            self.orientation_transition = Some(motion);
            return Ok((false, Some(wake_after)));
        }
        // Skip extra work at 60 Hz, but never jump past all intermediate frames if a
        // software renderer or window resize stalls the desktop thread.
        let step = (now - motion.last_sample).min(SWING_TIME / 4);
        motion.elapsed = (motion.elapsed + step).min(SWING_TIME);
        motion.last_sample = now;
        let fraction = motion.elapsed.as_secs_f64() / SWING_TIME.as_secs_f64();
        let mut camera = motion.cameras.at(fraction)?;
        camera.perspective =
            crate::view_orientation::projects_in_perspective(&camera, self.view_projection);
        self.apply_orientation_camera(camera)?;
        if fraction == 1. {
            if let Some(command) = motion.finish {
                self.navigate_view_in_document(command)?;
            }
            return Ok((true, None));
        }
        motion.next_frame = now + SWING_TICK.min(SWING_TIME - motion.elapsed);
        motion.awaiting_presentation = true;
        let wake_after = motion.next_frame - now;
        self.orientation_transition = Some(motion);
        Ok((true, Some(wake_after)))
    }

    pub(super) fn note_orientation_presented(&mut self) {
        if let Some(motion) = &mut self.orientation_transition {
            motion.awaiting_presentation = false;
        }
    }

    pub(super) fn apply_orientation_camera(
        &mut self,
        camera: ViewportCamera,
    ) -> Result<bool, String> {
        self.view.set_camera(camera)?;
        self.view.navigate(ViewportNavigation::FitDepth)?;
        Ok(true)
    }
}
