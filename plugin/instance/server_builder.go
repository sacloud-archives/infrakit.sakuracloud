package instance

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	instance_types "github.com/sacloud/infrakit.sakuracloud/plugin/instance/types"
	"github.com/sacloud/libsacloud/api"
	"github.com/sacloud/libsacloud/builder"
	"github.com/sacloud/libsacloud/sacloud"
	"github.com/sacloud/libsacloud/sacloud/ostype"
	"io/ioutil"
	"strings"
)

func validateProp(client *api.Client, params instance_types.Properties) error {
	// validate --- for disk mode params
	errs := validateServerDiskModeParams(params)
	if len(errs) > 0 {
		return fmt.Errorf("%s", flattenErrors(errs))
	}
	// select builder
	sb := createServerBuilder(client, params)

	var validators = []func(interface{}, instance_types.Properties) []error{
		validateServerNetworkParams,
		validateServerDiskEditParams,
	}
	for _, v := range validators {
		errs := v(sb, params)
		if len(errs) > 0 {
			return fmt.Errorf("%s", flattenErrors(errs))
		}
	}
	return nil
}

func createInstance(client *api.Client, params instance_types.Properties) (*sacloud.Server, error) {

	// validate --- for disk mode params
	errs := validateServerDiskModeParams(params)
	if len(errs) > 0 {
		return nil, fmt.Errorf("%s", flattenErrors(errs))
	}

	// select builder
	sb := createServerBuilder(client, params)

	// handle build processes
	for _, handler := range serverBuildHandlers {
		err := handler(sb, params)
		if err != nil {
			return nil, err
		}
	}

	// call Create(id)
	var b = sb.(serverBuilder)
	res, err := b.Build()
	if err != nil {
		return nil, fmt.Errorf("CreateInstance is failed: %s", err)
	}

	return res.Server, nil
}

func createServerBuilder(client *api.Client, params instance_types.Properties) interface{} {
	var sb interface{}

	switch params.DiskMode {
	case "create":
		if params.SourceDiskID > 0 {
			sb = builder.ServerFromDisk(client, params.Name, params.SourceDiskID)
		} else if params.SourceArchiveID > 0 {
			sb = builder.ServerFromArchive(client, params.Name, params.SourceArchiveID)
		} else {

			if params.OSType == "" {
				sb = builder.ServerBlankDisk(client, params.Name)
			} else {
				// Windows?
				if isWindows(params.OSType) {
					sb = builder.ServerPublicArchiveWindows(client, strToOSType(params.OSType), params.Name)
				} else {
					sb = builder.ServerPublicArchiveUnix(client, strToOSType(params.OSType), params.Name, params.Password)
				}
			}
		}
	case "connect":
		sb = builder.ServerFromExistsDisk(client, params.Name, params.DiskID)
	case "diskless":
		sb = builder.ServerDiskless(client, params.Name)
	}
	return sb
}

var serverBuildHandlers = []func(interface{}, instance_types.Properties) error{
	handleNetworkParams,
	handleDiskEditParams,
	handleDiskParams,
	handleServerCommonParams,
	handleDiskEvents,
	handleServerEvents,
}

func handleNetworkParams(sb interface{}, params instance_types.Properties) error {
	// validate --- for network params
	errs := validateServerNetworkParams(sb, params)
	if len(errs) > 0 {
		return fmt.Errorf("%s", flattenErrors(errs))
	}

	// set network params
	if sb, ok := sb.(serverNetworkParams); ok {
		switch params.NetworkMode {
		case "shared":
			sb.AddPublicNWConnectedNIC()
		case "switch":
			switch sb := sb.(type) {
			case serverConnectSwitchParam:
				sb.AddExistsSwitchConnectedNIC(fmt.Sprintf("%d", params.SwitchID))
			case serverConnectSwitchParamWithEditableDisk:
				sb.AddExistsSwitchConnectedNIC(
					fmt.Sprintf("%d", params.SwitchID),
					params.IPAddress,
					params.NwMasklen,
					params.DefaultRoute,
				)
			default:
				panic(fmt.Errorf("This server builder Can't connect to switch : %#v", sb))
			}

		case "disconnect":
			sb.AddDisconnectedNIC()
		case "none":
		// noop
		default:
			panic(fmt.Errorf("Unknown NetworkMode : %s", params.NetworkMode))
		}

		sb.SetUseVirtIONetPCI(params.UseNicVirtIO)
		if params.PacketFilterID != sacloud.EmptyID {
			sb.SetPacketFilterIDs([]int64{params.PacketFilterID})
		}
	}

	return nil
}

func handleDiskEditParams(sb interface{}, params instance_types.Properties) error {
	// validate --- for disk params
	errs := validateServerDiskEditParams(sb, params)
	if len(errs) > 0 {
		return fmt.Errorf("%s", flattenErrors(errs))
	}

	// set disk edit params
	if sb, ok := sb.(serverEditDiskParam); ok {
		sb.SetHostName(params.Hostname)
		sb.SetPassword(params.Password)
		sb.SetDisablePWAuth(params.DisablePasswordAuth)

		for _, v := range params.StartupScriptIDs {
			sb.AddNoteID(v)
		}
		for _, v := range params.StartupScripts {
			sb.AddNote(v)
		}
		sb.SetNotesEphemeral(params.StartupScriptsEphemeral)

		for _, v := range params.SSHKeyIDs {
			sb.AddSSHKeyID(v)
		}
		// pubkey(text)
		for _, v := range params.SSHKeyPublicKeys {
			sb.AddSSHKey(v)
		}
		// pubkey(from file)
		for _, v := range params.SSHKeyPublicKeyFiles {
			b, err := ioutil.ReadFile(v)
			if err != nil {
				return fmt.Errorf("CreateInstance is failed: %s", err)
			}
			sb.AddSSHKey(string(b))

		}
		sb.SetSSHKeysEphemeral(params.SSHKeyEphemeral)

	}
	return nil
}

func handleDiskParams(sb interface{}, params instance_types.Properties) error {
	// set disk params
	if sb, ok := sb.(serverDiskParams); ok {
		sb.SetDiskPlan(params.DiskPlan)
		sb.SetDiskConnection(sacloud.EDiskConnection(params.DiskConnection))
		sb.SetDiskSize(params.DiskSize)
		sb.SetDistantFrom(params.DistantFrom)
	}

	return nil
}

func handleServerCommonParams(sb interface{}, params instance_types.Properties) error {
	// set common params
	var b serverBuilder
	b, ok := sb.(serverBuilder)
	if !ok {
		panic(fmt.Errorf("CreateInstance is failed: %s", "ServerBuilder not implements common property."))
	}

	tags := params.Tags

	b.SetCore(params.Core)
	b.SetMemory(params.Memory)
	b.SetServerName(params.Name)
	b.SetDescription(params.Description)
	if params.UsKeyboard {
		tags = append(tags, sacloud.TagKeyboardUS)
	}
	b.SetTags(tags)
	b.SetIconID(params.IconID)
	b.SetISOImageID(params.ISOImageID)
	return nil
}

func handleDiskEvents(sb interface{}, params instance_types.Properties) error {
	// set events
	if diskEventBuilder, ok := sb.(serverDiskEventParam); ok {
		diskEventBuilder.SetDiskEventHandler(builder.DiskBuildOnCreateDiskBefore, func(value *builder.DiskBuildValue, result *builder.DiskBuildResult) {
			log.Debugln("CreateDisk:start")
		})
		diskEventBuilder.SetDiskEventHandler(builder.DiskBuildOnCreateDiskAfter, func(value *builder.DiskBuildValue, result *builder.DiskBuildResult) {
			log.Debugln("CreateDisk:finish")
		})

		// edit disk
		diskEventBuilder.SetDiskEventHandler(builder.DiskBuildOnEditDiskBefore, func(value *builder.DiskBuildValue, result *builder.DiskBuildResult) {
			log.Debugln("EditDisk:start")
		})
		diskEventBuilder.SetDiskEventHandler(builder.DiskBuildOnEditDiskAfter, func(value *builder.DiskBuildValue, result *builder.DiskBuildResult) {
			log.Debugln("EditDisk:finish")
		})

		// cleanup startup script
		diskEventBuilder.SetDiskEventHandler(builder.DiskBuildOnCleanupNoteBefore, func(value *builder.DiskBuildValue, result *builder.DiskBuildResult) {
			log.Debugln("Cleanup StartupScript:start")
		})
		diskEventBuilder.SetDiskEventHandler(builder.DiskBuildOnCleanupNoteAfter, func(value *builder.DiskBuildValue, result *builder.DiskBuildResult) {
			log.Debugln("Cleanup StartupScript:finish")
		})

		// cleanup ssh key script
		diskEventBuilder.SetDiskEventHandler(builder.DiskBuildOnCleanupSSHKeyBefore, func(value *builder.DiskBuildValue, result *builder.DiskBuildResult) {
			log.Debugln("Cleanup SSHKey:start")
		})
		diskEventBuilder.SetDiskEventHandler(builder.DiskBuildOnCleanupSSHKeyAfter, func(value *builder.DiskBuildValue, result *builder.DiskBuildResult) {
			log.Debugln("Cleanup SSHKey:finish")
		})
	}

	return nil
}

func handleServerEvents(sb interface{}, params instance_types.Properties) error {
	if serverEventBuilder, ok := sb.(serverEventparam); ok {
		serverEventBuilder.SetEventHandler(builder.ServerBuildOnCreateServerBefore, func(value *builder.ServerBuildValue, result *builder.ServerBuildResult) {
			log.Debugln("Create Server:start")
		})
		serverEventBuilder.SetEventHandler(builder.ServerBuildOnCreateServerAfter, func(value *builder.ServerBuildValue, result *builder.ServerBuildResult) {
			log.Debugln("Create Server:finish")
		})

		serverEventBuilder.SetEventHandler(builder.ServerBuildOnBootBefore, func(value *builder.ServerBuildValue, result *builder.ServerBuildResult) {
			log.Debugln("Boot Server:start")
		})
		serverEventBuilder.SetEventHandler(builder.ServerBuildOnBootAfter, func(value *builder.ServerBuildValue, result *builder.ServerBuildResult) {
			log.Debugln("Boot Server:finish")
		})

	}
	return nil
}

func validateServerDiskModeParams(params instance_types.Properties) []error {

	var errs []error
	var appendErrors = func(e []error) {
		errs = append(errs, e...)
	}
	var validateIfCtxIsSet = func(baseParamName string, baseParamValue interface{}, targetParamName string, targetValue interface{}) {
		if !isEmpty(targetValue) {
			appendErrors(validateConflictValues(baseParamName, baseParamValue, map[string]interface{}{
				targetParamName: targetValue,
			}))
		}
	}

	switch params.DiskMode {
	case "create":
		// check required values
		appendErrors(validateRequired("DiskPlan", params.DiskPlan))
		appendErrors(validateRequired("DiskConnection", params.DiskConnection))
		appendErrors(validateRequired("DiskSize", params.DiskSize))

		if params.SourceDiskID == 0 && params.SourceArchiveID == 0 {

		} else {
			validateIfCtxIsSet("SourceArchiveID", params.SourceArchiveID, "OSType", params.OSType)
			validateIfCtxIsSet("SourceArchiveID", params.SourceArchiveID, "OSType", params.OSType)
		}

		validateIfCtxIsSet("DiskMode", params.DiskMode, "DiskID", params.DiskID)

	case "connect":
		appendErrors(validateRequired("DiskID", params.DiskID))
		validateIfCtxIsSet("DiskMode", params.DiskMode, "DiskPlan", params.DiskPlan)
		validateIfCtxIsSet("DiskMode", params.DiskMode, "DiskConnection", params.DiskConnection)
		validateIfCtxIsSet("DiskSize", params.DiskMode, "DiskSize", params.DiskSize)
		validateIfCtxIsSet("DiskSize", params.DiskMode, "OSType", params.OSType)

	case "diskless":
		validateIfCtxIsSet("DiskMode", params.DiskMode, "DiskID", params.DiskID)
		validateIfCtxIsSet("DiskMode", params.DiskMode, "DiskPlan", params.DiskPlan)
		validateIfCtxIsSet("DiskMode", params.DiskMode, "DiskConnection", params.DiskConnection)
		validateIfCtxIsSet("DiskSize", params.DiskMode, "DiskSize", params.DiskSize)
		validateIfCtxIsSet("DiskSize", params.DiskMode, "OSType", params.OSType)
	}

	return errs
}

func validateServerNetworkParams(sb interface{}, params instance_types.Properties) []error {
	var errs []error
	var appendErrors = func(e []error) {
		errs = append(errs, e...)
	}
	var validateIfCtxIsSet = func(baseParamName string, baseParamValue interface{}, targetParamName string, targetValue interface{}) {
		if !isEmpty(targetValue) {
			appendErrors(validateConflictValues(baseParamName, baseParamValue, map[string]interface{}{
				targetParamName: targetValue,
			}))
		}
	}
	var validateProhibitedIfCtxIsSet = func(paramName string, paramValue interface{}) {
		if !isEmpty(paramValue) {
			appendErrors(validateSetProhibited(paramName, paramValue))
		}
	}

	if sb, ok := sb.(serverNetworkParams); ok {
		switch params.NetworkMode {
		case "shared", "disconnect", "none":
			validateIfCtxIsSet("NetworkMode", params.NetworkMode, "SwitchID", params.SwitchID)
			validateIfCtxIsSet("NetworkMode", params.NetworkMode, "IPAddress", params.IPAddress)
			validateIfCtxIsSet("NetworkMode", params.NetworkMode, "NwMasklen", params.NwMasklen)
			validateIfCtxIsSet("NetworkMode", params.NetworkMode, "DefaultRoute", params.DefaultRoute)

			if params.NetworkMode == "none" {
				validateIfCtxIsSet("NetworkMode", params.NetworkMode, "UseNicVirtIO", params.UseNicVirtIO)
				validateIfCtxIsSet("NetworkMode", params.NetworkMode, "PacketDilterID", params.PacketFilterID)
			}

		case "switch":
			switch sb.(type) {
			case serverConnectSwitchParam:
				appendErrors(validateRequired("SwitchID", params.SwitchID))

				validateProhibitedIfCtxIsSet("IPAddress", params.IPAddress)
				validateProhibitedIfCtxIsSet("NwMasklen", params.NwMasklen)
				validateProhibitedIfCtxIsSet("DefaultRoute", params.DefaultRoute)

			case serverConnectSwitchParamWithEditableDisk:

				appendErrors(validateRequired("SwitchID", params.SwitchID))
			}
		}

	} else {
		validateProhibitedIfCtxIsSet("NetworkMode", params.NetworkMode)
		validateProhibitedIfCtxIsSet("SwitchID", params.SwitchID)
		validateProhibitedIfCtxIsSet("IPAddress", params.IPAddress)
		validateProhibitedIfCtxIsSet("NwMasklen", params.NwMasklen)
		validateProhibitedIfCtxIsSet("DefaultRoute", params.DefaultRoute)
		validateProhibitedIfCtxIsSet("UseNicVirtIO", params.UseNicVirtIO)
		validateProhibitedIfCtxIsSet("PacketFilterID", params.PacketFilterID)
	}

	return errs
}

func validateServerDiskEditParams(sb interface{}, params instance_types.Properties) []error {
	var errs []error
	var appendErrors = func(e []error) {
		errs = append(errs, e...)
	}

	var validateProhibitedIfCtxIsSet = func(paramName string, paramValue interface{}) {
		if !isEmpty(paramValue) {
			appendErrors(validateSetProhibited(paramName, paramValue))
		}
	}

	if _, ok := sb.(serverEditDiskParam); !ok {
		validateProhibitedIfCtxIsSet("Hostname", params.Hostname)
		validateProhibitedIfCtxIsSet("Password", params.Password)
		validateProhibitedIfCtxIsSet("DisablePasswordAuth", params.DisablePasswordAuth)
		validateProhibitedIfCtxIsSet("StartupScriptIDs", params.StartupScriptIDs)
		validateProhibitedIfCtxIsSet("StartupScripts", params.StartupScripts)
		validateProhibitedIfCtxIsSet("StartupScriptsEphemeral", params.StartupScriptsEphemeral)
		validateProhibitedIfCtxIsSet("SSHKeyIDs", params.SSHKeyIDs)
		validateProhibitedIfCtxIsSet("SSHKeyPublicKeys", params.SSHKeyPublicKeys)
		validateProhibitedIfCtxIsSet("SSHKeyPublicKeyFiles", params.SSHKeyPublicKeyFiles)
		validateProhibitedIfCtxIsSet("SSHKeyEphemeral", params.SSHKeyEphemeral)
	}

	return errs
}

func isWindows(osType string) bool {
	return strToOSType(osType).IsWindows()
}

func strToOSType(strOSType string) ostype.ArchiveOSTypes {
	return ostype.StrToOSType(strOSType)
}

type serverBuilder interface {
	SetCore(int)
	SetMemory(int)
	SetServerName(string)
	SetDescription(string)
	SetTags([]string)
	SetIconID(int64)
	SetBootAfterCreate(bool)
	SetISOImageID(int64)

	Build() (*builder.ServerBuildResult, error)
}

type serverDiskParams interface {
	SetDiskPlan(string)
	SetDiskConnection(sacloud.EDiskConnection)
	SetDiskSize(int)
	SetDistantFrom([]int64)
}

type serverNetworkParams interface {
	SetUseVirtIONetPCI(bool)
	SetPacketFilterIDs([]int64)
	AddPublicNWConnectedNIC()
	AddDisconnectedNIC()
}

type serverConnectSwitchParamWithEditableDisk interface {
	AddExistsSwitchConnectedNIC(id string, ipaddress string, maskLen int, defRoute string)
}

type serverConnectSwitchParam interface {
	AddExistsSwitchConnectedNIC(id string)
}

type serverEditDiskParam interface {
	SetHostName(string)
	SetPassword(string)
	SetDisablePWAuth(bool)
	AddNote(string)
	AddNoteID(int64)
	SetNotesEphemeral(bool)
	AddSSHKey(string)
	AddSSHKeyID(int64)
	SetSSHKeysEphemeral(bool)
	SetGenerateSSHKeyName(string)
	SetGenerateSSHKeyPassPhrase(string)
	SetGenerateSSHKeyDescription(string)
}

type serverDiskEventParam interface {
	SetDiskEventHandler(event builder.DiskBuildEvents, handler builder.DiskBuildEventHandler)
}

type serverEventparam interface {
	SetEventHandler(event builder.ServerBuildEvents, handler builder.ServerBuildEventHandler)
}

func flattenErrors(errors []error) error {
	if len(errors) == 0 {
		return nil
	}
	var list = make([]string, 0)
	for _, str := range errors {
		list = append(list, str.Error())
	}
	return fmt.Errorf(strings.Join(list, "\n"))
}
