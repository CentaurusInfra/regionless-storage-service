#!/bin/bash

idx=0
if [ $# -eq 2 ]
then
    idx="$1"
    prefix="$2"
else
    echo "not enough argument given. required: [si index] [name prefix]"
    exit 1
fi

si_ip=`aws ec2 describe-instances --region us-west-2 --filters "Name=tag-value,Values=$prefix-rkv-lab-si-$idx" "Name=instance-state-name,Values=running" --query 'Reservations[].Instances[].PublicIpAddress' --output=text`
redis-cli -h $si_ip -p 6666
