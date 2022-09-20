#!/bin/bash

num_client=5
duration=20 #seconds
key=test

echo "Cleaning old log files..."
rm ./*.log

for (( client_id=1; client_id<=$num_client; client_id++ ))
do
    echo "Start running client $client_id..."
    ./consistency_log $client_id $duration $key &
done

sleep $duration
echo "Complete."