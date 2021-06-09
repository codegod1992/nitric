package worker

import (
	"fmt"
	"sync"
	"time"

	"github.com/nitric-dev/membrane/handler"
	pb "github.com/nitric-dev/membrane/interfaces/nitric/v1"
)

type FaasWorkerPool struct {
	maxWorkers int
	workerLock sync.Mutex
	workers    []*FaasWorker
}

// Ensure workers implement the trigger handler interface
func (s *FaasWorkerPool) GetTriggerHandler() (handler.TriggerHandler, error) {
	s.workerLock.Lock()
	defer s.workerLock.Unlock()

	if len(s.workers) > 0 {
		return s.workers[0], nil
	} else {
		return nil, fmt.Errorf("No available workers in this pool!")
	}
}

// Synchronously wait for at least one active worker
func (s *FaasWorkerPool) WaitForActiveWorkers(timeout int) error {
	// Dial the child port to see if it's open and ready...
	maxWaitTime := time.Duration(timeout) * time.Second
	// Longer poll times, e.g. 200 milliseconds results in slow lambda cold starts (15s+)
	pollInterval := time.Duration(15) * time.Millisecond

	var waitedTime = time.Duration(0)
	for {
		if s.getWorkerCount() >= 1 {
			break
		} else {
			if waitedTime < maxWaitTime {
				time.Sleep(pollInterval)
				waitedTime += pollInterval
			} else {
				return fmt.Errorf("No server available, has the FaaS grpc client been started?")
			}
		}
	}

	return nil
}

func (s *FaasWorkerPool) getWorkerCount() int {
	s.workerLock.Lock()
	defer s.workerLock.Unlock()
	return len(s.workers)
}

// Add a New FaaS worker to this pool
func (s *FaasWorkerPool) AddWorker(stream pb.Faas_TriggerStreamServer) error {
	s.workerLock.Lock()
	defer s.workerLock.Unlock()
	workerCount := len(s.workers)

	// Ensure we haven't reached the maximum number of workers
	if workerCount > s.maxWorkers {
		return fmt.Errorf("Max worker capacity reached! Cannot add more workers!")
	}

	// Add a new worker to this pool
	worker := newFaasWorker(stream)
	s.workers[workerCount] = worker

	worker.listen()

	return nil
}

func NewFaasWorkerPool() *FaasWorkerPool {
	return &FaasWorkerPool{
		// Only need one at the moment, but leaving open to future proofing
		maxWorkers: 1,
		workerLock: sync.Mutex{},
		// Pre-allocate this for efficiency
		workers: make([]*FaasWorker, 1),
	}
}
