package lsp

import "sync"

type DocumentStore struct {
	mu   sync.RWMutex
	docs map[string]string
}

func NewDocumentStore() *DocumentStore {
	return &DocumentStore{docs: make(map[string]string)}
}

func (s *DocumentStore) Open(uri, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = text
}

func (s *DocumentStore) Update(uri, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = text
}

func (s *DocumentStore) Close(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, uri)
}

func (s *DocumentStore) Get(uri string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.docs[uri]
}
