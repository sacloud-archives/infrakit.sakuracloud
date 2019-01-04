# Infrakit.SakuraCloud

[InfraKit](https://github.com/docker/infrakit) plugins for creating and managing resources in [SakuraCloud](http://cloud.sakura.ad.jp/).

**This project is still under development**

## Instance plugin

An InfraKit instance plugin which creates SakuraCloud servers.

### Building

To build the instance plugin, run `make build` or `make docker-build`. The plugin binary
will be located at `./build/infrakit-instance-sakuracloud`.

### Running

```
${PATH_TO_INFRAKIT}/infrakit-flavor-vanilla
${PATH_TO_INFRAKIT}/infrakit-group-default
./build/infrakit-instance-sakuracloud --token=[YOUR API TOEKN] --secret=[YOUR API SECRET] --zone=[TARGET ZONE]

${PATH_TO_INFRAKIT}/infrakit group commit sakuracloud-exemple01.json
```

NOTE: Following parameters are able to also set by environment variable.

| Parameter  | Environment Variable              |
|------------|-----------------------------------|
| `--token`  | `SAKURACLOUD_ACCESS_TOKEN`        |
| `--secret` | `SAKURACLOUD_ACCESS_TOKEN_SECRET` |
| `--zone`   | `SAKURACLOUD_ZONE`                |

## JSON example(with group-default and flavor-vanilla)


```js
{
  "ID": "instances",
  "Properties": {
    "Allocation": {
      "Size": 2
    },
    "Instance": {
      "Plugin": "instance-sakuracloud",
      "Properties": {
        "NamePrefix": "infrakit-sakuracloud",
        "Tags": ["ci-infrakit-sakuracloud"],
        "OSType": "centos",
        "SSHKeyPublicKeyFiles": ["/infrakit/id_rsa.pub"],
        "DisablePasswordAuth": true
      }
    },
    "Flavor": {
      "Plugin": "flavor-vanilla",
      "Properties": {
        "Init": [
          "sh -c \"echo 'Hello, World' > /hello\""
        ]
      }
    }
  }
}

```

## Instance Properties

Following parameters are available.

- `NamePrefix` 
- `Core`: (default: 1)
- `Memory`: GB(default: 1)
- `DiskMode`: [`create` or `connect` or `diskless`]
- `OSType` : see [OSType values](#param_ostype)
- `DiskPlan`: [`ssd` or `hdd`]
- `DiskConnection`: [`virtio` or `ide`]
- `DiskSize` : GB(default: 20)
- `SourceArchiveID`
- `SourceDiskID`
- `DistantFrom`
- `DiskID`
- `ISOImageID`
- `UseNicVirtIO` : (default: true)
- `PacketFilterID`
- `Hostname`
- `Password`
- `DisablePasswordAuth`: (default: false)
- `NetworkMode`: [`shared` or `switch` or `disconnect` or `none`](default: shared)
- `SwitchID`
- `IPAddress`
- `NwMasklen`
- `DefaultRoute`
- `StartupScripts`
- `StartupScriptIDs`
- `StartupScriptsEphemeral`: (default: true)
- `SSHKeyIDs`
- `SSHKeyPublicKeys`
- `SSHKeyPublicKeyFiles`
- `SSHKeyEphemeral`: (default: true)
- `Name`
- `Tags`
- `IconID`
- `UsKeyboard`: (default: false)

<a id="param_ostype"></a>
### OSType values

|value | Public Archive                          |
|---------------------------|--------------------|
| `centos`                  | CentOS 7|
| `ubuntu`                  | Ubuntu 16.04|
| `debian`                  | Debian |
| `vyos`                    | VyOS|
| `coreos`                  | CoreOS|
| `rancheros`               | RancherOS|
| `kusanagi`                | Kusanagi(CentOS7)|
| `site-guard`              | SiteGuard(CentOS7)|
| `plesk`                   | Plesk(CentOS7)|
| `freebsd`                 | FreeBSD|

## License

 `infrakit-instance-sakuracloud` Copyright (C) 2017-2019 Kazumichi Yamamoto.

  This project is published under [Apache 2.0 License](LICENSE.txt).
  
## Author

  * Kazumichi Yamamoto ([@yamamoto-febc](https://github.com/yamamoto-febc))
