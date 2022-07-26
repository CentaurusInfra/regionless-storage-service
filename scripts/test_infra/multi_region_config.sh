#!/bin/bash

file=../si_config.json

read_region_configs() {
    readarray -t KeyStoreRegions < <(jq -r '.RegionConfigs[].Region' $file) 
    readarray -t KeyNames < <(jq -r '.RegionConfigs[].KeyName' $file) 
    readarray -t KeyFiles < <(jq -r '.RegionConfigs[].FileName' $file) 
    readarray -t AMIs < <(jq -r '.RegionConfigs[].AMI' $file) 
}

read_stores() {
    readarray -t StoreRegions < <(jq -r '.Stores[].Region' $file) 
    readarray -t StoreCounts < <(jq -r '.Stores[].Count' $file) 
    readarray -t StorePorts < <(jq -r '.Stores[].Port' $file) 
    readarray -t StoreInstanceTypes < <(jq -r '.Stores[].InstanceType' $file) 
    readarray -t StoreNamePrefixs < <(jq -r '.Stores[].NamePrefix' $file) 
}

find_key_name() {
    local found=false
    local region_idx=0
    for i in "${!StoreRegions[@]}"; do
        local r=${StoreRegions[$i]}
	if [ "$r" != "$1" ]; then 
	    ((region_idx+=1))
	else
  	    found=true
	    break
        fi	   
    done
    if [ "$found" = true ] ; then
        echo "${KeyNames[$region_idx]}"
    else
        echo "key name not found for region $1"
    fi
}

find_key_file() {
    local found=false
    local region_idx=0
    for i in "${!StoreRegions[@]}"; do
        local r=${StoreRegions[$i]}
	if [ "$r" != "$1" ]; then 
	    ((region_idx+=1))
	else
  	    found=true
	    break
        fi	   
    done
    if [ "$found" = true ] ; then
        echo "${KeyFiles[$region_idx]}"
    else
        echo "key file not found for region $1"
    fi
}

find_ami() {
    local found=false
    local region_idx=0
    for i in "${!StoreRegions[@]}"; do
        local r=${StoreRegions[$i]}
	if [ "$r" != "$1" ]; then 
	    ((region_idx+=1))
	else
  	    found=true
	    break
        fi	   
    done
    if [ "$found" = true ] ; then
        echo "${AMIs[$region_idx]}"
    else
        echo "AMI not found for region $1"
    fi
}
