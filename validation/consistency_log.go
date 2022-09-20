package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const URL = "http://localhost:8090/kv"
const INT_BASE = 10
const INTER_OPS_MAX_DURATION_MS = 200

var DURATION int64 // in nanosecond
var CLIENT_ID string
var EVENT_ID int64
var KEY string

func checkFatal(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func Read() string {
	start := time.Now().UnixNano()
	resp, err := http.Get(URL + "?key=" + KEY)
	checkFatal(err)

	defer resp.Body.Close()
	end := time.Now().UnixNano()

	body, err := ioutil.ReadAll(resp.Body)
	checkFatal(err)

	value := strings.Split(strings.Split(string(body), "is ")[1], " with")[0]
	revision := strings.Split(strings.Split(string(body), "revision ")[1], "\n")[0]

	output := "read," +
		value + "," +
		strconv.FormatInt(int64(start), INT_BASE) + "," +
		strconv.FormatInt(int64(end), INT_BASE) + "," +
		CLIENT_ID + "," +
		strconv.FormatInt(EVENT_ID, INT_BASE) + "," +
		revision
	return output
}

func Write() string {
	payload, err := json.Marshal(map[string]interface{}{
		"key":   KEY,
		"value": strconv.FormatInt(int64(rand.Intn(1000)), INT_BASE),
	})
	checkFatal(err)

	client := &http.Client{}

	start := time.Now().UnixNano()
	req, err := http.NewRequest(http.MethodPut, URL, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	checkFatal(err)

	resp, err := client.Do(req)
	checkFatal(err)
	end := time.Now().UnixNano()

	body, err := ioutil.ReadAll(resp.Body)
	checkFatal(err)

	value := strings.Split(strings.Split(string(body), ")")[0], ",")[1]
	revision := strings.Split(strings.Split(string(body), "revision ")[1], " at")[0]

	output := "write," +
		value + "," +
		strconv.FormatInt(int64(start), INT_BASE) + "," +
		strconv.FormatInt(int64(end), INT_BASE) + "," +
		CLIENT_ID + "," +
		strconv.FormatInt(EVENT_ID, INT_BASE) + "," +
		revision
	return output
}

func main() {
	args := os.Args
	CLIENT_ID = args[1]
	_duration, err := strconv.ParseInt(args[2], INT_BASE, 64)
	DURATION = _duration * 1000000000
	KEY = args[3]

	rand.Seed(time.Now().UnixNano())

	f, err := os.Create("./events_client_" + CLIENT_ID + ".log")
	checkFatal(err)
	defer f.Close()

	var output string
	start := time.Now().UnixNano()

	for time.Now().UnixNano()-start < DURATION {
		EVENT_ID++

		// The test key may be already existing
		// To make sure porcupine work, the first operation cannot be READ
		// Otherwise read the existing value will makes porcupine think it is not correct
		// Manually set every first operation as WRITE
		if EVENT_ID == 1 {
			output = Write()
			_, err := f.WriteString(output + "\n")
			checkFatal(err)
		} else {
			// select a random operation
			if rand.Intn(2) == 0 {
				output = Read()
				_, err := f.WriteString(output + "\n")
				checkFatal(err)
			} else {
				output = Write()
				_, err := f.WriteString(output + "\n")
				checkFatal(err)
			}
		}

		// sleep for simulating random inter-operation durations
		time.Sleep(time.Duration(rand.Intn(INTER_OPS_MAX_DURATION_MS)) * time.Millisecond)
	}

	f.Close()
}
