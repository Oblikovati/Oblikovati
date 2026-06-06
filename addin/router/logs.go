// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati/api/wire"

	"oblikovati/app"
)

// logsTail serves wire.MethodLogsTail: a cursor-paged tail of the operation trace. It reads
// the router's own [trace.Buffer] (the one Handle fills), so a driver can poll for new records
// with the previous NextSeq and watch the kernel work in real time. It is registered like any
// method but is excluded from self-tracing in Router.record.
func (r *Router) logsTail(_ *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.LogsTailArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	return json.Marshal(r.trace.Tail(in.SinceSeq, in.Level, in.Max))
}
