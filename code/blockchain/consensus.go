package blockchain

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	mrand "math/rand"
	"time"
)

// Constants to define the minimum and maximum waiting time for PoET.
const (
	MinPoetWait = 1 * time.Second
	MaxPoetWait = 10 * time.Second
)

// generatePoetWaitTime generates a random waiting time between the defined minimum and maximum waiting time.
func generatePoetWaitTime() time.Duration {
	min := big.NewInt(int64(MinPoetWait))
	max := big.NewInt(int64(MaxPoetWait))
	delta := new(big.Int).Sub(max, min)

	r, err := rand.Int(rand.Reader, delta)
	if err != nil {
		// Handle error
		fmt.Println("Error generating random number: ", err)
	}
	return time.Duration(r.Int64()) + MinPoetWait
}

// Poet implements Proof of Elapsed Time (PoET) by waiting for a random amount of time.
func (block *Block) PoET(stopChan chan bool, sleepMode chan bool) (error, uint64) {
	waitTime := generatePoetWaitTime() //
	fmt.Println("PoET waiting time: ", waitTime)

	// Check the sleep mode status and adjust accordingly.
	select {
	case status, ok := <-sleepMode:
		if ok && !status {
			sleepMode <- true
		}
	default:
		sleepMode <- true
	}

	// Start PoET.
	select {
	case <-stopChan:
		select {
		case <-sleepMode:
			fmt.Println("Mining aborted due to sleep signal.")
			return errors.New("mining aborted"), uint64(0)
		default:
			fmt.Println("No sleep signal received.")
		}
	case <-time.After(waitTime):
		select {
		case <-sleepMode:
			sleepMode <- false // Node is awake
		default:
			// In case no goroutine is ready to receive from sleepMode channel.
		}
	}
	return nil, uint64(waitTime)
}

// ProofOfWork performs the Proof of Work (PoW) algorithm on a given block hash.
func ProofOfWork(blockHash []byte, difficulty uint8, ch chan bool) (uint64, float64) {
	var (
		Target  = big.NewInt(1)
		intHash = big.NewInt(1)
		nonce   = uint64(mrand.Intn(math.MaxUint32))
		hash    []byte
		count   float64
	)
	Target.Lsh(Target, 256-uint(difficulty))
	start := time.Now()
	for nonce < math.MaxUint64 {
		count++
		select {
		case <-ch:
			if DEBUG {
				fmt.Println()
				fmt.Printf("time: %v: hash: %v\n", time.Since(start).Seconds(), count)
			}
			return nonce, time.Since(start).Seconds()
		default:
			hash = HashSum(bytes.Join(
				[][]byte{
					blockHash,
					ToBytes(nonce),
				},
				[]byte{},
			))
			if DEBUG {
				fmt.Printf("\rMining: %s", Base64Encode(hash))
			}
			intHash.SetBytes(hash)
			if intHash.Cmp(Target) == -1 {
				if DEBUG {
					fmt.Println()
				}
				fmt.Printf("%v: %v\n", time.Since(start).Seconds(), count)
				return nonce, time.Since(start).Seconds()
			}
			nonce++
		}
	}
	return nonce, time.Since(start).Seconds()
}
