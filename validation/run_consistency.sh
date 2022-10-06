#!/bin/bash

num_client=5
duration=20 #seconds
consistency="linearizable"

echo "Cleaning old log files..."
rm ./*.log


if [ $# -gt 0 ]; then
    consistent_level=$1
fi

echo "Check the $consistency consistency ..."

echo "Generate a random key..."
key=$(tr -dc A-Za-z </dev/urandom | head -c 10 ; echo '')
echo "The random key is generated as $key"

for (( client_id=1; client_id<=$num_client; client_id++ ))
do
    echo "Start running client $client_id with the random key $key ..."
    ./consistency_log $client_id $duration $key consistency &
done

#sleep $duration
echo "Complete."