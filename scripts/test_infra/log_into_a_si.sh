#!/bin/bash

idx=0
if [ $# -eq 2 ]
then
    idx="$1"
    prefix="$2"
else
    echo "need two arguments: [si index] [ec2 vm tag prefix]"
    exit 1
fi

si_ip=`aws ec2 describe-instances --region us-west-2 --filters "Name=tag-value,Values=$prefix-rkv-lab-si-$idx" "Name=instance-state-name,Values=running" --query 'Reservations[].Instances[].PublicIpAddress' --output=text`
ssh -i regionless_kv_service_key_us_west_2.pem ubuntu@$si_ip
