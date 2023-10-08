package main

import (
	"sync"
)

func main() {
	var (
		wg sync.WaitGroup
		s  = Service(":8080")
	)

	wg.Add(1)
	 _ = s.Start(&wg)
	wg.Wait()
}
