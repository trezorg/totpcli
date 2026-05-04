package digram

import (
	"strings"

	"github.com/trezorg/totpcli/internal/provider"
)

type Engine struct {
	providers []provider.Provider
	root      *node
}

type node struct {
	children map[string]*node
	indexes  []int
}

func New(providers []provider.Provider) Engine {
	engine := Engine{
		providers: append([]provider.Provider(nil), providers...),
		root:      newNode(),
	}
	for index, candidate := range engine.providers {
		engine.insert(index, candidate.Issuer)
		engine.insert(index, candidate.Name)
		engine.insert(index, candidate.DisplayLabel())
	}
	return engine
}

func (e Engine) Search(query string) []provider.Provider {
	query = normalize(query)
	if query == "" {
		return append([]provider.Provider(nil), e.providers...)
	}

	node := e.root
	runes := []rune(query)
	for i := range runes {
		key := string(runes[i])
		if i > 0 {
			key = string([]rune{runes[i-1], runes[i]})
		}

		next := node.children[key]
		if next == nil {
			return nil
		}
		node = next
	}

	results := make([]provider.Provider, 0, len(node.indexes))
	for _, index := range node.indexes {
		results = append(results, e.providers[index])
	}
	return results
}

func (e Engine) insert(providerIndex int, value string) {
	runes := []rune(normalize(value))
	for start := range runes {
		node := e.root
		for i := start; i < len(runes); i++ {
			key := string(runes[i])
			if i > start {
				key = string([]rune{runes[i-1], runes[i]})
			}

			child := node.children[key]
			if child == nil {
				child = newNode()
				node.children[key] = child
			}
			child.add(providerIndex)
			node = child
		}
	}
}

func newNode() *node {
	return &node{children: make(map[string]*node)}
}

func (n *node) add(providerIndex int) {
	if len(n.indexes) > 0 && n.indexes[len(n.indexes)-1] == providerIndex {
		return
	}
	n.indexes = append(n.indexes, providerIndex)
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
