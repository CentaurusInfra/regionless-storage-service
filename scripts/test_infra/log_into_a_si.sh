#!/bin/bash

idx=0
if [ $# -eq 4 ]
then
    idx="$1"
    prefix="$2"
    region="$3"
    zone="$4"
else
    echo "need arguments: [si index] [ec2 vm tag prefix] [US region, e.g. west] [US zone, e.g. 2]"
    exit 1
fi

si_ip=`aws ec2 describe-instances --region us-$region-$zone --filters "Name=tag-value,Values=$prefix-rkv-lab-si-$idx" "Name=instance-state-name,Values=running" --query 'Reservations[].Instances[].PublicIpAddress' --output=text`
ssh -i regionless_kv_service_key_us_$region\_$zone.pem ubuntu@$si_ip
