#!/bin/bash

if [ ! -f deephealth.pid ]; then
  echo "No prior run found. Skip"
  exit 0
fi

dh_pid=$(cat deephealth.pid)

if [ -z "$dh_pid" ];then
  echo "Empty pid file. Skip"
  rm deephealth.pid
  exit 0
fi

kill -9 $dh_pid

if [ $? -eq 0 ]; then
  echo "Stopped deep health server (PID $dh_pid)"
  rm deephealth.pid
else
  echo "Failed to stop deep health server (PID $dh_pid)"
fi

pkill -9 hview-server 2>/dev/null
