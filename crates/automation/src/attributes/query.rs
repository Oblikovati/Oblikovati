// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{decode, encode};
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};

pub(crate) fn dispatch<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    match method {
        METHOD_ATTRIBUTES_GET => {
            let request: GetAttributeArgs = decode(method, request)?;
            encode(&get(owned(workspace, request.document)?, request)?)
        }
        METHOD_ATTRIBUTES_LIST => {
            let request: ListAttributesArgs = decode(method, request)?;
            encode(&list(owned(workspace, request.document)?, request)?)
        }
        METHOD_ATTRIBUTES_LIST_SETS => {
            let request: ListAttributeSetsArgs = decode(method, request)?;
            encode(&sets(owned(workspace, request.document)?)?)
        }
        METHOD_ATTRIBUTES_FIND => encode(&find(workspace, decode(method, request)?)?),
        _ => Err(Error(format!(
            "attribute query {method:?}: expected get, list, listSets, or find"
        ))),
    }
}

fn owned<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    id: u64,
) -> Result<&DocumentAttributes, Error> {
    Ok(&workspace
        .session(id)
        .map_err(|error| Error(error.to_string()))?
        .snapshot()
        .attributes)
}

fn list(
    attributes: &DocumentAttributes,
    request: ListAttributesArgs,
) -> Result<ListAttributesResult, Error> {
    let selected = target::key(&request.target)?;
    let mut entries = Vec::new();
    for anchor in attributes
        .anchors()
        .iter()
        .filter(|anchor| request.all_targets || anchor.target == selected)
    {
        let target = if request.all_targets {
            target::name(&anchor.target)?
        } else {
            request.target.clone()
        };
        entries.extend(
            anchor
                .sets()
                .iter()
                .filter(|set| request.set.is_empty() || set.name == request.set)
                .flat_map(|set| {
                    set.attributes()
                        .iter()
                        .map(|attribute| info(&set.name, &target, attribute))
                }),
        );
    }
    Ok(ListAttributesResult {
        attributes: (!entries.is_empty()).then_some(entries),
    })
}

fn sets(attributes: &DocumentAttributes) -> Result<ListAttributeSetsResult, Error> {
    let document = target::key("")?;
    let mut names: Vec<String> = attributes
        .anchors()
        .iter()
        .filter(|anchor| anchor.target == document)
        .flat_map(|anchor| anchor.sets().iter().map(|set| set.name.clone()))
        .collect();
    names.sort();
    Ok(ListAttributeSetsResult { sets: Some(names) })
}

fn find<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    request: FindByAttributeArgs,
) -> Result<FindByAttributeResult, Error> {
    if request.set.is_empty() {
        return Err(Error(
            "attribute set \"\": expected a nonempty set name".into(),
        ));
    }
    let mut matches = Vec::new();
    for &document in workspace.documents() {
        for hit in owned(workspace, document)?.find(&request.set, &request.name) {
            matches.push(AttributeMatch {
                document,
                attribute: info(&hit.set.name, &target::name(hit.target)?, hit.attribute),
            });
        }
    }
    Ok(FindByAttributeResult {
        matches: (!matches.is_empty()).then_some(matches),
    })
}
