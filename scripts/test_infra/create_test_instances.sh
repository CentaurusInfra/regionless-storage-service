#!/bin/bash

source rkv_common.sh

create_ec2_instance(){
    output=`aws ec2 run-instances --image-id $AMI \
	    --security-group-ids $SECURITY_GROUP \
	    --instance-type $INSTANCE_TYPE \
	    --key-name $KEY_NAME  \
	    --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=$INSTANCE_TAG}]" \
	    #				--block-device-mappings 'DeviceName=/dev/sda1,Ebs={VolumeSize=32}' \
	    `	# end block 

    instance_id=`jq '.Instances[0].InstanceId' <<< $output`
    instance_id=`sed -e 's/^"//' -e 's/"$//' <<<"$instance_id"`	# remove double quote from string $instance_id
    #launched_instance_ids+=($instance_id)
    #echo "just launched: ${launched_instance_ids[@]}"

    [[ -z "$instance_id" ]] && { echo "invalid instance_id " ; exit 1; }
    echo ">>>> just launched: ${instance_id}"
    
    state=""
    while [[ "$state" == "" ]]
    do
	    echo ">>>> waiting for 3 sec"
	    sleep 3 
	    state=`aws ec2 describe-instances \
		    --instance-ids $instance_id \
		    --filters "Name=instance-state-name,Values=running" \
		    --output text`
    done
    host_public_ip=`aws ec2 describe-instances --instance-ids ${instance_id} --query 'Reservations[].Instances[].PublicIpAddress' --output=text`
    echo ">>>> ${instance_id} is running, public ip is ${host_public_ip}"
}

install_redis_fn() {
	sudo apt -y update > /tmp/apt.log 2>&1
	sudo apt -y install redis-server > /tmp/apt.log 2>&1
	sudo systemctl restart redis.service
}

prepare_host() {
    host_ip=$1
    echo ">>>> preparing host $host_ip"    
    until ssh -i regionless_kv_service_key.pem -o "StrictHostKeyChecking no" ubuntu@$host_ip "$(typeset -f install_redis_fn); install_redis_fn"; do
        echo ">>>> ssh not ready, retry in 3 sec"    
        sleep 3
    done
}

validate_redis_up(){
    resp=`ssh -i regionless_kv_service_key.pem ubuntu@$host_ip "sudo redis-cli ping"`
    if [[ "$resp" == *"PONG"* ]]; then
	      echo "Redis is ready on host ${host_public_ip}"
    fi
}

create_ec2_instance

prepare_host $host_public_ip
    
validate_redis_up
