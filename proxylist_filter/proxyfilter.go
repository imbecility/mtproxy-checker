package proxylist_filter

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/imbecility/mtproxy-checker/single_check"
	"github.com/imbecility/mtproxy-checker/types"
)

// CheckAndFilterProxies многопоточно проверяет список прокси, отбрасывает мертвые и сортирует по пингу.
func CheckAndFilterProxies(proxies []string, maxWorkers int, localAddr string, timeout time.Duration) types.CheckResults {
	log.Printf("[Checker] проверка %d прокси в %d потоков...\n", len(proxies), maxWorkers)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var alive []types.CheckResult
	var dead []string
	var checked atomic.Int64

	sem := make(chan struct{}, maxWorkers)
	total := len(proxies)

	for _, p := range proxies {
		wg.Add(1)
		go func(rawURL string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res, err := single_check.CheckProxy(rawURL, localAddr, timeout)

			mu.Lock()
			if err == nil {
				alive = append(alive, types.CheckResult{URL: rawURL, Ping: res.Ping})
			} else {
				dead = append(dead, rawURL)
			}
			mu.Unlock()

			current := checked.Add(1)
			fmt.Printf("\r[Checker] проверено %d/%d", current, total)
		}(p)
	}

	wg.Wait()
	fmt.Println()

	sort.Slice(alive, func(i, j int) bool {
		return alive[i].Ping < alive[j].Ping
	})

	log.Printf("[Checker] проверка завершена: живых - %d, мертвых - %d\n", len(alive), len(dead))

	var aliveURLs []string
	for _, r := range alive {
		aliveURLs = append(aliveURLs, r.URL)
	}
	return types.CheckResults{
		Alive: aliveURLs,
		Dead:  dead,
	}
}
