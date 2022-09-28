#!/bin/bash

if [ $# -ne 2 ]
then
    echo "Usage:"
    echo "  select_config.sh [num of regions] [vm size]"
    exit 0
fi

svc_cfg=./config_examples/test-lab.val.$2
si_cfg=./config_examples/si_def_$1_region_$2.json

if [[ -f $svc_cfg && -f $si_cfg ]]; then
    svc_soft_link=test-lab.val
    si_soft_link=si_def.json
    echo ">>> using $si_cfg (linked to $si_soft_link) and $svc_cfg (linked to $svc_soft_link)" 
else
    echo ">>> Make sure both $si_cfg and $svc_cfg exist. It's nice to exist." 
    exit 0
fi

rm -f $svc_soft_link $si_soft_link 

ln -s $svc_cfg $svc_soft_link 
ln -s $si_cfg $si_soft_link

echo
echo ">>>----------------"
ls -la $svc_soft_link $si_soft_link
echo ">>>----------------"
echo

echo "Config OK. Now deploy with:"
echo " ./setup_test_lab.sh $si_soft_link"
