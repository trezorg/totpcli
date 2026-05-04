package search

import (
	"fmt"
	"strings"

	"github.com/trezorg/totpcli/internal/provider"
)

const (
	EngineDigram  = "digram"
	EngineLexical = "lexical"
)

type Engine interface {
	Search(query string) []provider.Provider
}

func NewEngine(name string, providers []provider.Provider) (Engine, error) {
	name, err := ParseEngineName(name)
	if err != nil {
		return nil, err
	}

	switch name {
	case EngineDigram:
		return NewDigram(providers), nil
	case EngineLexical:
		return NewLexical(providers), nil
	default:
		return nil, fmt.Errorf("unsupported search engine %q: expected digram or lexical", name)
	}
}

func ParseEngineName(name string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", EngineDigram:
		return EngineDigram, nil
	case EngineLexical:
		return EngineLexical, nil
	default:
		return "", fmt.Errorf("unsupported search engine %q: expected digram or lexical", name)
	}
}

type Lexical struct {
	providers []provider.Provider
}

func NewLexical(providers []provider.Provider) Lexical {
	return Lexical{providers: append([]provider.Provider(nil), providers...)}
}

func (l Lexical) Search(query string) []provider.Provider {
	query = normalize(query)
	if query == "" {
		return append([]provider.Provider(nil), l.providers...)
	}

	results := make([]provider.Provider, 0, len(l.providers))
	for _, candidate := range l.providers {
		if matches(query, candidate) {
			results = append(results, candidate)
		}
	}
	return results
}

type Digram struct {
	providers []provider.Provider
	root      *digramNode
}

type digramNode struct {
	children map[string]*digramNode
	indexes  []int
}

func NewDigram(providers []provider.Provider) Digram {
	engine := Digram{
		providers: append([]provider.Provider(nil), providers...),
		root:      newDigramNode(),
	}
	for index, candidate := range engine.providers {
		engine.insert(index, candidate.Issuer)
		engine.insert(index, candidate.Name)
		engine.insert(index, candidate.DisplayLabel())
	}
	return engine
}

func (d Digram) Search(query string) []provider.Provider {
	query = normalize(query)
	if query == "" {
		return append([]provider.Provider(nil), d.providers...)
	}

	node := d.root
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
		results = append(results, d.providers[index])
	}
	return results
}

func (d Digram) insert(providerIndex int, value string) {
	runes := []rune(normalize(value))
	for start := range runes {
		node := d.root
		for i := start; i < len(runes); i++ {
			key := string(runes[i])
			if i > start {
				key = string([]rune{runes[i-1], runes[i]})
			}

			child := node.children[key]
			if child == nil {
				child = newDigramNode()
				node.children[key] = child
			}
			child.add(providerIndex)
			node = child
		}
	}
}

func newDigramNode() *digramNode {
	return &digramNode{children: make(map[string]*digramNode)}
}

func (n *digramNode) add(providerIndex int) {
	if len(n.indexes) > 0 && n.indexes[len(n.indexes)-1] == providerIndex {
		return
	}
	n.indexes = append(n.indexes, providerIndex)
}

func matches(query string, candidate provider.Provider) bool {
	return strings.Contains(normalize(candidate.Issuer), query) ||
		strings.Contains(normalize(candidate.Name), query) ||
		strings.Contains(normalize(candidate.DisplayLabel()), query)
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
