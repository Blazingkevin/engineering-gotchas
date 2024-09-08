package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

/**

I would like to think of the asynchronous processing in the case to be for paying salaries of staffs provided in say an excel sheet

The core idea here being the use of vault, lock and key analogy.

The vault keeps all keys to open  clients' locks. A client could be an organization that wants their excel sheet of salaries processed.

A manager is in charge of processing the transaction.

In our case, we hired 3 managers.

It is important to make sure that two managers don't access the vault at once,

Both account managers might go for the same key, we don't want two managers workin on the same task.

When a manager gets a key, he tells other managers; hey you can use the vault now.

You have returned the key and telling other managers they can use the vault?

What if another manager picks up the key and goes for the same task?

Oh okay, once a manager gets a key to a client and makes the vault available, he has to announce again - "I am on client tttttt!, don't come close😠"

He locks access to the client and focuses on just that client.


In reality, this whole thing will most likely be an implementation in a distributed system.

In that case you can imagine each go routine to be a separate node. Then you are tempted to ask,
mapping this oversimplified code to a distributed system, how do we maintain a distributed storage to replicate VaultKeyMap and VaultKeyMutex
with the appropriate locking mechanism. Well, technologies like Redis provides a mechanism for distributed locking using Redis-based distributed locks.

Speaking of redis distributed locks, there is a serious bottleneck in this oversimplified code that is worth highlighting...

Each manager locks the client, to process the transactions associated with that client alone. How long should this processing take? Given that we are
retrying calls, what if the calls are taking longer than expected? Just imagine something goes wrong and the manager never announces that he is done? This is
a common problem when dealing with locks. Redis locks typically have an expiration to prevent scenarios where a lock is held indefinitely due to a crashed node or long processing time.


How really are we ensuring fairness ?

There are two primary ways we are ensuring fairness in here,

1. Each transaction batch is submitted to a queue, and transactions are processed in the order they arrive
2. Once a manager starts processing a client, other managers are locked out from working on the same client until the task is completed.

This ensures that each client’s transactions are processed fairly and in the order they are received, avoiding race conditions or out-of-order execution.


...and that's episode 1, I hope to have the time during the week to come organize the above
*/

// represents a batch of transactions for a client
type TransactionBatch struct {
	clientID      int
	transactionID int
	transactions  []string // Example: list of transaction records (like salary payments)
}

//	a channel for submitting transaction batches
//
// intentionally buffered (size 10) because the test case here has less than transactions
// what if we have more than 10 transactions ? well, for this oversimplified  case, the calling go routine is blocked after
// 10 entries except of course our managers do their job fast enough
var TransactionQueue = make(chan TransactionBatch, 10)

// this is like a vault holding the locks (keys) for each client's account
var VaultKeyMap = make(map[int]*sync.Mutex)

// to control access to the vault itself (to avoid conflicts), we don't want more than one manager looking into the vault for key
var VaultKeyMutex = sync.Mutex{}

// defines the number of times to retry a failed transaction
const maxRetries = 3

// defines the time to wait before retrying (increased with each retry)
const retryBackoff = time.Second

// simulates an account manager processing transactions
func AccountManager(managerID int, wg *sync.WaitGroup) {
	defer wg.Done()

	for batch := range TransactionQueue {
		fmt.Printf("Account Manager %d received transaction batch %d for client %d\n", managerID, batch.transactionID, batch.clientID)

		// Lock the vault to get the key for this client's account
		VaultKeyMutex.Lock()
		clientLock, exists := VaultKeyMap[batch.clientID]
		if !exists {
			clientLock = &sync.Mutex{}
			VaultKeyMap[batch.clientID] = clientLock
		}
		VaultKeyMutex.Unlock()

		// Lock the client's account to make sure only this manager processes their transactions
		clientLock.Lock()
		fmt.Printf("Account Manager %d is processing transaction batch %d for client %d\n", managerID, batch.transactionID, batch.clientID)

		// Process each transaction with retry logic in case of failure
		for _, transaction := range batch.transactions {
			success := processWithRetries(managerID, batch.clientID, batch.transactionID, transaction)
			if !success {
				fmt.Printf("Failed to process transaction %s for client %d (batch %d) after %d retries\n", transaction, batch.clientID, batch.transactionID, maxRetries)
			}
		}

		// Unlock the client's account once all transactions are processed
		clientLock.Unlock()
		fmt.Printf("Account Manager %d finished processing transaction batch %d for client %d\n", managerID, batch.transactionID, batch.clientID)
	}
}

// processes a transaction and retries on failure
func processWithRetries(managerID, clientID, transactionID int, transaction string) bool {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if processTransaction(managerID, clientID, transactionID, transaction) {
			return true
		}

		// Log the retry attempt
		fmt.Printf("Account Manager %d retrying transaction %s for client %d (batch %d), attempt %d\n", managerID, transaction, clientID, transactionID, attempt)

		// Wait before retrying (backoff)
		time.Sleep(retryBackoff * time.Duration(attempt))
	}
	return false
}

// simulates the processing of a single transaction ()
// returns true if successful, false on failure (simulated failure)
func processTransaction(managerID, clientID, transactionID int, transaction string) bool {
	fmt.Printf("Account Manager %d processing transaction %s for client %d (batch %d)\n", managerID, transaction, clientID, transactionID)

	// Simulate random failure (e.g network or system issue)
	if rand.Float32() < 0.3 { // 30% chance of failure
		fmt.Printf("Account Manager %d encountered an error processing transaction %s for client %d (batch %d)\n", managerID, transaction, clientID, transactionID)
		return false
	}

	// Simulate successful processing
	time.Sleep(100 * time.Millisecond) // Simulate processing time
	fmt.Printf("Account Manager %d successfully processed transaction %s for client %d (batch %d)\n", managerID, transaction, clientID, transactionID)
	return true
}

func main() {
	var wg sync.WaitGroup

	// Start multiple account managers
	numManagers := 3
	for i := 1; i <= numManagers; i++ {
		wg.Add(1)
		go AccountManager(i, &wg)
	}

	// Simulate submitting transaction batches for different clients
	transactionBatches := []TransactionBatch{
		{clientID: 1, transactionID: 1, transactions: []string{"Salary A", "Salary B", "Salary C"}},
		{clientID: 2, transactionID: 2, transactions: []string{"Salary D", "Salary E", "Salary F"}},
		{clientID: 1, transactionID: 3, transactions: []string{"Salary G", "Salary H", "Salary I"}},
		{clientID: 3, transactionID: 4, transactions: []string{"Salary J", "Salary K", "Salary L"}},
		{clientID: 2, transactionID: 5, transactions: []string{"Salary M", "Salary N", "Salary O"}},
	}

	// Submit the transaction batches into the TransactionQueue
	for _, batch := range transactionBatches {
		TransactionQueue <- batch
	}

	// Close the queue after submitting all transaction batches
	close(TransactionQueue)

	// Wait for all account managers to finish
	wg.Wait()
}
