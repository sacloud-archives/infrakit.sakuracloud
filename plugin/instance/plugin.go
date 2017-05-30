package instance

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/pkg/spi"
	"github.com/docker/infrakit/pkg/spi/instance"
	"github.com/docker/infrakit/pkg/types"
	instance_types "github.com/sacloud/infrakit.sakuracloud/plugin/instance/types"
	"github.com/sacloud/infrakit.sakuracloud/version"
	"github.com/sacloud/libsacloud/api"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Spec is just whatever that can be unmarshalled into a generic JSON map
type Spec map[string]interface{}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

type builderAPI interface {
	Build()
}

type plugin struct {
	client        *api.Client
	namespaceTags map[string]string
}

// NewSakuraCloudInstancePlugin creates a new SakuraCloud instance plugin
func NewSakuraCloudInstancePlugin(client *api.Client, namespace map[string]string) instance.Plugin {

	return &plugin{
		client:        client,
		namespaceTags: namespace,
	}
}

// Info returns a vendor specific name and version
func (p *plugin) VendorInfo() *spi.VendorInfo {
	return &spi.VendorInfo{
		InterfaceSpec: spi.InterfaceSpec{
			Name:    "infrakit-instance-sakuracloud",
			Version: version.Version,
		},
		URL: "https://github.com/sacloud/infrakit.sakuracloud",
	}
}

// Validate performs local validation on a provision request.
func (p *plugin) Validate(req *types.Any) error {
	log.Debugln("validate", req.String())

	spec := Spec{}
	if err := req.Decode(&spec); err != nil {
		return err
	}

	properties, err := instance_types.ParseProperties(req)
	if err != nil {
		return err
	}

	err = validateProp(p.client, properties)
	if err != nil {
		return err
	}

	log.Debugln("Validated:", spec)
	return nil
}

// Label labels the instance
func (p *plugin) Label(instance instance.ID, labels map[string]string) error {
	log.Debugf("label instance %s with %v", instance, labels)
	id, err := strconv.ParseInt(string(instance), 10, 64)
	if err != nil {
		return err
	}
	server, err := p.client.Server.Read(id)
	if err != nil {
		return err
	}

	tags := doTags(mapToStringSlice(labels))
	server.Description = tags

	_, err = p.client.Server.Update(id, server)
	if err != nil {
		return err
	}

	return nil
}

// Provision creates a new instance based on the spec.
func (p *plugin) Provision(spec instance.Spec) (*instance.ID, error) {
	properties, err := instance_types.ParseProperties(spec.Properties)
	if err != nil {
		return nil, err
	}

	// the name must be given suffix
	properties.Name = fmt.Sprintf("%s-%s", properties.NamePrefix, randomSuffix(6))

	// tags to include namespace tags and injected tags
	tags := instance_types.ParseTags(spec)
	_, tags = mergeTags(tags, sliceToMap(properties.Tags), p.namespaceTags) // scope this resource with namespace tags
	properties.Description = doTags(mapToStringSlice(tags))

	// Set init script
	if spec.Init != "" {
		properties.StartupScripts = append(properties.StartupScripts, fmt.Sprintf(startupScriptTemplate, spec.Init))
	}

	res, err := createInstance(p.client, properties)
	if err != nil {
		return nil, err
	}
	id := instance.ID(res.GetStrID())
	return &id, nil
}

// Destroy terminates an existing instance.
func (p *plugin) Destroy(instance instance.ID) error {
	id, err := strconv.ParseInt(string(instance), 10, 64)
	if err != nil {
		return err
	}

	api := p.client.GetServerAPI()

	s, err := api.Read(id)
	if err != nil {
		return fmt.Errorf("Destroy is failed: %s", err)
	}

	if s.IsUp() {

		_, err = api.Stop(id)
		if err != nil {
			return fmt.Errorf("Destroy is failed: %s", err)
		}

		err = api.SleepUntilDown(id, p.client.DefaultTimeoutDuration)
		if err != nil {
			return fmt.Errorf("Destroy is failed: %s", err)
		}
	}

	// call Delete(id)
	if len(s.Disks) > 0 {
		_, err = api.DeleteWithDisk(id, s.GetDiskIDs())
		if err != nil {
			return fmt.Errorf("Destroy is failed: %s", err)
		}
	} else {
		_, err = api.Delete(id)
		if err != nil {
			return fmt.Errorf("Destroy is failed: %s", err)
		}
	}
	return nil
}

// DescribeInstances returns descriptions of all instances matching all of the provided tags.
func (p *plugin) DescribeInstances(tags map[string]string, properties bool) ([]instance.Description, error) {
	log.Debugln("describe-instances", tags)

	_, tags = mergeTags(tags, p.namespaceTags)

	result := []instance.Description{}

	res, err := p.client.Server.Find()
	if err != nil {
		return nil, err
	}
	instances := res.Servers

	log.Debugln("total count:", len(instances))

	for _, server := range instances {
		instTags := sliceToMap(undoTags(server.Description))
		if hasDifferentTag(tags, instTags) {
			log.Debugf("Skipping %v", server.Name)
			continue
		}

		description := instance.Description{
			ID:   instance.ID(fmt.Sprintf("%d", server.ID)),
			Tags: instTags,
		}

		if properties {
			if any, err := types.AnyValue(server); err == nil {
				description.Properties = any
			} else {
				log.Warningln("error encoding instance properties:", err)
			}
		}

		result = append(result, description)
	}

	return result, nil
}

var startupScriptTemplate = `#!/bin/sh
# @sacloud-once
# @sacloud-desc provisioning by infrakit-instance-sakuracloud
%s
exit 0`

func doTags(tags []string) string {
	return strings.Join(tags, "\n")
}

func undoTags(tags string) []string {
	return strings.Split(tags, "\n")
}

// mergeTags merges multiple maps of tags, implementing 'last write wins' for colliding keys.
// Returns a sorted slice of all keys, and the map of merged tags.  Sorted keys are particularly useful to assist in
// preparing predictable output such as for tests.
func mergeTags(tagMaps ...map[string]string) ([]string, map[string]string) {
	keys := []string{}
	tags := map[string]string{}
	for _, tagMap := range tagMaps {
		for k, v := range tagMap {
			if _, exists := tags[k]; exists {
				log.Warnf("Overwriting tag value for key %s", k)
			} else {
				keys = append(keys, k)
			}
			tags[k] = v
		}
	}
	sort.Strings(keys)
	return keys, tags
}

func mapToStringSlice(m map[string]string) []string {
	s := []string{}
	for key, value := range m {
		if value != "" {
			s = append(s, key+":"+value)
		} else {
			s = append(s, key)
		}
	}
	return s
}

func sliceToMap(s []string) map[string]string {
	m := map[string]string{}
	for _, v := range s {
		parts := strings.SplitN(v, ":", 2)
		switch len(parts) {
		case 1:
			m[parts[0]] = ""
		case 2:
			m[parts[0]] = parts[1]
		}
	}
	return m
}

func hasDifferentTag(expected, actual map[string]string) bool {
	if len(actual) == 0 {
		return true
	}
	for k, v := range expected {
		if a, ok := actual[k]; ok && a != v {
			return true
		}
	}

	return false
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

// RandomSuffix generate a random instance name suffix of length `n`.
func randomSuffix(n int) string {
	suffix := make([]rune, n)

	for i := range suffix {
		suffix[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	return string(suffix)
}
