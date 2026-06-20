// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// DeleteBodyData persists a delete-body feature (#1078): the reference key of the body it removes,
// base64-encoded like the other key references.
type DeleteBodyData struct {
	Body string `yaml:"body"`
}

func serializeDeleteBody(def *DeleteBodyDefinition) *DeleteBodyData {
	return &DeleteBodyData{Body: encodeKey(def.BodyKey)}
}

func restoreDeleteBody(fs *PartFeatures, d *DeleteBodyData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("delete-body feature is missing its payload")
	}
	key, err := decodeKey(d.Body)
	if err != nil {
		return nil, fmt.Errorf("delete-body: bad body key: %w", err)
	}
	return fs.AddDeleteBody(key), nil
}
