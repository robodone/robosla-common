package pubsub

import "sync"

type Manager struct {
	mu        sync.Mutex
	nodes     map[string]*Node
	subs      map[*Sub]string
	initPaths []string
}

func NewManager(initPaths ...string) *Manager {
	return &Manager{
		nodes:     make(map[string]*Node),
		subs:      make(map[*Sub]string),
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
	}
	return node
}

func (m *Manager) Sub(nodeName string, paths ...string) (*Sub, error) {
	sub, err := m.getNode(nodeName).Sub(paths...)
	if err != nil {
		return nil, err
	}
	m.subs[sub] = nodeName
	return sub, err
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
