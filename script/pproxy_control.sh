#!/bin/bash

CUR_DIR=$(dirname $0)

BIN_NAME="./pproxy"
DEFAULT_CONF="./conf/pproxy.conf"
INTRO="get more info from github.com/hidu/pproxy"


CONF_FILE=$2

if [ -z "$CONF_FILE" ];then
    cd $CUR_DIR
    CONF_FILE="$DEFAULT_CONF"
fi

CONF_PATH=$(readlink -f "$CONF_FILE")

cd $CUR_DIR

BIN_PATH=$(readlink -f $BIN_NAME)

if [ ! -f "$CONF_PATH" ];then
   echo "conf file[${CONF_PATH}] not exists!"
   exit 2
fi


RUN_CMD="$BIN_PATH -conf $CONF_PATH"

function start(){
    nohup $RUN_CMD>/dev/null 2>&1 &  
    status=$?
   if [ "$status" == "0" ];then
        echo "start suc! pid="$!
    else
       echo "start failed!"
       exit 2
    fi
}

function stop(){
    list=$(ps aux|grep "$RUN_CMD"|grep -v grep)
    if [ -z "${list}" ];then
       echo "no process to kill"
    else
       pid=$( echo "$list"|awk '{print $2}')
       kill $pid
       if [ "$?"=="0" ];then
           echo "stop suc! pid=${pid}"
       else
          echo "stop failed! pid=${pid}"
          exit 3
       fi
    fi
}

function restart(){
   stop
   start
}

function useage(){
   echo "pproxy useage:"
   echo $0 "start|stop|restart" [conf_path]
   echo  -e "$INTRO"
}

if [ $# -lt 1 ]; then
    useage
    exit 1
fi

case "$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
esac