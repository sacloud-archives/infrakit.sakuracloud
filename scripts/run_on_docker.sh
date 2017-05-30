#!/bin/bash

docker run -it --rm \
  --name infrakit-instance-sakuracloud \
  -e SAKURACLOUD_ACCESS_TOKEN \
  -e SAKURACLOUD_ACCESS_TOKEN_SECRET \
  -e SAKURACLOUD_DEFAULT_ZONE \
  -e SAKURACLOUD_TRACE_MODE \
  build-infrakit-instance-sakuracloud:latest $@