#/bin/bash
cd $(dirname $0)
cd serve
#gofmt -tabs=false -w=true -tabwidth=4 .
gofmt -w=true -s=true .

