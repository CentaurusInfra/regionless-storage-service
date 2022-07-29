#!/bin/bash

idx=0
if [ $# -eq 0 ]
then
    echo "no argument given. using default storage instance #$idx"
else
    idx="$1"
fi

si_ip=`aws ec2 describe-instances --region us-west-2 --filters "Name=tag-value,Values=pengdu-rkv-lab-si-$idx" "Name=instance-state-name,Values=running" --query 'Reservations[].Instances[].PublicIpAddress' --output=text`
redis-cli -h $si_ip -p 6666
