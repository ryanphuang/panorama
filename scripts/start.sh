#!/bin/bash

if [ $# -ne 1 ];then
  echo "Usage: $0 CONFIG"
  exit 1
fi

config=$1

if [ ! -f $config ]; then
	echo "Could not find config file $config"
	exit 1
fi

if [ -f deephealth.pid ]; then
  echo "Deep health server process has already started. Stop it first."
  exit 0
fi

hview-server -config $config > deephealth.out 2>&1 &
dh_pid=$!
sleep 1
if ps -p$dh_pid > /dev/null; then
  echo $dh_pid > deephealth.pid
  echo "Deep health server started with PID $cdfs_pid"
else
  echo "Deep health server has exited"
fi
