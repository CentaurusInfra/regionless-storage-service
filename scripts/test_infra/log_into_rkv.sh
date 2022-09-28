#!/bin/bash

if [ $# -eq 3 ]
then
    prefix="$1"
    region="$2"
    zone="$3"
else
    echo "need arguments: [ec2 vm tag prefix] [US region, e.g. west] [US zone, e.g. 2]"
    echo "example: ./log_into_rkv.sh pengd west 2"
    exit 1
fi

rkv_ip=`aws ec2 describe-instances --region us-$region-$zone --filters "Name=tag-value,Values=$prefix-rkv-lab-rkv" "Name=instance-state-name,Values=running" --query 'Reservations[].Instances[].PublicIpAddress' --output=text`
ssh -i regionless_kv_service_key_us_$region\_$zone.pem ubuntu@$rkv_ip
