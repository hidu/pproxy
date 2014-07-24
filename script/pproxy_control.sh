#!/bin/bash

cd $(dirname $0)

conf_file=$2

if [ -z "$conf_file" ];then
    conf_file="./conf/pproxy.conf"
fi

conf_path=$(readlink -f $conf_file)

if [ ! -f "$conf_path" ];then
   echo "conf file[${conf_path}] not exists!"
   exit 2
fi

bin_path=$(readlink -f ./pproxy)

run_cmd="$bin_path -conf $conf_path"

function start(){
    nohup $run_cmd>/dev/null 2>&1 &  
    status=$?
   if [ "$status" == "0" ];then
        echo "start suc! pid="$!
    else
       echo "start failed!"
       exit 2
    fi
}

function stop(){
    list=$(ps aux|grep "$run_cmd"|grep -v grep)
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