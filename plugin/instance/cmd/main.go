package main

import (
	"os"
	"strings"

	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/pkg/cli"
	instance_plugin "github.com/docker/infrakit/pkg/rpc/instance"
	"github.com/sacloud/infrakit.sakuracloud/plugin/instance"
	"github.com/sacloud/infrakit.sakuracloud/version"
	"github.com/sacloud/libsacloud/api"
	"github.com/spf13/cobra"
)

func main() {

	cmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "SakuraCloud instance plugin",
	}
	name := cmd.Flags().String("name", "instance-sakuracloud", "Plugin name to advertise for discovery")
	logLevel := cmd.Flags().Int("log", cli.DefaultLogLevel, "Logging level. 0 is least verbose. Max is 5")
	namespaceTags := cmd.Flags().StringSlice("namespace-tags", []string{},
		"A list of key=value resource tags to namespace all resources created")

	accessToken := cmd.Flags().String("token", "", "SakuraCloud token")
	accessSecret := cmd.Flags().String("secret", "", "SakuraCloud secret")
	zone := cmd.Flags().String("zone", "is1b", "SakuraCloud zone")

	if accessToken == nil || *accessToken == "" {
		v := os.Getenv("SAKURACLOUD_ACCESS_TOKEN")
		accessToken = &v
	}
	if accessSecret == nil || *accessSecret == "" {
		v := os.Getenv("SAKURACLOUD_ACCESS_TOKEN_SECRET")
		accessSecret = &v
	}
	if zone == nil || *zone == "" {
		v := os.Getenv("SAKURACLOUD_ZONE")
		zone = &v
	}

	cmd.Run = func(c *cobra.Command, args []string) {
		cli.SetLogLevel(*logLevel)

		namespace := map[string]string{}
		for _, tagKV := range *namespaceTags {
			kv := strings.Split(tagKV, "=")
			if len(kv) != 2 {
				log.Errorln("Namespace tags must be formatted as key=value")
				os.Exit(1)
			}
			namespace[kv[0]] = kv[1]
		}

		requires := map[string]*string{
			"token":  accessToken,
			"secret": accessSecret,
			"zone":   zone,
		}
		for k, v := range requires {
			if v == nil || *v == "" {
				log.Errorln("%q is required", k)
				os.Exit(1)
			}
		}

		client := api.NewClient(*accessToken, *accessSecret, *zone)

		client.UserAgent = fmt.Sprintf("infrakit-instance-sakuracloud:%s", version.Version)

		cli.RunPlugin(*name, instance_plugin.PluginServer(instance.NewSakuraCloudInstancePlugin(client, namespace)))
	}

	cmd.AddCommand(cli.VersionCommand())

	err := cmd.Execute()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
