#!/bin/bash

DATA=` ps aux | grep scalarm_load_balancer | awk '{if($11 ~ /scalarm_load_balancer/) print $1" "$2}'`
PID=`echo $DATA | awk '{print $2}'`
USER=`echo $DATA | awk '{print $1}'`

if [ $USER = "root" ]; then
	echo "kill $PID"  | sudo sh
else
	kill $PID
fi
