#!/bin/bash

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd $DIR

PORT=`cat ../config/config.json | grep Port | awk -F'\"' '{ print $4 }'`

if [ -z "$PORT" ]; then
	PORT=443
fi

if [ $PORT = "443" ]; then
	echo "nohup $GOPATH/bin/scalarm_load_balancer ../config/config.json >/dev/null 2>&1 &" | sudo sh
else
	nohup $GOPATH/bin/scalarm_load_balancer ../config/config.json >/dev/null 2>&1 &
fi
