!/bin/bash

source ../common.sh
source ./multi_region_config.sh 

create_rkv_ec2_instance(){
    local ami=$(find_ami $1) 
    local k_name=$(find_key_name $1)

    output=`aws ec2 run-instances --region $1 \
	    --placement  "AvailabilityZone=$2" \
	    --image-id $ami \
	    --security-group-ids $SECURITY_GROUP \
	    --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=$3}]" \
	    --instance-type $4\
	    --key-name $k_name  \
	    --block-device-mappings "DeviceName=/dev/sda1,Ebs={VolumeSize=${ROOT_DISK_VOLUME}}" \
	    `	# end block 

    instance_id=`jq '.Instances[0].InstanceId' <<< $output`
    instance_id=`sed -e 's/^"//' -e 's/"$//' <<<"$instance_id"`	# remove double quote from string $instance_id

    [[ -z "$instance_id" ]] && { echo "invalid instance_id " ; exit 1; }
    echo "just launched: ${instance_id} in region $1 az $2"
    
    state=""
    while [[ "$state" == "" ]]
    do
	    echo "waiting for 3 sec"
	    sleep 3 
	    state=`aws ec2 describe-instances --region $1 \
		    --instance-ids ${instance_id} \
		    --filters "Name=instance-state-name,Values=running" \
		    --output text`
    done
    host_public_ip=`aws ec2 describe-instances --region $1 \
			--instance-ids ${instance_id} \
			--query 'Reservations[].Instances[].PublicIpAddress' --output=text`
    echo "${instance_id} is running, public ip is ${host_public_ip}"
}

create_ec2_instance(){
    local ami=$(find_ami $1)
    local k_name=$(find_key_name $1)

    output=`aws ec2 run-instances --region $1 \
	    --placement  "AvailabilityZone=$2" \
	    --image-id $ami \
	    --security-group-ids $SECURITY_GROUP \
	    --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=$3}]" \
	    --instance-type $4\
	    --key-name $k_name  \
	    --block-device-mappings "DeviceName=/dev/sda1,Ebs={VolumeSize=${ROOT_DISK_VOLUME}}" \
	    `	# end block 

    instance_id=`jq '.Instances[0].InstanceId' <<< $output`
    instance_id=`sed -e 's/^"//' -e 's/"$//' <<<"$instance_id"`	# remove double quote from string $instance_id

    [[ -z "$instance_id" ]] && { echo "invalid instance_id " ; exit 1; }
    echo "just launched: ${instance_id} in region $1 az $2"
    
    state=""
    while [[ "$state" == "" ]]
    do
	    echo "waiting for 3 sec"
	    sleep 3 
	    state=`aws ec2 describe-instances --region $1 \
		    --instance-ids ${instance_id} \
		    --filters "Name=instance-state-name,Values=running" \
		    --output text`
    done
    host_public_ip=`aws ec2 describe-instances --region $1 \
			--instance-ids ${instance_id} \
			--query 'Reservations[].Instances[].PublicIpAddress' --output=text`
    echo "${instance_id} is running, public ip is ${host_public_ip}"
}

install_redis_fn() {
	sudo apt -y update >> /tmp/apt.log 2>&1
	sudo apt -y install redis-server >> /tmp/apt.log 2>&1
	sudo systemctl restart redis.service
}

configure_redis_fn() {
    sudo sed -i 's/^bind 127.0.0.1 ::1/#bind 127.0.0.1 ::1/' /etc/redis/redis.conf
    sudo sed -i 's/protected-mode yes/protected-mode no/' /etc/redis/redis.conf
    sudo sed -i 's/port 6379/port 6666/' /etc/redis/redis.conf
    sudo sed -i 's/^save \d*/# save /' /etc/redis/redis.conf
    sudo sed -i 's/^#\W*save ""/save ""/' /etc/redis/redis.conf

    sudo sudo systemctl restart redis
}

configure_redis() {
    for i in "${!ready_si_hosts[@]}"; do
        local r=${ready_si_regions[$i]}
	local k_file=$(find_key_file $r)
        local host_ip=${ready_si_hosts[$i]}

        echo "configuring redis on host $host_ip in region $r"
	ssh -i $k_file ubuntu@$host_ip "$(typeset -f configure_redis_fn); configure_redis_fn" &
    done
    wait
}

install_storage_binaries() {
    local k_file=$(find_key_file $1)
    local host_ip=$2
    echo "installing storage binaries on host $host_ip in region $1 using key $k_file"    
    until ssh -i $k_file -o "StrictHostKeyChecking no" ubuntu@$host_ip "$(typeset -f install_redis_fn); install_redis_fn"; do
        echo "ssh not ready, retry in 3 sec"    
        sleep 3
    done
}

validate_redis_up(){
    local k_file=$(find_key_file $1)
    local host_ip=$2
    resp=`ssh -i $k_file ubuntu@$host_ip "sudo redis-cli ping"`
    if [[ "$resp" == *"PONG"* ]]; then
        echo "redis is ready on host ${host_public_ip} in region $1"
    else
        echo "redis is NOT ready on host ${host_public_ip} in region $1"
    fi
}

provision_a_storage_instance() {
    # ${REGION_I} ${AZ_I} ${INSTANCE_TAG} ${INSTANCE_TYPE_I} ${PORT_I}
    # ${PORT_I} to be used later. hardcoded to 6666 at the time
    create_ec2_instance $1 $2 $3 $4	# this func assigns value $host_public_ip
    install_storage_binaries $1 $host_public_ip
    validate_redis_up $1 $host_public_ip
}

# create storage instances
provision_storage_instances() {
    source ./common_storage_instance.sh

    INSTANCE_IDX=0
    for i in "${!StoreRegions[@]}"; do 
        REGION_I=${StoreRegions[$i]}
        AZ_I=${StoreAvailabilityZones[$i]}
        COUNT_I=${StoreCounts[$i]}
        NAME_PREFIX_I=${StoreNamePrefixs[$i]}
        INSTANCE_TYPE_I=${StoreInstanceTypes[$i]}
        PORT_I=${StorePorts[$i]}
        LOG_NAME=si.log
        ANTI_THROTTLE_PAUSE=45
        ANTI_THROTTLE_GROUP_SIZE=12

        for (( j=1; j<=$COUNT_I; j++ ))
        do 
            if ! ((j % $ANTI_THROTTLE_GROUP_SIZE)); then
		echo sleeping $ANTI_THROTTLE_PAUSE sec to avoid throttling
	        sleep $ANTI_THROTTLE_PAUSE;
	    fi

            INSTANCE_TAG="${NAME_TAG}-${NAME_PREFIX_I}-${INSTANCE_IDX}"
            echo "ˁ˚ᴥ˚ˀ provisioning storage host ${INSTANCE_TAG} in region ${REGION_I} and AZ ${AZ_I}, see log ${LOG_NAME} for details"
	    provision_a_storage_instance ${REGION_I} ${AZ_I} ${INSTANCE_TAG} ${INSTANCE_TYPE_I} ${PORT_I} >> ${LOG_NAME} 2>&1 &
            ((INSTANCE_IDX+=1))
        done
    done
    wait
    
    INSTANCE_IDX=0
    for i in "${!StoreRegions[@]}"; do 
        REGION_I=${StoreRegions[$i]}
        AZ_I=${StoreAvailabilityZones[$i]}
        COUNT_I=${StoreCounts[$i]}
        NAME_PREFIX_I=${StoreNamePrefixs[$i]}
        INSTANCE_TYPE_I=${StoreInstanceTypes[$i]}

        for (( j=1; j<=$COUNT_I; j++ ))
        do 
            INSTANCE_TAG="${NAME_TAG}-${NAME_PREFIX_I}-${INSTANCE_IDX}"
            local host=`aws ec2 describe-instances --region ${REGION_I} --query 'Reservations[].Instances[].PublicIpAddress' \
	   					--filters "Name=tag-value,Values=${INSTANCE_TAG}" "Name=instance-state-name,Values=running" "Name=availability-zone,Values=${AZ_I}" \
	    					--output=text`
            ready_si_tags+=($INSTANCE_TAG)
            ready_si_hosts+=($host)
            ready_si_regions+=($REGION_I)
            ready_si_azs+=($AZ_I)
            
	    ((INSTANCE_IDX+=1))
        done
    done
	
    configure_redis
    
    print_green "the following storage instance(s) have been provisioned:" 

    for i in "${!ready_si_hosts[@]}"; do
        local r=${ready_si_regions[$i]}
        local z=${ready_si_azs[$i]}
        local h=${ready_si_hosts[$i]}
        print_light_green "$h in region $r az $z" 
    done
}

install_rkv_fn() {
    git clone https://github.com/CentaurusInfra/regionless-storage-service /home/ubuntu/regionless-storage-service >> /tmp/rkv.log 2>&1
    /home/ubuntu/regionless-storage-service/scripts/setup_env.sh >> /tmp/rkv.log 2>&1
    cd /home/ubuntu/regionless-storage-service
    source ~/.profile
    make 
}

setup_rkv_env() {
    host_ip=$1
    echo "setting up rkv env on $host_ip"
    ssh -i $KEY_FILE ubuntu@$host_ip "$(typeset -f install_rkv_fn); install_rkv_fn"
}

provision_a_rkv_instance() {
    repo_path=$REPO_ROOT
    # ${REGION_I} ${AZ} ${INSTANCE_TAG} ${INSTANCE_TYPE_I} ${PORT_I}
    create_rkv_ec2_instance ${RKV_REGION} ${RKV_AZ} ${INSTANCE_TAG} ${INSTANCE_TYPE}	# this func assigns value $host_public_ip
    
    until ssh -i $KEY_FILE -o "StrictHostKeyChecking no" ubuntu@$host_public_ip "sudo apt -y update >> /tmp/rkv.log 2>&1"; do
        echo "ssh not ready, retry in 3 sec"    
        sleep 3
    done
    setup_rkv_env $host_public_ip $repo_path
}

# create rkv instances
provision_rkv_instances() {
    source ./common_rkv_instance.sh

    local log_name=rkv.log
    echo "=^..^= provisioning rkv host, see log ${log_name} for details"
    provision_a_rkv_instance >${log_name} 2>&1
    
    hosts=`aws ec2 describe-instances --region ${RKV_REGION} --query 'Reservations[].Instances[].PublicIpAddress' \
	    				--filters "Name=tag-value,Values=${INSTANCE_TAG}" "Name=instance-state-name,Values=running" "Name=availability-zone,Values=${RKV_AZ}" \
    					--output=text`
    read -ra ready_rkv_hosts<<< "$hosts" # split by whitespaces

    print_green "the following rkv instance(s) have been provisioned:"
    for host in "${ready_rkv_hosts[@]}"
    do
        print_light_green "$host in region $RKV_REGION"
    done
}

setup_config() {
    size=${#ready_si_hosts[@]}
    config=$(jq -n --arg hashing "rendezvous" \
                  --argjson bucketsize 10 \
                  --arg storetype "redis" \
                  --argjson concurrent true \
                  --arg hashingmanagertype "syncAsync" \
                  --arg pipingtype "localSyncRemoteAsync" \
                  --argjson remotestorelatencythresholdinmillisec 50 \
                  --argjson localreplicanum 2 \
				  --argjson remotereplicanum 1 \
                  --argjson stores "[]" \
	          '{"ConsistentHash": $hashing, "BucketSize": $bucketsize, "LocalReplicaNum": $localreplicanum, "RemoteReplicaNum": $remotereplicanum, "StoreType": $storetype, "Concurrent": $concurrent, "HashingManagerType": $hashingmanagertype, "PipingType": $pipingtype, "RemoteStoreLatencyThresholdInMilliSec": $remotestorelatencythresholdinmillisec, "Stores": $stores}'
    )

    for i in "${!ready_si_hosts[@]}"; do
        local ip=${ready_si_hosts[$i]}
        local region=${ready_si_regions[$i]}
        local az=${ready_si_azs[$i]}
        local t=${ready_si_tags[$i]}
        inner=$(jq -n --arg name $t \
    	    --arg host $ip \
            --arg regionname $region \
            --arg azname $az \
          --argjson port 6666 \
            '{"Region": $regionname, "AvailabilityZone": $azname, "Name": $name, "Host": $host, "Port": $port}'
        )
        config="$(jq --argjson val "$inner" '.Stores += [$val]' <<< "$config")"
    done

    config_file_name=generated_config.json
    echo $config > $config_file_name 

    print_green "rkv service config file created:"
    jq . generated_config.json 
    
    for host in "${ready_rkv_hosts[@]}"
    do
        print_green "copying config to rkv instance $host:/tmp/config.json. Note the file name change here!"
	scp -i $KEY_FILE generated_config.json ubuntu@$host_ip:/tmp/config.json
    done
}

SCRIPTPATH="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
export REPO_ROOT=$( cd ${SCRIPTPATH}/../.. && pwd -P )

read_stores # read store configs

read_region_configs # read ssh key configs

provision_storage_instances

provision_rkv_instances
    
setup_config
