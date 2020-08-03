package dns

import (
	"github.com/emirpasic/gods/maps/hashmap"
	"github.com/miekg/dns"
	"sync"
)

type Cache struct {
	*hashmap.Map
	mutex sync.Mutex
}

func NewCache() *Cache {
	return &Cache{
		Map:   hashmap.New(),
		mutex: sync.Mutex{},
	}
}

func (cache *Cache) Set(question dns.Question, answer dns.RR) {

}

func (cache *Cache) Get() {

}
