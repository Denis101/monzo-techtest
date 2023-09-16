package crawler

import "sync"

type hashSet struct {
	data map[string]bool
	lock sync.RWMutex
}

func (s *hashSet) add(t string) *hashSet {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.data == nil {
		s.data = make(map[string]bool)
	}
	_, ok := s.data[t]
	if !ok {
		s.data[t] = true
	}
	return s
}

func (s *hashSet) addSlice(a []string) *hashSet {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.data == nil {
		s.data = make(map[string]bool)
	}

	for _, t := range a {
		_, ok := s.data[t]
		if !ok {
			s.data[t] = true
		}
	}
	return s
}

func (s *hashSet) has(element string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	_, ok := s.data[element]
	return ok
}

func (s *hashSet) slice() []string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	result := make([]string, len(s.data))
	i := 0
	for k := range s.data {
		result[i] = k
		i++
	}
	return result
}

func (s *hashSet) size() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.data)
}
