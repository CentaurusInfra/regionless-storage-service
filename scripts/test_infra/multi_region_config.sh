#!/bin/bash

read_region_configs() {
    readarray -t KeyStoreRegions < <(jq -r '.RegionConfigs[].Region' ${SI_DEF_FILE}) 
    readarray -t KeyNames < <(jq -r '.RegionConfigs[].KeyName' ${SI_DEF_FILE}) 
    readarray -t KeyFiles < <(jq -r '.RegionConfigs[].FileName' ${SI_DEF_FILE}) 
    readarray -t AMIs < <(jq -r '.RegionConfigs[].AMI' ${SI_DEF_FILE}) 
}

read_stores() {
    readarray -t StoreRegions < <(jq -r '.Stores[].Region' ${SI_DEF_FILE}) 
    readarray -t StoreCounts < <(jq -r '.Stores[].Count' ${SI_DEF_FILE}) 
    readarray -t StorePorts < <(jq -r '.Stores[].Port' ${SI_DEF_FILE}) 
    readarray -t StoreInstanceTypes < <(jq -r '.Stores[].InstanceType' ${SI_DEF_FILE}) 
    readarray -t StoreNamePrefixs < <(jq -r '.Stores[].NamePrefix' ${SI_DEF_FILE}) 
}

find_key_name() {
    local found=false
    local region_idx=0
    for i in "${!KeyStoreRegions[@]}"; do
        local r=${KeyStoreRegions[$i]}
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
    for i in "${!KeyStoreRegions[@]}"; do
        local r=${KeyStoreRegions[$i]}
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
    for i in "${!KeyStoreRegions[@]}"; do
        local r=${KeyStoreRegions[$i]}
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
