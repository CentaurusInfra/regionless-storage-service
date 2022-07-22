#!/bin/bash

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
	sudo apt -y update >> /tmp/apt.log 2>&1
	sudo apt -y install redis-server >> /tmp/apt.log 2>&1
	sudo systemctl restart redis.service
}

configure_redis_fn() {
    sudo sed -i 's/^bind 127.0.0.1 ::1/#bind 127.0.0.1 ::1/' /etc/redis/redis.conf
    sudo sudo systemctl restart redis
}

configure_redis() {
    echo ${ready_si_hosts}
    for host_ip in "${ready_si_hosts[@]}"
    do
        echo "configuring redis on host $host_ip"
	ssh -i regionless_kv_service_key.pem ubuntu@$host_ip "$(typeset -f configure_redis_fn); configure_redis_fn" &
    done
    wait
}

install_storage_binaries() {
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
	      ready_si_hosts+=$host_public_ip
    fi
}

provision_a_storage_instance() {
    create_ec2_instance	# this func assigns $host_public_ip
    install_storage_binaries $host_public_ip
    validate_redis_up
}

# create storage instances
provision_storage_instances() {
    source ./common_storage_instance.sh

    for i in $( eval echo {1..$NUM_OF_INSTANCE} ) 
    do
       log_name=$i.log
       echo "ˁ˚ᴥ˚ˀ provisioning storage host ${i}, see log ${log_name} for details"
       provision_a_storage_instance > ${log_name} 2>&1 & 
    done
    wait

    hosts=`aws ec2 describe-instances --query 'Reservations[].Instances[].PublicIpAddress' \
    					--filters "Name=tag-value,Values=${INSTANCE_TAG}" "Name=instance-state-name,Values=running" \
    					--output=text`
    read -ra ready_si_hosts<<< "$hosts" # split by whitespaces
    
    configure_redis	# $ready_si_hosts is created just above 

    echo "the following storage instance(s) have been provisioned:"
    for host in "${ready_si_hosts[@]}"
    do
        echo "$host"
    done
}

install_rkv_fn() {
    sudo /home/ubuntu/regionless-storage-service/scripts/setup_env.sh >> /tmp/rkv.log 2>&1
    cd /home/ubuntu/regionless-storage-service
    source ~/.profile
    make 
}

setup_rkv_env() {
    host_ip=$1
    echo ">>>> copying repo to $host_ip"    
    scp -r -i regionless_kv_service_key.pem -o "StrictHostKeyChecking no" $2 ubuntu@$host_ip:~

    echo ">>>> setup rkv env on $host_ip"    
    ssh -i regionless_kv_service_key.pem ubuntu@$host_ip "$(typeset -f install_rkv_fn); install_rkv_fn"
}

provision_a_rkv_instance() {
    repo_path=/root/go/src/github.com/regionless-storage-service
    create_ec2_instance # this func assigns $host_public_ip
    
    until ssh -i regionless_kv_service_key.pem -o "StrictHostKeyChecking no" ubuntu@$host_public_ip "sudo apt -y update >> /tmp/rkv.log 2>&1"; do
        echo ">>>> ssh not ready, retry in 3 sec"    
        sleep 3
    done
    setup_rkv_env $host_public_ip $repo_path
}

# create rkv instances
provision_rkv_instances() {
    source ./common_rkv_instance.sh

    log_name=rkv.log
    echo "=^..^= provisioning rkv host, see log ${log_name} for details"
    provision_a_rkv_instance >${log_name} 2>&1
    
    hosts=`aws ec2 describe-instances --query 'Reservations[].Instances[].PublicIpAddress' \
    					--filters "Name=tag-value,Values=${INSTANCE_TAG}" "Name=instance-state-name,Values=running" \
    					--output=text`
    read -ra ready_rkv_hosts<<< "$hosts" # split by whitespaces

    echo "the following rkv instance(s) have been provisioned:"
    for host in "${ready_rkv_hosts[@]}"
    do
        echo "$host"
    done
}

setup_config() {
    size=${#ready_si_hosts[@]}
    config=$(jq -n --arg hashing "rendezvous" \
                  --argjson bucketsize 10 \
                  --arg storetype "reids" \
                  --argjson replicanum 2 \
                  --argjson stores "[]" \
	          '{"ConsistentHash": $hashing, "BucketSize": $bucketsize, "ReplicaNum": $replicanum, "StoreType": $storetype, "Stores": $stores}'
    )

    for ip in "${ready_si_hosts[@]}"
    do
        inner=$(jq -n --arg name "si-$ip" \
    	    --arg host $ip \
    	    --argjson port 6379 \
    	    '{"Name": $name, "Host": $host, "Port": $port}'
        )
        config="$(jq --argjson val "$inner" '.Stores += [$val]' <<< "$config")"
    done

    config_file_name=generated_config.json
    echo $config > $config_file_name 

    echo "config file created:"
    jq . generated_config.json 
    
    for host in "${ready_rkv_hosts[@]}"
    do
        echo "copying config.json to rkv instance $host:/tmp/config.json. !!Note the file name change here!!"
	scp -i regionless_kv_service_key.pem generated_config.json ubuntu@$host_ip:/tmp/config.json
    done
}

provision_storage_instances

provision_rkv_instances
    
setup_config
