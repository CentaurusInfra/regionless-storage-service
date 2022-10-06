#!/bin/bash

num_client=5
duration=20 #seconds
key="test"
consistency="linearizable"

echo "Cleaning old log files..."
rm ./*.log

if [ $# -eq 1 ]; then
    consistency=$1
fi

echo "Check the $consistency consistency ..."

for (( client_id=1; client_id<=$num_client; client_id++ ))
do
    echo "Start running client $client_id with the key $key ..."
    ./consistency_log $client_id $duration $key $consistency &
done

sleep $duration
echo "Complete."
