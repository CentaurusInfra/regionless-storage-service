#!/bin/bash

AMI=ami-0b152cfd354c4c7a4
SECURITY_GROUP=regionless_kv_service
INSTANCE_TYPE=t2.micro
KEY_NAME=regionless_kv_service_key
INSTANCE_TAG=rkv_perf_test_pengdu

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
    echo ">>>> ${instance_id} is running"
}

create_ec2_instance
