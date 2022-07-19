#!/bin/bash

source rkv_common.sh

running_instances=`aws ec2 describe-instances 	--query 'Reservations[].Instances[].InstanceId' \
						--filters "Name=tag-value,Values=$INSTANCE_TAG" "Name=instance-state-name,Values=running" \
						--output text`

read -ra instances <<<"$running_instances"

for element in "${instances[@]}"
do
    echo "$element"
done
