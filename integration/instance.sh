#!/usr/bin/env bash

set -e

BASEDIR=$(dirname "$0")
INFRAKIT_IMAGE=infrakit/devbundle:master-1580
docker_run="docker run -v infrakit:/infrakit -e INFRAKIT_PLUGINS_DIR=/infrakit"


cleanup() {
  echo Clean up

  docker rm -f flavor group instance-sakuracloud 2>/dev/null || true
  docker volume rm infrakit 2>/dev/null || true
  rm id_rsa* 2>/dev/null || true
}

remove_previous_instances() {
  echo Remove previous instances

  # FIXME(yamamoto) use usacloud here
  count=`usacloud server ls -q --name infrakit-sakuracloud | wc -l`
  if [ $count -gt 0 ] ; then
    usacloud server rm -f -y infrakit-sakuracloud
  fi
}


run_infrakit() {
  echo Run Infrakit

  docker volume create --name infrakit

  ssh-keygen  -C "" -f id_rsa -N ""
  $docker_run -v $PWD:/work --rm busybox cp /work/id_rsa /infrakit/
  $docker_run -v $PWD:/work --rm busybox cp /work/id_rsa.pub /infrakit/

  $docker_run -d --name=flavor ${INFRAKIT_IMAGE} infrakit-flavor-vanilla --log=5
  $docker_run -d --name=group ${INFRAKIT_IMAGE} infrakit-group-default --log=5
}

run_infrakit_sakuracloud_instance() {
  echo Run Infrakit SakuraCloud Instance Plugin

  $docker_run -d --name=instance-sakuracloud \
    -e SAKURACLOUD_ACCESS_TOKEN \
    -e SAKURACLOUD_ACCESS_TOKEN_SECRET \
    -e SAKURACLOUD_ZONE \
    infrakit-instance-sakuracloud:latest \
    infrakit-instance-sakuracloud --log=5
}

create_group() {
  echo Create Instance Group

  docker cp ${BASEDIR}/instances.json group:/infrakit/

  $docker_run --rm busybox cat /infrakit/instances.json
  $docker_run --rm ${INFRAKIT_IMAGE} infrakit group commit /infrakit/instances.json
  sleep 30
}

check_instances_created() {
  echo Check that the instances are there

  count=`usacloud server ls -q --name infrakit-sakuracloud | wc -l`
  if [ $count -ne 2 ] ; then
    echo "ERROR: Expect instances:2, Actual:${count} "
    exit 1
  fi

}

check_instance_properties() {
  echo Check that the instances are well configured

  # FIXME(yamamoto) use usacloud for this
}

delete_instances() {
  echo Delete instances

  target=`usacloud server ls -q --name infrakit-sakuracloud | head -n1`
  usacloud server wait-for-boot $target
  usacloud server rm -f -y $target
  sleep 30
}

destroy_group() {
  echo Destroy Instance Group

  $docker_run --rm ${INFRAKIT_IMAGE} infrakit group destroy instances
}

check_instances_gone() {
  echo Check that the instances are gone

  count=0

  # FIXME(yamamoto) use usacloud for this
  count=`usacloud server ls -q --name infrakit-sakuracloud | wc -l`
  if [ $count -gt 0 ] ; then
    echo "ERROR: ${count} instances are still around"
    exit 1
  fi

}

assert_contains() {
  STDIN=$(cat)
  echo "${STDIN}" | grep -q "${1}" || (echo "Expected [${STDIN}] to contain [${1}]" && return 1)
}

assert_equals() {
  STDIN=$(cat)
  [ "${STDIN}" == "${1}" ] || (echo "Expected [${1}], got [${STDIN}]" && return 1)
}

if [ -z "${SAKURACLOUD_ACCESS_TOKEN}" ] ; then
  echo "SAKURACLOUD_ACCESS_TOKEN is requied"
  exit 1
fi

if [ -z "${SAKURACLOUD_ACCESS_TOKEN_SECRET}" ] ; then
  echo "SAKURACLOUD_ACCESS_TOKEN_SECRET is requied"
  exit 1
fi

if [ -z "${SAKURACLOUD_ZONE}" ] ; then
  echo "SAKURACLOUD_ZONE is requied"
  exit 1
fi

# which usacloud > /dev/null || echo "usacloud command is required" && exit 1

echo "Integration Test:Start"

cleanup
remove_previous_instances
run_infrakit
run_infrakit_sakuracloud_instance
create_group
check_instances_created
check_instance_properties
delete_instances
check_instances_created
check_instance_properties
destroy_group
check_instances_gone
cleanup

echo "Integration Test:Finish"
exit 0
