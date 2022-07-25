#!/usr/bin/env bash
set -euo pipefail

source common.sh 

print_usage() {
    echo "Usage:"
    echo "  cd scripts"
    echo "  export KEY_NAME=<your ec2 key name>"
    echo "  export KEY_FILE=<path-to-ec2-key-file>"
    echo "  export NAME_TAG=<name_tag to identify your resources>"
    echo "  and then run ./setup-test-lab.sh"
}

if [ -z ${NAME_TAG:=} ] || [ -z ${KEY_NAME:=} ] || [ -z ${KEY_FILE:=} ]
then
    echo "=^..^= One of more env variable need to be defined"
    print_usage
    echo "exited."
      exit 1
fi

## this is for AWS env only
## to set up singular region test lab of rkv perf

## get the default values
. ./test-lab.val
    
#
# start redis vm instances and rkv server
#
# we will make a few changes to rkv config and start its service later
cd test_infra && ./create_test_instances.sh

#
## start jaeger server
#
jeager_vmid=$(aws ec2 run-instances \
  --image-id ${JAEGER_AMI} \
  --security-groups ${SECURITY_GROUP} \
  --instance-type ${JAEGER_INSTANCE_TYPE} \
  --key-name ${KEY_NAME} \
  --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=${JAEGER_VM_NAME}}]" \
  --block-device-mappings "DeviceName=/dev/sda1,Ebs={VolumeSize=${JAEGER_ROOT_DISK_VOLUME}}" \
  --output text \
  --query 'Instances[*].InstanceId')
aws ec2 wait instance-status-ok --instance-ids ${jeager_vmid}
jaeger_vmip=$(aws ec2 describe-instances \
  --instance-ids ${jeager_vmid} \
  --query "Reservations[].Instances[].NetworkInterfaces[].PrivateIpAddresses[].Association.PublicIp" \
  --output text)
print_green "jaeger server provisioned, ip addr is ${jaeger_vmip}"

#
## identify rkv service
#
rkv_vmid=$(aws ec2 describe-instances --filters "Name=tag:Name, Values=${RKV_VM_NAME}" "Name=instance-state-name,Values=running" --output text --query 'Reservations[*].Instances[*].InstanceId')
aws ec2 wait instance-status-ok --instance-ids ${rkv_vmid}
rkv_vmip=$(aws ec2 describe-instances \
  --instance-ids ${rkv_vmid} \
  --query "Reservations[].Instances[].NetworkInterfaces[].PrivateIpAddresses[].Association.PublicIp" \
  --output text)
echo "rkv service ip addr is ${rkv_vmip}"

#
## launch rkv service with proper jaeger endpoint
#
ssh -i ${KEY_FILE} ubuntu@${rkv_vmip} -o "StrictHostKeyChecking no" <<ENDS
cp /tmp/config.json ~/regionless-storage-service/cmd/http/config.json
nohup ~/regionless-storage-service/main --jaeger-server=http://${jaeger_vmip}:14268 >/tmp/rkv.log 2>&1 &
ENDS
print_green "rkv service launched on ${rkv_vmip}"

#
## todo: start prometheus server
#

#
## now, it is ok to run go-ycsb against rkv service
#
ycsb_vmid=$(aws ec2 run-instances \
  --image-id ${YCSB_AMI} \
  --security-groups ${SECURITY_GROUP} \
  --instance-type ${YCSB_INSTANCE_TYPE} \
  --key-name ${KEY_NAME} \
  --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=${YCSB_VM_NAME}}]" \
  --block-device-mappings "DeviceName=/dev/sda1,Ebs={VolumeSize=${YCSB_ROOT_DISK_VOLUME}}" \
  --output text \
  --query 'Instances[*].InstanceId')
aws ec2 wait instance-status-ok --instance-ids ${ycsb_vmid}

ycsb_vmip=$(aws ec2 describe-instances \
  --instance-ids ${ycsb_vmid} \
  --query "Reservations[].Instances[].NetworkInterfaces[].PrivateIpAddresses[].Association.PublicIp" \
  --output text)
print_green "ycsb client ip addr is ${rkv_vmip}"
echo "rkv endpoint is at ${rkv_vmip}:8090"

# set rkv ip addr properly for go-ycsb to test against
ssh -i ${KEY_FILE} ubuntu@${ycsb_vmip} -o "StrictHostKeyChecking no" <<ENDS
sudo sed -i '/rkv/d' /etc/hosts
echo ${rkv_vmip} rkv | sudo tee -a /etc/hosts > /dev/null
ENDS

print_green "tests can be fired up against rkv service now  d(^o^)b"
#
## run workloada for now; saving output to /tmp/ycsb-a.log
## todo: run more workloads
#
echo "running a test, log in /tmp/ycsb-a.log"
ssh -i ${KEY_FILE} ubuntu@${ycsb_vmip} -o "StrictHostKeyChecking no" "cd work/go-ycsb && ./bin/go-ycsb load rkv -P workloads/workloada" > /tmp/ycsb-a.log 2>&1

print_green "jaeger tracing at http://${jaeger_vmip}:16686"
