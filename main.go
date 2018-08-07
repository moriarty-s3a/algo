package algo

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"sync"

	log "github.com/sirupsen/logrus"
)

// If we need higher accuracy and had enough knowledge to fix the decimal, this could be implemented as fixed point
// arithmetic instead of floating point
func CalculateRewards(url string) float64 {
	rewardChan := make(chan float64, 1000)
	readerChan := make(chan string, 1000)
	resultChan := make(chan float64, 1)
	cache := make(map[string]cacheentry)
	var cacheLock sync.Mutex

	var readGroup, rewardGroup sync.WaitGroup
	const NUM_WORKERS = 100
	for i:= 0; i<NUM_WORKERS;i++ {
		readGroup.Add(1)
		go worker(readerChan, rewardChan, cache, &cacheLock, &readGroup)
	}
	rewardGroup.Add(1)
	go readRewards(resultChan, rewardChan, &rewardGroup)
	readerChan <- url

	readGroup.Wait()
	close(rewardChan)
	rewardGroup.Wait()
	return <-resultChan
}

func readRewards(resultChan chan float64, rewardChan chan float64, wg *sync.WaitGroup) {
	defer wg.Done()
	total := 0.0
	for reward := range rewardChan {
		total += reward
		log.Debugf("Adding %f, new total %f", reward, total)
	}
	resultChan <- total
}

func worker(readerChan chan string, rewardChan chan float64, cache map[string]cacheentry, cacheLock *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	for url := range readerChan {
		exists, inProgress, reward := getOrSet(url, cache, cacheLock)
		if exists {
			if !inProgress {
				rewardChan <- reward
			} else {
				readerChan <- url
			}
			continue
		}
		log.Debugf("trying url %s", url)
		resp, err := http.Get(url)
		if err != nil {
			log.Debugf("Error %v connecting to URL %s", err, url)
			continue
		}
		buf, err := ioutil.ReadAll(resp.Body)
		var res result
		json.Unmarshal(buf, &res)
		log.Debugf("Url was: %s     Response was: %+v", url, res)
		rewardChan <- res.Reward
		for _, child := range res.Children {
			readerChan <- child
		}
		updateCache(url, res.Reward, cache, cacheLock)
		log.Debugf("Found %d children, reader channel is %d long ", len(res.Children), len(readerChan))
		// Even though this chain is finished, other chains may still be adding to the work queue
		if len(res.Children) == 0 && len(readerChan) == 0 && finished(cache, cacheLock) {
			log.Debugf("Closing after url %s", url)
			close(readerChan)
		}
	}
}
func finished(cache map[string]cacheentry, cacheLock *sync.Mutex) bool{
	cacheLock.Lock()
	defer cacheLock.Unlock()
	for _, entry := range cache {
		if entry.InProgress {
			return false
		}
	}
	return true
}

func updateCache(url string, reward float64, cache map[string]cacheentry, cacheLock *sync.Mutex) {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	entry := cache[url]
	entry.Reward = reward
	entry.InProgress = false
	cache[url] = entry
}

func getOrSet(url string, cache map[string]cacheentry, cacheLock *sync.Mutex) (bool, bool, float64) {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	entry, exists := cache[url]
	if exists {
		log.Debugf("Found %s in the cache", url)
		return true, entry.InProgress, entry.Reward
	} else {
		cache[url] = cacheentry{
			InProgress:true,
		}
		return false, false, 0.0
	}
}

type result struct {
	Children []string `json:children`
	Reward float64 `json:reward`
}

type cacheentry struct {
	Reward float64
	InProgress bool
}