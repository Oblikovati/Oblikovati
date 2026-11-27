// SPDX-License-Identifier: GPL-2.0-only
//! Cache the property table projection by document generation for frame snapshots.
use super::*;
use oblikovati_api::wire::ModelStateTableView;
use std::sync::Arc;

#[derive(Default)]
pub(super) struct ModelStateTableCache(Option<(u64, u64, Arc<ModelStateTableView>)>);
impl ModelStateTableCache {
    pub(super) fn get(
        &mut self,
        document: u64,
        generation: u64,
        load: impl FnOnce() -> Result<ModelStateTableView, String>,
    ) -> Result<Arc<ModelStateTableView>, String> {
        if let Some((owner, revision, view)) = &self.0
            && (*owner, *revision) == (document, generation)
        {
            return Ok(Arc::clone(view));
        }
        let view = Arc::new(load()?);
        self.0 = Some((document, generation, Arc::clone(&view)));
        Ok(view)
    }
}
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> crate::ScriptDocumentOwner for DesktopOwner<O, W, F, G, R, S>
{
    type Host<'a>
        = FileHost<'a, O, W, F>
    where
        Self: 'a;
    fn script_host(&mut self) -> Self::Host<'_> {
        self.document_host()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn cache_reuses_only_the_matching_document_and_generation() {
        let mut cache = ModelStateTableCache::default();
        let first = cache
            .get(1, 3, || Ok(ModelStateTableView::default()))
            .unwrap();
        let again = cache
            .get(1, 3, || panic!("unchanged state should reuse its view"))
            .unwrap();
        assert!(Arc::ptr_eq(&first, &again));
        let changed = cache
            .get(1, 4, || Ok(ModelStateTableView::default()))
            .unwrap();
        assert!(!Arc::ptr_eq(&first, &changed));
        let other = cache
            .get(2, 4, || Ok(ModelStateTableView::default()))
            .unwrap();
        assert!(!Arc::ptr_eq(&changed, &other));
    }
}
