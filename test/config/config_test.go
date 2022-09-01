package config

import (
	"testing"

	"github.com/regionless-storage-service/pkg/config"
	"github.com/regionless-storage-service/pkg/constants"
)

func TestLocalStores(t *testing.T) {
	stores := []config.KVStore{
		{
			Region:                "us-east-1",
			AvailabilityZone:      "us-east-1a",
			Name:                  "us-east-11",
			Host:                  "127.0.0.1",
			Port:                  6379,
			ArtificialLatencyInMs: 1,
		},
		{
			Region:                "us-east-2",
			AvailabilityZone:      "us-east-2b",
			Name:                  "us-east-21",
			Host:                  "172.31.9.142",
			Port:                  6379,
			ArtificialLatencyInMs: 100,
		},
		{
			Region:                "us-east-2",
			AvailabilityZone:      "us-east-2b",
			Name:                  "us-east-22",
			Host:                  "172.31.9.141",
			Port:                  6379,
			ArtificialLatencyInMs: 200,
		},
		{
			Region:                "us-east-2",
			AvailabilityZone:      "us-east-1b",
			Name:                  "us-east-21",
			Host:                  "172.31.9.140",
			Port:                  6379,
			ArtificialLatencyInMs: 200,
		},
		{
			Region:                "us-east-1",
			AvailabilityZone:      "us-east-1b",
			Name:                  "us-east-13",
			Host:                  "127.0.0.2",
			Port:                  6379,
			ArtificialLatencyInMs: 10,
		},
		{
			Region:                "us-east-1",
			AvailabilityZone:      "us-east-1c",
			Name:                  "us-east-14",
			Host:                  "127.0.0.6",
			Port:                  6379,
			ArtificialLatencyInMs: 90,
		},
		{
			Region:                "us-east-1",
			AvailabilityZone:      "us-east-1c",
			Name:                  "us-east-15",
			Host:                  "127.0.0.7",
			Port:                  6379,
			ArtificialLatencyInMs: 90,
		},
		{
			Region:                "us-east-1",
			AvailabilityZone:      "us-east-1b",
			Name:                  "us-east-24",
			Host:                  "172.31.9.144",
			Port:                  6379,
			ArtificialLatencyInMs: 200,
		},
	}
	c := &config.KVConfiguration{
		ReplicaNum:                            config.ReplicaNum{Local: 3, Remote: 1},
		StoreType:                             constants.DummyLatency,
		Concurrent:                            false,
		RemoteStoreLatencyThresholdInMilliSec: 100,
		Stores:                                stores,
	}
	localNodes, remoteNodes, err := c.GetReplications()
	if err != nil {
		t.Fatalf("failed to get replications with the eror %v", err)
	}
	if len(localNodes) != 3 {
		t.Fatalf("The local store number is %d", len(localNodes))
	}
	if len(remoteNodes) != 4 {
		t.Fatalf("The remote store number is %d", len(remoteNodes))
	}
	for az, azNodes := range localNodes {
		if az != "us-east-1a" && az != "us-east-1b" && az != "us-east-1c" {
			t.Fatalf("The local availiblity zone %s is not expected", az)
		}
		for _, localNode := range azNodes {
			if localNode[0:9] != "us-east-1" {
				t.Fatalf("The local node %s is not expected", localNode)
			}
		}
	}
}

func TestRemoteStores(t *testing.T) {
	stores := []config.KVStore{
		{
			Region:                "us-east-1",
			AvailabilityZone:      "us-east-1a",
			Name:                  "us-east-11",
			Host:                  "127.0.0.1",
			Port:                  6379,
			ArtificialLatencyInMs: 1,
		},
		{
			Region:                "us-east-2",
			AvailabilityZone:      "us-east-2b",
			Name:                  "us-east-21",
			Host:                  "172.31.9.142",
			Port:                  6379,
			ArtificialLatencyInMs: 100,
		},
		{
			Region:                "us-east-2",
			AvailabilityZone:      "us-east-2b",
			Name:                  "us-east-22",
			Host:                  "172.31.9.141",
			Port:                  6379,
			ArtificialLatencyInMs: 200,
		},
		{
			Region:                "us-east-2",
			AvailabilityZone:      "us-east-1b",
			Name:                  "us-east-21",
			Host:                  "172.31.9.140",
			Port:                  6379,
			ArtificialLatencyInMs: 200,
		},
		{
			Region:                "us-east-1",
			AvailabilityZone:      "us-east-1b",
			Name:                  "us-east-13",
			Host:                  "127.0.0.2",
			Port:                  6379,
			ArtificialLatencyInMs: 10,
		},
		{
			Region:                "us-east-1",
			AvailabilityZone:      "us-east-1c",
			Name:                  "us-east-14",
			Host:                  "127.0.0.6",
			Port:                  6379,
			ArtificialLatencyInMs: 90,
		},
		{
			Region:                "us-east-1",
			AvailabilityZone:      "us-east-1c",
			Name:                  "us-east-15",
			Host:                  "127.0.0.7",
			Port:                  6379,
			ArtificialLatencyInMs: 90,
		},
		{
			Region:                "us-east-1",
			AvailabilityZone:      "us-east-1b",
			Name:                  "us-east-24",
			Host:                  "172.31.9.144",
			Port:                  6379,
			ArtificialLatencyInMs: 200,
		},
	}
	c := &config.KVConfiguration{
		ReplicaNum:                            config.ReplicaNum{Local: 3, Remote: 1},
		StoreType:                             constants.DummyLatency,
		Concurrent:                            false,
		RemoteStoreLatencyThresholdInMilliSec: 100,
		Stores:                                stores,
	}
	localNodes, remoteNodes, err := c.GetReplications()
	if err != nil {
		t.Fatalf("failed to get replications with the eror %v", err)
	}
	if len(localNodes) != 3 {
		t.Fatalf("The local store number is %d", len(localNodes))
	}
	if len(remoteNodes) != 4 {
		t.Fatalf("The remote store number is %d", len(remoteNodes))
	}
	for _, remoteNode := range remoteNodes {
		if remoteNode[0:9] != "us-east-2" {
			t.Fatalf("The remote node %s is not expected", remoteNode)
		}
	}
}
