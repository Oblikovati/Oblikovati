// SPDX-License-Identifier: GPL-2.0-only
use crate::{DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId, history::DocumentEditHistory, sketch::SketchError,
};
/// Resolve a sketch for every published history state.
pub trait SketchContinuityResolver: Send + Sync {
    /// Validate the current placement; example: `resolver.resolve(history, sketch)?`.
    fn resolve(
        &self,
        history: &DocumentEditHistory,
        sketch: DocumentSketchId,
    ) -> Result<(), String>;
}
use std::sync::{
    Arc, Weak,
    atomic::{AtomicBool, Ordering},
};
/// A read-only lifetime observation, not a frozen geometry or generation handle.
#[derive(Clone, Debug)]
pub struct SketchContinuityWatch {
    sketch: DocumentSketchId,
    live: Arc<AtomicBool>,
}
impl SketchContinuityWatch {
    /// Read the watched identity; example: `watch.sketch()` before resolving the current plane.
    pub fn sketch(&self) -> DocumentSketchId {
        self.sketch
    }
    /// Test uninterrupted published presence; redo cannot revive a lost observation.
    /// Example: `watch.is_live()` before retaining an interactive edit.
    pub fn is_live(&self) -> bool {
        self.live.load(Ordering::Acquire)
    }
}
struct WatchedSketch {
    sketch: DocumentSketchId,
    live: Weak<AtomicBool>,
}
#[derive(Default)]
pub(crate) struct SketchContinuityWatches {
    watches: Vec<WatchedSketch>,
    resolver: Option<Box<dyn SketchContinuityResolver>>,
}
impl SketchContinuityWatches {
    fn observe(&mut self, sketch: DocumentSketchId) -> SketchContinuityWatch {
        self.watches.retain(|watch| watch.live.strong_count() != 0);
        let live = Arc::new(AtomicBool::new(true));
        self.watches.push(WatchedSketch {
            sketch,
            live: Arc::downgrade(&live),
        });
        SketchContinuityWatch { sketch, live }
    }
    fn resolve(
        &self,
        history: &DocumentEditHistory,
        sketch: DocumentSketchId,
    ) -> Result<(), String> {
        resolve_sketch(self.resolver.as_deref(), history, sketch)
    }
    pub(crate) fn refresh(&mut self, history: &DocumentEditHistory) {
        let resolver = self.resolver.as_deref();
        self.watches.retain(|watch| {
            let Some(live) = watch.live.upgrade() else {
                return false;
            };
            if resolve_sketch(resolver, history, watch.sketch).is_err() {
                live.store(false, Ordering::Release);
                return false;
            }
            true
        });
    }
}
fn resolve_sketch(
    resolver: Option<&dyn SketchContinuityResolver>,
    history: &DocumentEditHistory,
    sketch: DocumentSketchId,
) -> Result<(), String> {
    history
        .snapshot()
        .sketches
        .get(sketch)
        .map_err(|error| error.to_string())?;
    match resolver {
        Some(resolver) => resolver.resolve(history, sketch),
        None => history
            .snapshot()
            .resolve_sketch_plane(sketch)
            .map(|_| ())
            .map_err(|error| error.to_string()),
    }
}
impl Drop for SketchContinuityWatches {
    fn drop(&mut self) {
        for watch in &self.watches {
            if let Some(live) = watch.live.upgrade() {
                live.store(false, Ordering::Release);
            }
        }
    }
}
impl<O: TransactionObserver> DocumentSession<O> {
    /// Install a placement resolver for published sketch continuity checks.
    /// Example: `session.set_sketch_continuity_resolver(Box::new(resolver))`.
    pub fn set_sketch_continuity_resolver(&mut self, resolver: Box<dyn SketchContinuityResolver>) {
        self.sketch_continuity.resolver = Some(resolver);
        self.sketch_continuity.refresh(&self.history);
    }
    /// Observe sketch lifetime across every published history change, including batched undo/redo.
    /// Example: `let watch = session.watch_sketch_continuity(sketch)?;` for a transient editor.
    pub fn watch_sketch_continuity(
        &mut self,
        sketch: DocumentSketchId,
    ) -> Result<SketchContinuityWatch, SketchError> {
        self.sketch_continuity
            .resolve(&self.history, sketch)
            .map_err(|cause| {
                SketchError::Definition(format!(
                    "sketch continuity {sketch:?}: expected a current placed sketch: {cause}"
                ))
            })?;
        Ok(self.sketch_continuity.observe(sketch))
    }
}
