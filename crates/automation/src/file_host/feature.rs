// SPDX-License-Identifier: GPL-2.0-only
//! Route authored features to their certified owners (#4093).
use super::*;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn add_feature(&mut self, request: &[u8]) -> Result<Vec<u8>, Error> {
        let args: AddFeatureArgs = decode(METHOD_FEATURES_ADD, request)?;
        if [
            featureargs::KIND_MOVE_BODY,
            featureargs::KIND_HOLE,
            featureargs::KIND_SHELL,
            featureargs::KIND_COIL,
            featureargs::KIND_REVOLVE,
            featureargs::KIND_SWEEP,
            featureargs::KIND_PATTERN_RECTANGULAR,
            featureargs::KIND_PATTERN_CIRCULAR,
            featureargs::KIND_MIRROR,
        ]
        .contains(&args.kind.as_str())
        {
            return self.add_part_feature(args);
        }
        if [featureargs::KIND_FILLET, featureargs::KIND_CHAMFER].contains(&args.kind.as_str()) {
            return self.add_blend(args);
        }
        let replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>> =
            match self.part_update.as_deref_mut() {
                Some(owner) => Some(owner),
                None => self
                    .part_topology
                    .as_deref_mut()
                    .map(|owner| owner as &mut dyn crate::PartUpdateAccess<O, W, F>),
            };
        crate::extrusion::apply(self.files, self.extrusion, replay, request)
    }
    fn add_part_feature(&mut self, args: AddFeatureArgs) -> Result<Vec<u8>, Error> {
        let replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>> =
            match self.part_update.as_deref_mut() {
                Some(owner) => Some(owner),
                None => self
                    .part_topology
                    .as_deref_mut()
                    .map(|owner| owner as &mut dyn crate::PartUpdateAccess<O, W, F>),
            };
        match args.kind.as_str() {
            featureargs::KIND_MOVE_BODY => crate::move_body::apply(self.files, replay, args),
            featureargs::KIND_COIL => crate::coil::apply(self.files, replay, args),
            featureargs::KIND_REVOLVE => crate::revolve::apply(self.files, replay, args),
            featureargs::KIND_SWEEP => crate::sweep::apply(self.files, replay, args),
            featureargs::KIND_SHELL => crate::shell::apply(self.files, replay, args),
            featureargs::KIND_PATTERN_RECTANGULAR
            | featureargs::KIND_PATTERN_CIRCULAR
            | featureargs::KIND_MIRROR => crate::extrusion_pattern::apply(self.files, replay, args),
            _ => crate::hole::apply(self.files, replay, args),
        }
    }
}
