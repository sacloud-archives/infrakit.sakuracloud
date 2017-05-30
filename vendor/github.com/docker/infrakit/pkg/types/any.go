package types

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
)

// Any is the raw configuration for the plugin
type Any json.RawMessage

// AnyString returns an Any from a string that represents the marshaled/encoded data
func AnyString(s string) *Any {
	return AnyBytes([]byte(s))
}

// AnyYAML constructs any Any from a yaml
func AnyYAML(y []byte) (*Any, error) {
	buff, err := yaml.YAMLToJSON(y)
	if err != nil {
		return nil, err
	}
	return AnyBytes(buff), nil
}

// AnyYAMLMust constructs any Any from a yaml, panics on error
func AnyYAMLMust(y []byte) *Any {
	any, err := AnyYAML(y)
	if err != nil {
		panic(err)
	}
	return any
}

// AnyBytes returns an Any from the encoded message bytes
func AnyBytes(data []byte) *Any {
	any := &Any{}
	*any = data
	return any
}

// AnyCopy makes a copy of the data in the given ptr.
func AnyCopy(any *Any) *Any {
	if any == nil {
		return &Any{}
	}
	return AnyBytes(any.Bytes())
}

// AnyValue returns an Any from a value by marshaling / encoding the input
func AnyValue(v interface{}) (*Any, error) {
	if v == nil {
		return nil, nil // So that any omitempty will see an empty/zero value
	}
	any := &Any{}
	err := any.marshal(v)
	return any, err
}

// AnyValueMust returns an Any from a value by marshaling / encoding the input. It panics if there's error.
func AnyValueMust(v interface{}) *Any {
	any, err := AnyValue(v)
	if err != nil {
		panic(err)
	}
	return any
}

// Decode decodes the any into the input typed struct
func (c *Any) Decode(typed interface{}) error {
	if c == nil || len([]byte(*c)) == 0 {
		return nil // no effect on typed
	}
	return json.Unmarshal([]byte(*c), typed)
}

// marshal populates this raw message with a decoded form of the input struct.
func (c *Any) marshal(typed interface{}) error {
	buff, err := json.MarshalIndent(typed, "", "")
	if err != nil {
		return err
	}
	*c = Any(json.RawMessage(buff))
	return nil
}

// Bytes returns the encoded bytes
func (c *Any) Bytes() []byte {
	if c == nil {
		return nil
	}
	return []byte(*c)
}

// String returns the string representation.
func (c *Any) String() string {
	return string([]byte(*c))
}

// MarshalJSON implements the json Marshaler interface
func (c *Any) MarshalJSON() ([]byte, error) {
	if c == nil {
		return nil, nil
	}
	return []byte(*c), nil
}

// UnmarshalJSON implements the json Unmarshaler interface
func (c *Any) UnmarshalJSON(data []byte) error {
	*c = Any(json.RawMessage(data))
	return nil
}

// MarshalYAML marshals to yaml
func (c *Any) MarshalYAML() ([]byte, error) {
	data, err := c.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return yaml.JSONToYAML(data)
}

// UnmarshalYAML decodes from yaml and populates the any
func (c *Any) UnmarshalYAML(data []byte) error {
	j, err := yaml.YAMLToJSON(data)
	if err != nil {
		return err
	}
	return c.UnmarshalJSON(j)
}

// Fingerprint returns a MD5 hash of the opague blob.  It also removes newlines and tab characters that
// are common in JSON but don't contribute to the actual content.
func Fingerprint(m ...*Any) string {
	h := md5.New()
	for _, mm := range m {
		buff := mm.Bytes()
		buff = bytes.Replace(buff, []byte(": "), []byte(":"), -1) // not really proud of this.
		buff = bytes.Replace(buff, []byte(":"), []byte(":"), -1)  // not really proud of this.
		buff = bytes.Replace(buff, []byte("\n"), nil, -1)
		buff = bytes.Replace(buff, []byte("\t"), nil, -1)
		h.Write(buff)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
