#!/usr/bin/env bash
set -euo pipefail

## this is for AWS env only
## to set up singular region test lab of rkv perf

## get the default values
. ./test-lab.val

## start redis vm instances and rkv server
## we will make a few changes to rkv config and start its service later
cd test_infra && ./create_test_instances.sh

## start jaeger server
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
echo "jaeger server ip addr is ${jaeger_vmip}"

## identify rkv service
rkv_vmid=$(aws ec2 describe-instances --filters "Name=tag:Name, Values=${RKV_VM_NAME}" "Name=instance-state-name,Values=running" --output text --query 'Reservations[*].Instances[*].InstanceId')
aws ec2 wait instance-status-ok --instance-ids ${rkv_vmid}
rkv_vmip=$(aws ec2 describe-instances \
  --instance-ids ${rkv_vmid} \
  --query "Reservations[].Instances[].NetworkInterfaces[].PrivateIpAddresses[].Association.PublicIp" \
  --output text)
echo "rkv service ip addr is ${rkv_vmip}"
## launch rkv service with proper jaeger endpoint
ssh -i ${KEY_FILE} ubuntu@${rkv_vmip} -o "StrictHostKeyChecking no" <<ENDS
cp /tmp/config.json ~/regionless-storage-service/cmd/http/config.json
nohup ~/regionless-storage-service/main --jaeger-server=http://${jaeger_vmip}:14268 >/tmp/rkv.log 2>&1 &
ENDS

## todo: start prometheus server

## now, it is ok to run go-ycsb against rkv service
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
echo "ycsb client ip addr is ${rkv_vmip}"
echo "rkv endpoint is at ${rkv_vmip}:8090"
# set rkv ip addr properly for go-ycsb to test against
ssh -i ${KEY_FILE} ubuntu@${ycsb_vmip} -o "StrictHostKeyChecking no" <<ENDS
sudo sed -i '/rkv/d' /etc/hosts
echo ${rkv_vmip} rkv | sudo tee -a /etc/hosts > /dev/null
ENDS
## run workloada for now; saving output to /tmp/ycsb-a.log
## todo: run more workloads
ssh -i ${KEY_FILE} ubuntu@${ycsb_vmip} -o "StrictHostKeyChecking no" "cd work/go-ycsb && ./bin/go-ycsb load rkv -P workloads/workloada" | tee /tmp/ycsb-a.log
echo "you can run other tests against rkv service now"
echo "you can take a look at tracing at http://${jaeger_vmip}:16686"
