# Regionless KV Service (RKV)

## What Is RKV?
RKV overcomes various limitations of ETCD such as storage capacity, and provides a "regionless" style storage service for large scale geo-distributed platform such as [Centaurus Arktos](https://github.com/CentaurusInfra/arktos). Versioned key-value pairs are managed (geo-partition, replicated with flexible consistency) and exposed for access with compatible APIs of ETCD such as range query and list-watch. 

## Highlighted Features

<img width="70%" alt="image" src="https://user-images.githubusercontent.com/252020/182258636-8c0d7e09-da4e-4209-b9f0-3c4f11e50c53.png">

- Region-agnostic data access API
- Partitioned and horrizontally scalable data storage with open backend store options
- Replicated for HA and fast data access
- Versioned Key-value pairs
- CRUD API together with range query and list-watch
- Supporting batch KV access (known as "txn" in ETCD)
- Flexible (Configurable) replication consistency including (but not limited to) linearizability, sequential, "session", and eventual consistency
- Smart caching for high performance data access

## Data Model

<img width="60%" alt="image" src="https://user-images.githubusercontent.com/252020/182257499-e2bc8954-1519-46ab-baa5-9464f8b92eb9.png">


## Architecture

<img width="80%" alt="image" src="https://user-images.githubusercontent.com/252020/174407480-ed632074-0daf-4bd0-9169-85519f46b3eb.png">

## Setup Guide
A one-click deploy script is provided to provision a full set of running RKV with multiple backend storage instances from multiple regions. [Here](docs/setup/multi_region_setup.md) is the setup guide.

## Community Meetings 

Pacific Time: **Tuesday, 6:00PM PT (Weekly).** Please check our discussion page [here](https://github.com/CentaurusInfra/arktos/discussions/1422) for the latest meeting information. 
