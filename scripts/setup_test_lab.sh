#!/usr/bin/env bash
set -euo pipefail

launch_rkv_fn() {
    jaeger_vmip=$1
    cp /tmp/config.json ~/regionless-storage-service/cmd/http/config.json
    nohup ~/regionless-storage-service/main --jaeger-server=http://${jaeger_vmip}:14268 >/tmp/rkv.log 2>&1 &
}

config_ycsb_fn() {
    rkv_vmip=$1
    sudo sed -i '/rkv/d' /etc/hosts
    echo ${rkv_vmip} rkv | sudo tee -a /etc/hosts > /dev/null
}

create_jaeger_vm() {
    jaeger_vmid=$(aws ec2 run-instances \
      --region ${JAEGER_REGION} \
      --placement  "AvailabilityZone=${JAEGER_AZ}" \
      --image-id ${JAEGER_AMI} \
      --security-groups ${SECURITY_GROUP} \
      --instance-type ${JAEGER_INSTANCE_TYPE} \
      --key-name ${KEY_NAME} \
      --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=${JAEGER_VM_NAME}}]" \
      --block-device-mappings "DeviceName=/dev/sda1,Ebs={VolumeSize=${JAEGER_ROOT_DISK_VOLUME}}" \
      --output text \
      --query 'Instances[*].InstanceId')
    aws ec2 wait instance-status-ok --instance-ids ${jaeger_vmid}
}

create_ycsb_vm() {
    ycsb_vmid=$(aws ec2 run-instances \
      --region ${YCSB_REGION} \
      --placement  "AvailabilityZone=${YCSB_AZ}" \
      --image-id ${YCSB_AMI} \
      --security-groups ${SECURITY_GROUP} \
      --instance-type ${YCSB_INSTANCE_TYPE} \
      --key-name ${KEY_NAME} \
      --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=${YCSB_VM_NAME}}]" \
      --block-device-mappings "DeviceName=/dev/sda1,Ebs={VolumeSize=${YCSB_ROOT_DISK_VOLUME}}" \
      --output text \
      --query 'Instances[*].InstanceId')
    aws ec2 wait instance-status-ok --instance-ids ${ycsb_vmid}
}

source common.sh 

print_usage() {
    echo "Usage:"
    echo "  cd scripts"
    echo "  export KEY_NAME=<your ec2 key name>"
    echo "  export KEY_FILE=<path-to-ec2-key-file>"
    echo "  export NAME_TAG=<name_tag to identify your resources>"
    echo "  and then run ./setup-test-lab.sh [optional: si definition file name, yes, name without path, 'scripts/si_def.json' if not provided"
}

if [ -z ${NAME_TAG:=} ] || [ -z ${KEY_NAME:=} ] || [ -z ${KEY_FILE:=} ]
then
    echo "=^..^= One or more env variable need to be defined"
    print_usage
    echo "exited."
      exit 1
fi

cat splash.art 

## this is for AWS env only
## to set up singular region test lab of rkv perf

## get the default values
. ./test-lab.val

if [ $# -eq 0 ]
then
    echo "no argument given. using default storage config file $SI_DEF_FILE"
else
    SI_DEF_FILE="../$1"
fi

echo "using $SI_DEF_FILE as the storage instance definition"

#
# start redis vm instances and rkv server
#
# we will make a few changes to rkv config and start its service later
cd test_infra && ./create_test_instances.sh

#
## start jaeger & ycsb vm 
## todo: start prometheus server
#
echo "creating jaeger vm and ycsb vm"
jaeger_vmid=""
ycsb_vmid=""
create_jaeger_vm &
create_ycsb_vm &
wait

jaeger_vmip=$(aws ec2 describe-instances \
  --filters "Name=tag-value,Values=${JAEGER_VM_NAME}" "Name=instance-state-name,Values=running" \
  --query "Reservations[].Instances[].NetworkInterfaces[].PrivateIpAddresses[].Association.PublicIp" \
  --output text)
print_green "jaeger vm provisioned, ip addr is ${jaeger_vmip}"

ycsb_vmip=$(aws ec2 describe-instances \
  --instance-ids ${ycsb_vmid} \
  --filters "Name=tag-value,Values=${YCSB_VM_NAME}" "Name=instance-state-name,Values=running" \
  --query "Reservations[].Instances[].NetworkInterfaces[].PrivateIpAddresses[].Association.PublicIp" \
  --output text)
print_green "ycsb vm provisioned, ip addr is ${ycsb_vmip}"

#
## identify rkv service
#
rkv_vmid=$(aws ec2 describe-instances --filters "Name=tag:Name, Values=${RKV_VM_NAME}" "Name=instance-state-name,Values=running" --output text --query 'Reservations[*].Instances[*].InstanceId')
aws ec2 wait instance-status-ok --instance-ids ${rkv_vmid}
rkv_vmip=$(aws ec2 describe-instances \
  --filters "Name=tag-value,Values=${RKV_VM_NAME}" "Name=instance-state-name,Values=running" \
  --query "Reservations[].Instances[].NetworkInterfaces[].PrivateIpAddresses[].Association.PublicIp" \
  --output text)
echo "rkv service ip addr is ${rkv_vmip}"

#
## launch rkv service with proper jaeger endpoint
#
ssh -i ${KEY_FILE} ubuntu@${rkv_vmip} -o "StrictHostKeyChecking no" "$(typeset -f launch_rkv_fn); launch_rkv_fn $jaeger_vmip" >>rkv.log 2>&1
print_green "rkv service launched on ${rkv_vmip}"

#
# set rkv ip addr properly for go-ycsb to test against
#
ssh -i ${KEY_FILE} ubuntu@${ycsb_vmip} -o "StrictHostKeyChecking no" "$(typeset -f config_ycsb_fn); config_ycsb_fn $rkv_vmip" >>rkv.log 2>&1
print_green "\nRKV TEST LAB READY d(^o^)b"
print_green "tests can be fired up against rkv service now"

#
## run workloada for now; saving output to /tmp/ycsb-a.log
## todo: run more workloads
#
echo "firing a test, log in /tmp/ycsb-a.log"
ssh -i ${KEY_FILE} ubuntu@${ycsb_vmip} -o "StrictHostKeyChecking no" "cd work/go-ycsb && ./bin/go-ycsb load rkv -P workloads/workloada" > /tmp/ycsb-load-a.log 2>&1
ssh -i ${KEY_FILE} ubuntu@${ycsb_vmip} -o "StrictHostKeyChecking no" "cd work/go-ycsb && ./bin/go-ycsb run rkv -P workloads/workloada" > /tmp/ycsb-run-a.log 2>&1

print_green "jaeger tracing at http://${jaeger_vmip}:16686"
