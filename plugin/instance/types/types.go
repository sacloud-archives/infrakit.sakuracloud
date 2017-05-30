package types

import (
	"github.com/docker/infrakit/pkg/spi/instance"
	"github.com/docker/infrakit/pkg/types"
	"github.com/pkg/errors"
)

const (
	// InfrakitLogicalID is a metadata key that is used to tag instances created with a LogicalId.
	InfrakitLogicalID = "infrakit-logical-id"

	// InfrakitSakuraCloudVersion is a metadata key that is used to know which version of the plugin was used to create
	// the instance.
	InfrakitSakuraCloudVersion = "infrakit-sakuracloud-version"

	// InfrakitSakuraCloudCurrentVersion is incremented each time the plugin introduces incompatibilities with previous
	// versions
	InfrakitSakuraCloudCurrentVersion = "1"
)

// Properties is the configuration schema for the plugin, provided in instance.Spec.Properties
type Properties struct {
	NamePrefix      string
	Core            int
	Memory          int
	DiskMode        string
	OSType          string
	DiskPlan        string
	DiskConnection  string
	DiskSize        int
	SourceArchiveID int64
	SourceDiskID    int64

	DistantFrom []int64
	DiskID      int64

	ISOImageID     int64
	UseNicVirtIO   bool
	PacketFilterID int64

	Hostname            string
	Password            string
	DisablePasswordAuth bool

	NetworkMode  string
	SwitchID     int64
	IPAddress    string
	NwMasklen    int
	DefaultRoute string

	StartupScripts          []string
	StartupScriptIDs        []int64
	StartupScriptsEphemeral bool

	SSHKeyIDs            []int64
	SSHKeyPublicKeys     []string
	SSHKeyPublicKeyFiles []string
	SSHKeyEphemeral      bool

	Name        string
	Description string
	Tags        []string
	IconID      int64
	UsKeyboard  bool
}

// ParseProperties parses instance Properties from a json description.
func ParseProperties(req *types.Any) (Properties, error) {
	parsed := Properties{
		Core:                    1,
		Memory:                  1,
		DiskMode:                "create",
		DiskPlan:                "ssd",
		DiskConnection:          "virtio",
		DiskSize:                20,
		NetworkMode:             "shared",
		UseNicVirtIO:            true,
		StartupScriptsEphemeral: true,
		SSHKeyEphemeral:         true,
	}

	if err := req.Decode(&parsed); err != nil {
		return parsed, errors.Wrap(err, "invalid properties")
	}
	return parsed, nil
}

// ParseTags returns a key/value map from the instance specification.
func ParseTags(spec instance.Spec) map[string]string {
	tags := make(map[string]string)

	for k, v := range spec.Tags {
		tags[k] = v
	}

	// Do stuff with proprerties here

	if spec.LogicalID != nil {
		tags[InfrakitLogicalID] = string(*spec.LogicalID)
	}

	tags[InfrakitSakuraCloudVersion] = InfrakitSakuraCloudCurrentVersion

	return tags
}
