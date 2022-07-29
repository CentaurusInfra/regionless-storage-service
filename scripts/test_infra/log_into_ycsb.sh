#!/bin/bash

if [ $# -eq 1 ]
then
    prefix="$1"
else
    echo "need one arguments: [ec2 vm tag prefix]"
    exit 1
fi

ycsb_ip=`aws ec2 describe-instances --region us-west-2 --filters "Name=tag-value,Values=$prefix-rkv-lab-ycsb" "Name=instance-state-name,Values=running" --query 'Reservations[].Instances[].PublicIpAddress' --output=text`
ssh -i regionless_kv_service_key_us_west_2.pem ubuntu@$ycsb_ip
