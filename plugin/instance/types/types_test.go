package types

import (
	"testing"

	"github.com/docker/infrakit/pkg/spi/instance"
	"github.com/docker/infrakit/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestParsePropertiesFail(t *testing.T) {
	properties := types.AnyString(`{
	  "NamePrefix": "bar",
	  "tags": {
	    "foo": "bar",
	  }
	}`)

	_, err := ParseProperties(properties)
	assert.Error(t, err)
}

func TestParseTags(t *testing.T) {
	id := instance.LogicalID("foo")
	spec := instance.Spec{
		Tags: map[string]string{
			"foo":    "bar",
			"banana": "",
		},
		LogicalID: &id,
	}

	tags := ParseTags(spec)
	assert.Equal(t, map[string]string{
		"foo":                      "bar",
		"banana":                   "",
		InfrakitLogicalID:          string(id),
		InfrakitSakuraCloudVersion: InfrakitSakuraCloudCurrentVersion,
	}, tags)
}
