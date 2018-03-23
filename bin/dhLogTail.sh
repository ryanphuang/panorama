#!/bin/bash

if [ $# -ne 1 ];then
  echo "Usage: $0 start|stop"
  exit 1
fi

if [ "$1" == "start" ]; then
  hview-logtail -stale=-1 -log ~/software/zookeeper/zookeeper.out zookeeper --ensemble ~/software/zookeeper/conf/zoo.cfg  --myid /mnt/fuser-ryan/zookeeper/myid --filter conf/zoo_filter.json > logtail.log 2>logtail.err &
elif [ "$1" == "stop" ]; then
  pkill -9 hview-logtail
else
  echo "Must be start or stop"
  exit 1
fi

