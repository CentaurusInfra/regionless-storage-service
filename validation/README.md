# CONSISTENCY VALIDATION GUIDE

## STEP 1: Deploy a multiple-node working environment

Example:
- Create 4 Instances (1 us-east-1, 1 us-east-2, 1 us-west-1, 1 us-west-2)
- Configure the number of sync and async nodes and latency threshold
    - Change the parameters in `scripts/test_infra/create_test_instance.sh`
    - Parameters: `localreplicanum`, `remotereplicanum`, `remotestorelatencythresholdinmillisec`
- Modify read behaviors if necessary
    - In `pkg/piping/sync_async_piping_manager.go`
    - An example of reading from a random selected sync node
    ```
    func (sap *SyncAsyncPiping) Read(ctx context.Context, rev index.Revision) (string, error) {
        ...
        target := ""
        if len(syncNodes) > 0 {
                rand.Seed(time.Now().UnixNano()) 
                inx := rand.Intn(len(syncNodes)) 
                target = syncNodes[inx]
        } else if len(asyncNodes) > 0 {
                target = asyncNodes[0]
        } else {
                return "", fmt.Errorf("the rev %v does not have any nodes", rev)
        }
        ...
    }
    ```

## STEP 2: Build code and run consistency tests
- `$cd validation`
- `$go build consistency_log.go`
- Modify the test parameters in `run_consistency.sh`: `num_clients`, `duration`, `key`
- Run a test `$./run_consistency.sh`
- Log files will be generated per client under the current directory.

## STEP 3: Run porcupine tests for linearizability validation
- `$cd validation/porcupine`
- `$go test`
- The results will show "PASS" or "FAIL"
- The visualization file will be saved to `/tmp/XXXXXX.html` shown in the output.  
