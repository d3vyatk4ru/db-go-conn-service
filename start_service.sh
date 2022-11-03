#!/bin/bash

function ProgressBar {

    let _progress=(${1}*100/${2}*100)/100
    let _done=(${_progress}*4)/10
    let _left=40-$_done

    _fill=$(printf "%${_done}s")
    _empty=$(printf "%${_left}s")

printf "\rProgress : [${_fill// /#}${_empty// /-}] ${_progress}%%"

}

echo "Start docker container"

docker-compose up -d

echo "Waiting 150 seconds for container setting"

_start=1

_end=100

for number in $(seq ${_start} ${_end})
do
    sleep 1.5
    ProgressBar ${number} ${_end}
done

printf "\nTime is over!!!\n"

go test -v