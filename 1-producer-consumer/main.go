//////////////////////////////////////////////////////////////////////
//
// Given is a producer-consumer scenario, where a producer reads in
// tweets from a mockstream and a consumer is processing the
// data. Your task is to change the code so that the producer as well
// as the consumer can run concurrently
//

package main

import (
	"fmt"
	"sync"
	"time"
)

var wg sync.WaitGroup

func producer(stream Stream) chan *Tweet {
	tweetStream := make(chan *Tweet)
	go func() {
		defer wg.Done()
		defer close(tweetStream)
		for {
			tweet, err := stream.Next()
			if err == ErrEOF {
				return
			}
			tweetStream <- tweet
		}
	}()

	return tweetStream
}

func consumer(tweetStream chan *Tweet) {
	go func() {
		defer wg.Done()
		for t := range tweetStream {
			if t.IsTalkingAboutGo() {
				fmt.Println(t.Username, "\ttweets about golang")
			} else {
				fmt.Println(t.Username, "\tdoes not tweet about golang")
			}
		}
	}()
}

func main() {
	start := time.Now()
	stream := GetMockStream()

	wg.Add(2)
	// Producer
	tweetStream := producer(stream)

	// Consumer
	consumer(tweetStream)
	wg.Wait()

	fmt.Printf("Process took %s\n", time.Since(start))
}
