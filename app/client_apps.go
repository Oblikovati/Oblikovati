// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/wire"
)

// ClientApplicationRegistry tracks external client applications driving the session
// — out-of-process automation drivers (an MCP client, a test harness) that announce
// themselves so the session can list who is connected. Distinct from in-process
// add-ins ([AddInManager]); the ClientApplications equivalent (M05-F01, #245).
type ClientApplicationRegistry struct {
	nextID int
	order  []int
	names  map[int]string
}

// NewClientApplicationRegistry returns an empty external-client registry.
func NewClientApplicationRegistry() *ClientApplicationRegistry {
	return &ClientApplicationRegistry{nextID: 1, names: map[int]string{}}
}

// Register announces an external client and returns its session-unique id.
func (r *ClientApplicationRegistry) Register(name string) (int, error) {
	if name == "" {
		return 0, fmt.Errorf("app: client application name is empty; expected a display name like \"acme-pipeline\"")
	}
	id := r.nextID
	r.nextID++
	r.names[id] = name
	r.order = append(r.order, id)
	return id, nil
}

// Unregister removes a previously registered external client by its id.
func (r *ClientApplicationRegistry) Unregister(id int) error {
	if _, ok := r.names[id]; !ok {
		return fmt.Errorf("app: no client application %d", id)
	}
	delete(r.names, id)
	for i, x := range r.order {
		if x == id {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
	return nil
}

// List returns the registered external clients in registration order.
func (r *ClientApplicationRegistry) List() []wire.ClientApplicationInfo {
	out := make([]wire.ClientApplicationInfo, len(r.order))
	for i, id := range r.order {
		out[i] = wire.ClientApplicationInfo{ID: id, Name: r.names[id]}
	}
	return out
}
