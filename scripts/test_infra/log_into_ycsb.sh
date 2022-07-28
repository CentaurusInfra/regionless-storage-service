#!/bin/bash

ycsb_ip=`aws ec2 describe-instances --region us-west-2 --filters "Name=tag-value,Values=pengdu-rkv-lab-ycsb" "Name=instance-state-name,Values=running" --query 'Reservations[].Instances[].PublicIpAddress' --output=text`
ssh -i regionless_kv_service_key_us_west_2.pem ubuntu@$ycsb_ip
