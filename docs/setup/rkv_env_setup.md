# regionless-storage-service
Development Environment Setup

The following steps are to be done as root

## 1. Clone the Repo

```bash
mkdir -p ~/go/src/github.com
cd ~/go/src/github.com
git clone https://github.com/CentaurusInfra/regionless-storage-service.git
```

##  2. Install Redis

```bash
cd ~/go/src/github.com/regionless-storage-service
./scripts/install_redis.sh
```
Please run the following command to ensure that the redis is running by seeing "active (running)" in green

```bash
sudo systemctl status redis
```
Another alternative is to run a redis cli command as follows

```bash
redis-cli -h 127.0.0.1 -p 6379
127.0.0.1:6379> PING
PONG
```

## 3. Update the config.json

Please visit the config.json to check the backend setup.
```bash
cat cmd/http/config.json
{
    "ConsistentHash": "rendezvous",
    "BucketSize": 2,
    "StoreType": "redis",
    "Stores": [
        {
            "Name": "store1",
            "Host": "127.0.0.1",
            "Port": 6379
        }
    ]
}
```

## 4. Install Development Environment

The following command is to set up golang dev environemt
```bash
./script/setup_env.sh
```


## 5. Start Key Value Store

The following command is to set up a kv store
```bash
./script/start_kv.sh
```

## 6. Curl Commands for CRUD

```bash
curl -X POST -k http://localhost:8090/kv -d '{"key":"key1", "value": "v1"}'
curl -X PUT -k http://localhost:8090/kv -d '{"key":"key1", "value": "v2"}
curl -X DELETE http://localhost:8090/kv/key1
curl -sS 'http://localhost:8090/kv?key=key1'
curl -sS 'http://localhost:8090/kv?key=key1&fromRev=1'
```
