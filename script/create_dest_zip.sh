#!/bin/bash
cd $(dirname $0)
cd ../

if [ -z "$1" ];then
    bash build.sh
    bash build.sh windows
fi

version=$(cat res/version)

cd dest
################################################

if [ -d conf ];then
  rm -rf conf
fi

mkdir conf
cp ../res/conf/demo.conf conf/pproxy.conf
echo -e "#name psw isAdmin\nadmin psw admin">conf/users
cp ../conf/req_rewrite_8080.js conf/
echo -e "news.baidu.com 127.0.0.1\nnews.163.com 127.0.0.1:81">conf/hosts_8080

t=$(date +"%Y%m%d%H")
################################################
target_linux="pproxy_${version}_linux_$t.tar.gz"
if [ -f $target_linux ];then
   rm $target_linux
fi


mkdir -p linux/data

cp pproxy ../script/pproxy_control.sh linux/
cp -r conf linux/conf


dir_new="pproxy_${version}"
mv linux $dir_new
tar -czvf $target_linux $dir_new

rm -rf  $dir_new


################################################
target_windows="pproxy_${version}_windows_$t.zip"
if [ -f $target_windows ];then
   rm $target_windows
fi

mkdir -p windows/data

cp pproxy.exe windows
cp ../script/windows_run.bat windows/start.bat 
cp -r conf windows/conf


mv windows $dir_new
zip -r $target_windows $dir_new

rm -rf  $dir_new conf



