## Setup Guide

### Components
RKV has four major components: 

- RKV service
- Storage instances
- Benchmarking
- Profiling 

All 4 components can be configured and deployed on AWS with a single script.

A [forked version of go-ycsb](https://github.com/CentaurusInfra/go-ycsb) is used for benchmarking. [Jaeger](https://www.jaegertracing.io/) is used for profiling. 

### Configuration
Two configurations files are used. And the following quick-and-easy scripts are provided to utilize sampler configuration files.

- scripts/use_micro.sh
- scripts/use_large.sh

As their names suggest, use_micro.sh sets up config files with t2.micro instance type, and use_large.sh explores more expensive VM types. Pick one to run, and both will generate:
 
- test-lab.val: configurations for RKV, benchmarking and profiling
- si_def_1_region.json: storage instance configuration

The storage instance configuration file is self-explanatory. Here's an example

```json
{
  "RegionConfigs": [
    {
      "Region": "us-west-1",
      "KeyName": "regionless_kv_service_key",
      "FileName": "regionless_kv_service_key_us_west_1.pem",
      "AMI": "ami-067f8db0a5c2309c0"
    },
    {
      "Region": "us-west-2",
      "KeyName": "regionless_kv_service_key",
      "FileName": "regionless_kv_service_key_us_west_2.pem",
      "AMI": "ami-0bf8f78223ea6f3f6"
    },
    {
      "Region": "us-east-1",
      "KeyName": "regionless_kv_service_key",
      "FileName": "regionless_kv_service_key_us_east_1.pem",
      "AMI": "ami-0729e439b6769d6ab"
    },
    {
      "Region": "us-east-2",
      "KeyName": "regionless_kv_service_key",
      "FileName": "regionless_kv_service_key_us_east_2.pem",
      "AMI": "ami-00978328f54e31526"
    }
   ],
  "Stores": [
    {
      "NamePrefix": "rkv-lab-si",
      "Region": "us-west-1",
      "Port": 6666,
      "Count": 2,
      "InstanceType": "t2.large"
    },
    {
      "NamePrefix": "rkv-lab-si",
      "Region": "us-west-2",
      "Port": 6666,
      "Count": 2,
      "InstanceType": "t2.large"
    }
  ]
}
```
Note that key pairs need to be created for each involved region, and the pem files need to be present in the *test_infra* sub-folder. Be aware that security groups, AMI and key pairs are all regional for AWS so these need to be created before the script could run successfully.

### "One-click" Deploy

scripts/setup_test_lab.sh is the script that provision all 4 components of RKV. If run without option, it prints this usage guide:

```
Usage:
  cd scripts
  export KEY_NAME=<your ec2 key name>
  export KEY_FILE=<path-to-ec2-key-file>
  export NAME_TAG=<name_tag to identify your resources>
  and then run ./setup-test-lab.sh [optional: si definition file name, yes, name without path, 'scripts/si_def.json' if not provided
exited.
```

Set the environment variables and then fire up the service. For example
```bash
./setup_test_lab.sh si_def_1_region.json
```

And here's a screenshot of what follows:

<img width="613" alt="image" src="https://user-images.githubusercontent.com/252020/182262687-4d7f2642-6a07-4ccc-a50b-01225670e551.png">

Once the script is done, RKV will be up running and a quick YCSB test will be attempted. 

### Utilities

For easy debugging, a few short scripts are provided to access various hosts:

```
log_into_ycsb.sh
log_into_rkv.sh
log_into_a_si.sh
connect_to_a_si.sh
```
