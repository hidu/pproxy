#/bin/bash
cd $(dirname $0)
gofmt -tabs=false -w=true -tabwidth=4 .

