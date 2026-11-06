// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Client,
    types::{DOCUMENT_ASSEMBLY, DOCUMENT_PART},
    wire::SketchArgs,
};
use oblikovati_gui::{GuiBrowserFeature, GuiBrowserSketch, GuiModelBrowser};
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn gui_browser_snapshot(&self, document: u64) -> Result<GuiModelBrowser, String> {
        let mut browser = self.gui_browser_root(document)?;
        if !browser.has_origin {
            return Ok(browser);
        }
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let query = oblikovati_automation::FileQueryHost::new(&self.files)
            .with_sketch_edit(self.sketch_inspection_edit());
        populate_browser(&mut browser, snapshot, &Client::new(query));
        browser.selected_sketches = self.selected_browser_sketches(document);
        Ok(browser)
    }
    fn gui_browser_root(&self, document: u64) -> Result<GuiModelBrowser, String> {
        let info = self
            .files
            .document_info(document)
            .map_err(|cause| cause.to_string())?;
        let kind = self
            .files
            .document_type(document)
            .map_err(|cause| cause.to_string())?;
        Ok(GuiModelBrowser {
            name: info.name,
            has_origin: matches!(kind, DOCUMENT_PART | DOCUMENT_ASSEMBLY),
            ..Default::default()
        })
    }
}
fn populate_browser<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    browser: &mut GuiModelBrowser,
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    client: &Client<oblikovati_automation::FileQueryHost<'_, O, W, F>>,
) {
    browser.sketches = snapshot
        .sketches
        .sketches()
        .iter()
        .enumerate()
        .map(|(index, sketch)| browser_sketch(snapshot, sketch, index, client))
        .collect();
    browser.features = snapshot
        .work_geometry
        .features()
        .iter()
        .map(|feature| GuiBrowserFeature {
            id: feature.id(),
            name: feature.name().into(),
            suppressed: feature.suppressed(),
        })
        .collect();
}
fn browser_sketch<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    sketch: &oblikovati_model::document_sketch::PlacedSketch,
    index: usize,
    client: &Client<oblikovati_automation::FileQueryHost<'_, O, W, F>>,
) -> GuiBrowserSketch {
    let inspection = i64::try_from(index)
        .map_err(|_| format!("browser sketch index {index}: expected an i64 index"))
        .and_then(|sketch_index| {
            client
                .sketch()
                .get(&SketchArgs { sketch_index })
                .map_err(|cause| cause.to_string())
        });
    GuiBrowserSketch {
        id: sketch.id(),
        visible: sketch.visible(),
        name: sketch.name().into(),
        inspection,
        consumed_by: exclusive_sketch_consumer(snapshot, sketch.id()),
    }
}
fn exclusive_sketch_consumer(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
) -> Option<oblikovati_model::feature::FeatureId> {
    let mut consumers = snapshot.work_geometry.sketch_consumers(sketch);
    let first = consumers.next()?;
    // Go app/browser.go keeps a sketch at top level when several features consume it.
    consumers.next().is_none().then(|| first.id())
}
