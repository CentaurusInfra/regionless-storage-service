#!/bin/bash

idx=0
if [ $# -eq 0 ]
then
    echo "no argument given. using default storage instance #$idx"
else
    idx="$1"
fi

si_ip=`aws ec2 describe-instances --region us-west-2 --filters "Name=tag-value,Values=pengdu-rkv-lab-si-$idx" "Name=instance-state-name,Values=running" --query 'Reservations[].Instances[].PublicIpAddress' --output=text`
ssh -i regionless_kv_service_key_us_west_2.pem ubuntu@$si_ip
