package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	NumProcesses       = 5
	requestProbability = 0.6
)

type LamportMutex struct {
	Timestamp  int
	Requesting bool
	ReplyCount int
}

type Message struct {
	FromProcess int
	Type        string
	Timestamp   int
}

var mutexes [NumProcesses]LamportMutex
var timestamps [NumProcesses]int
var queues [NumProcesses][]Message
var wg sync.WaitGroup
var messageChans [NumProcesses]chan Message

func init() {
	for i := 0; i < NumProcesses; i++ {
		mutexes[i].Timestamp = 0
		mutexes[i].Requesting = false
		mutexes[i].ReplyCount = 0
		timestamps[i] = 0
		queues[i] = []Message{}
		messageChans[i] = make(chan Message, NumProcesses)
	}
}

func process(processID int) {
	mutex := &mutexes[processID]
	for {
		rand.Seed(time.Now().UnixNano())
		randomNumber := rand.Float64()

		if randomNumber < requestProbability {
			mutex.Requesting = true
			mutex.Timestamp++
			timestamps[processID] = mutex.Timestamp
			queues[processID] = insertInQueue(queues[processID], Message{FromProcess: processID, Type: "request", Timestamp: mutex.Timestamp})

			// Send request message to other processes
			tempTimestamp := mutex.Timestamp
			for i := 0; i < NumProcesses; i++ {
				if i != processID {
					messageChans[i] <- Message{FromProcess: processID, Type: "request", Timestamp: tempTimestamp}
				}
			}

			fmt.Printf("Process %d requesting critical section #time: %d\n", processID, mutex.Timestamp)
		} else {
			time.Sleep(1000 * time.Millisecond)
			continue
		}

		// Keep checking if it's the process's turn and received all reply messages
		for !mutex.Requesting || !isFirstInQueue(processID) || mutex.ReplyCount != NumProcesses-1 {
			time.Sleep(10 * time.Millisecond)
		}

		// Critical section
		fmt.Printf("\n%d accessing critical section\n\n", processID)
		time.Sleep(1 * time.Second)
		fmt.Printf("\n%d leaving critical section\n\n", processID)

		mutex.ReplyCount = 0 // Reset reply count

		// Send release message to other processes
		for i := 0; i < NumProcesses; i++ {
			if i != processID {
				messageChans[i] <- Message{FromProcess: processID, Type: "release", Timestamp: mutex.Timestamp}
			}
		}

		mutex.Requesting = false

		fmt.Printf("Process %d releasing the critical section\n", processID)
	}
}

func receiveMessages(processID int) {
	for {
		message := <-messageChans[processID]

		switch message.Type {
		case "request":
			timestamps[message.FromProcess] = max(timestamps[message.FromProcess], mutexes[processID].Timestamp) + 1
			mutexes[processID].Timestamp = timestamps[message.FromProcess]
			queues[processID] = insertInQueue(queues[processID], message)
			//fmt.Printf("queue of %d: %d\n", processID, queues[processID])
			messageChans[message.FromProcess] <- Message{FromProcess: processID, Type: "reply", Timestamp: mutexes[processID].Timestamp} // Send reply message
		case "release":
			queues[processID] = removeFromQueue(queues[processID], message.FromProcess, processID)
		case "reply":
			mutexes[processID].ReplyCount++
			fmt.Printf("%d got REPLY from %d\n", processID, message.FromProcess)
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func isFirstInQueue(processID int) bool {
	if len(queues[processID]) > 0 && queues[processID][0].FromProcess == processID {
		return true
	}
	return false
}

func canEnterCriticalSection(processID int) bool {
	for _, msg := range queues[processID] {
		if timestamps[msg.FromProcess] < timestamps[processID] ||
			(timestamps[msg.FromProcess] == timestamps[processID] && msg.FromProcess < processID) {
			return false
		}
	}
	return true
}

func insertInQueue(queue []Message, message Message) []Message {
	index := 0
	for i, msg := range queue {
		if msg.Timestamp > message.Timestamp || (msg.Timestamp == message.Timestamp && msg.FromProcess > message.FromProcess) {
			break
		}
		index = i + 1
	}

	queue = append(queue, Message{})
	copy(queue[index+1:], queue[index:])
	queue[index] = message

	return queue
}

func removeFromQueue(queue []Message, processID int, printProcess int) []Message {
	index := -1
	for i, msg := range queue {
		if msg.FromProcess == processID {
			index = i
			break
		}
	}

	if index != -1 {
		fmt.Printf("%d removing  (%d, %d)\n", printProcess, queue[index].FromProcess, queue[index].Timestamp)
		queue = append(queue[:index], queue[index+1:]...)
	}

	return queue
}

func main() {
	wg.Add(NumProcesses)

	for i := 0; i < NumProcesses; i++ {
		go receiveMessages(i)
	}

	for i := 0; i < NumProcesses; i++ {
		go process(i)
	}

	wg.Wait()
}
