#!/bin/bash

source rkv_common.sh

create_ec2_instance(){
    output=`aws ec2 run-instances --image-id $AMI \
	    --security-group-ids $SECURITY_GROUP \
	    --instance-type $INSTANCE_TYPE \
	    --key-name $KEY_NAME  \
	    --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=$INSTANCE_TAG}]" \
	    				--block-device-mappings "DeviceName=/dev/sda1,Ebs={VolumeSize=${ROOT_DISK_VOLUME}}" \
	    `	# end block 

    instance_id=`jq '.Instances[0].InstanceId' <<< $output`
    instance_id=`sed -e 's/^"//' -e 's/"$//' <<<"$instance_id"`	# remove double quote from string $instance_id

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

install_stuff() {
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
	      ready_hosts+=$host_public_ip
    fi
}

provision_host() {
    create_ec2_instance
    install_stuff $host_public_ip
    validate_redis_up
}

for i in {1..2}
do
   log_name=$i.log
   echo "provisioning redis host ${i}, see log ${log_name} for details"
   provision_host > ${log_name} 2>&1 & 
done
wait

hosts=`aws ec2 describe-instances --query 'Reservations[].Instances[].PublicIpAddress' \
					--filters "Name=tag-value,Values=${INSTANCE_TAG}" "Name=instance-state-name,Values=running" \
					--output=text`
read -ra ready_hosts<<< "$hosts" # split by whitespaces

echo "the following host(s) have been provisioned:"
for host in "${ready_hosts[@]}"
do
    echo "$host"
done
