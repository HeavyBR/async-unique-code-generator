package main

import (
	"awesomeProject3/buffer"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var quantity int64
var size int
var prefix string
var output string

func buildKey(code string) string {
	return fmt.Sprintf("%s:%s", "prefix", code)
}

const (
	letterBytes = "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	flag.Int64Var(&quantity, "quantity", 10_000, "the quantity of codes that will be generated. E.g: -quantity 10000")
	flag.IntVar(&size, "size", 10, "the size of each generated code. E.g: -size 10")
	flag.StringVar(&prefix, "prefix", "", "select a prefix to each generated code. E.g: -prefix PEPSI")
	flag.StringVar(&output, "output", "codes.txt", "filename to save the codes. E.g: -output file.txt")
	flag.Parse()
}

func main() {
	GenerateUniqueCodes(size, quantity, prefix, buildKey)
}

func GenerateUniqueCodes(size int, quantity int64, prefix string, keyBuilder func(code string) string) {
	var start = time.Now()

	// Channels
	var ch = make(chan string, quantity)
	var created = make(chan string, quantity)
	var topic = make(chan string, quantity)
	var duplicates = make(chan string, quantity)

	// Databases
	var codes = make(map[string]bool)
	var memcached = make(map[string]bool) // Mock to a distributed cache

	// Sync
	var m = &sync.Mutex{}
	var wg = &sync.WaitGroup{}

	// Buffers
	var buf = &buffer.Buffer{}

	// Misc
	var chunkSize = quantity / 100
	var sent int64

	wg.Add(1)
	go func() {
		var workers = quantity / chunkSize
		for int64(len(created)) < quantity {
			for i := 0; int64(i) < workers; i++ {
				go func() {
					for i := 0; int64(i) < chunkSize; i++ {
						code := prefix + SecureRandomString(letterBytes, size-len(prefix))
						m.Lock()
						codes[code] = true
						m.Unlock()
						ch <- code
					}
				}()
			}

		loop:
			for {
				select {
				case code := <-ch:
					go func(code string) {
						time.Sleep(10 * time.Millisecond) // Simulate network latency

						// Verify existence
						m.Lock()
						if _, ok := memcached[keyBuilder(code)]; ok {
							m.Unlock()
							duplicates <- code
							return
						}
						m.Unlock()

						time.Sleep(10 * time.Millisecond) // Simulate network latency
						// Write to cache
						m.Lock()
						memcached[keyBuilder(code)] = true
						m.Unlock()

						time.Sleep(10 * time.Millisecond) // Simulate network latency
						// Send to topic
						topic <- code
						created <- code

						// Write to file
						fmt.Fprintf(buf, "%s\n", code)

						atomic.AddInt64(&sent, 1)
					}(code)
				default:
					if (sent == (quantity - int64(len(duplicates)))) && len(duplicates) > 0 {
						break loop
					}

					if atomic.LoadInt64(&sent) == quantity {
						break loop
					}
				}
			}

			if int64(len(codes)) < quantity {
				newQuantity := quantity - int64(len(codes))
				if newQuantity < chunkSize {
					chunkSize = newQuantity
					workers = 1
				} else {
					workers = newQuantity / chunkSize
				}
			}
		}

		wg.Done()
	}()

	wg.Wait()
	f, _ := os.Create(output)
	f.Write(buf.Bytes())

	fmt.Println(fmt.Sprintf("%.2f seconds", time.Since(start).Seconds()))
}

func SecureRandomString(availableCharBytes string, length int) string {

	// Compute bitMask
	availableCharLength := len(availableCharBytes)
	if availableCharLength == 0 || availableCharLength > 256 {
		log.Panicln("availableCharBytes length must be greater than 0 and less than or equal to 256")
	}
	var bitLength byte
	var bitMask byte
	for bits := availableCharLength - 1; bits != 0; {
		bits = bits >> 1
		bitLength++
	}
	bitMask = 1<<bitLength - 1

	// Compute bufferSize
	bufferSize := length + length/3

	// Create random string
	result := make([]byte, length)
	for i, j, randomBytes := 0, 0, []byte{}; i < length; j++ {
		if j%bufferSize == 0 {
			// Random byte buffer is empty, get a new one
			randomBytes = SecureRandomBytes(bufferSize)
		}
		// Mask bytes to get an index into the character slice
		if idx := int(randomBytes[j%length] & bitMask); idx < availableCharLength {
			result[i] = availableCharBytes[idx]
			i++
		}
	}

	return string(result)
}

// SecureRandomBytes returns the requested number of bytes using crypto/rand
func SecureRandomBytes(length int) []byte {
	var randomBytes = make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatal("Unable to generate random bytes")
	}
	return randomBytes
}
