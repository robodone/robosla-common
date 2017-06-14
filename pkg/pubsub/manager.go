package pubsub

import (
	"log"
	"sync"
)

const universalSubStart = 1E10

type Manager struct {
	mu    sync.Mutex
	nodes map[string]*Node
	subs  map[*Sub]string

	// List of subs subscribed to all current and future nodes.
	uniSubs   map[*Sub]bool
	cnt       int64
	initPaths []string
}

func NewManager(initPaths ...string) *Manager {
	return &Manager{
		nodes:     make(map[string]*Node),
		subs:      make(map[*Sub]string),
		uniSubs:   make(map[*Sub]bool),
		cnt:       universalSubStart,
		initPaths: initPaths,
	}
}

func (m *Manager) getNode(nodeName string) *Node {
	m.mu.Lock()
	defer m.mu.Unlock()
	node, ok := m.nodes[nodeName]
	if !ok {
		node = NewNode(m.initPaths...)
		m.nodes[nodeName] = node
		for sub := range m.uniSubs {
			if err := node.subSub(sub, sub.paths...); err != nil && err != ErrNodeAlreadyStopped {
				// TODO(krasin): do something about it
				log.Printf("Failed to subscribe a node to universal sub with paths: %q, err: %v", sub.paths, err)
			}
		}
	}
	return node
}

func (m *Manager) Sub(nodeName string, paths ...string) (*Sub, error) {
	// log.Printf("Manager.Sub(%q, %q)", nodeName, paths)
	sub, err := m.getNode(nodeName).Sub(paths...)
	if err != nil {
		return nil, err
	}
	m.subs[sub] = nodeName
	return sub, err
}

// SubAll subscribes to all current or future nodes for the specified paths.
func (m *Manager) SubAll(paths ...string) (*Sub, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// TODO(krasin): validate paths. They must be conforming json rules.
	paths = cleanPaths(paths)
	ch := make(chan string, backlogSize)
	m.cnt++
	res := &Sub{paths: paths, ch: ch, id: m.cnt}
	for _, node := range m.nodes {
		if err := node.subSub(res, res.paths...); err != nil && err != ErrNodeAlreadyStopped {
			// TODO(krasin): properly call Unsub on already subscribed nodes.
			return nil, err
		}
	}
	m.uniSubs[res] = true
	return res, nil
}

func (m *Manager) Pub(nodeName, jsonStr string) error {
	return m.getNode(nodeName).Pub(jsonStr)
}

func (m *Manager) Unsub(sub *Sub) {
	nodeName := m.subs[sub]
	if nodeName == "" {
		return
	}
	delete(m.subs, sub)
	m.getNode(nodeName).Unsub(sub)
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, node := range m.nodes {
		node.Stop()
	}
	m.nodes = nil
}
