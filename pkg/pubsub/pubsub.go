package pubsub

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// Right now, backlog is implemented poorly, as if it's full, it's the new messages
// which are discarded, not the old ones. This is why the size is that large.
const backlogSize = 10

type Node struct {
	mu       sync.Mutex
	stopped  bool
	cnt      int
	subPaths map[string][]*Sub
	subs     []*Sub
	state    map[string]interface{}
}

func NewNode(initPaths ...string) *Node {
	res := &Node{
		subPaths: make(map[string][]*Sub),
		state:    make(map[string]interface{}),
	}
	for _, p := range initPaths {
		pp := strings.Split(p, ".")
		setIfCan(res.state, pp, make(map[string]interface{}))
	}
	return res
}

func (nd *Node) Stop() {
	nd.mu.Lock()
	defer nd.mu.Unlock()

	for _, sub := range nd.subs {
		close(sub.ch)
	}
	nd.stopped = true
	nd.subPaths = nil
	nd.subs = nil
	nd.state = nil
}

type subSlice []*Sub

func (ss subSlice) Len() int           { return len(ss) }
func (ss subSlice) Less(i, j int) bool { return ss[i].id < ss[j].id }
func (ss subSlice) Swap(i, j int)      { ss[i], ss[j] = ss[j], ss[i] }

func scanPathsInternal(prefix string, m map[string]interface{}, paths *[]string) {
	for k, v := range m {
		curPath := prefix + k
		*paths = append(*paths, curPath)
		if v == nil {
			continue
		}
		switch v.(type) {
		case string:
			continue
		case float64:
			continue
		case bool:
			continue
		case []interface{}:
			log.Printf("Error: scanPathInternal: arrays not supported. Ignore.")
			continue
		case map[string]interface{}:
			scanPathsInternal(curPath+".", v.(map[string]interface{}), paths)
		}
	}
}

func scanPaths(m map[string]interface{}) []string {
	var paths []string
	scanPathsInternal("", m, &paths)
	sort.Strings(paths)
	return paths
}

func (nd *Node) Pub(jsonStr string) error {
	nd.mu.Lock()
	defer nd.mu.Unlock()
	if nd.stopped {
		return errors.New("node is already stopped")
	}
	// log.Printf("Pub(%s), prior state: %s", jsonStr, mustJson(nd.state))
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return err
	}
	paths := scanPaths(m)
	subsM := make(map[*Sub]bool)
	for _, p := range paths {
		for _, sub := range nd.subPaths[p] {
			subsM[sub] = true
		}
		assignIfCan(nd.state, m, p)
	}
	var subs []*Sub
	for sub, _ := range subsM {
		subs = append(subs, sub)
	}
	sort.Sort(subSlice(subs))
	for _, sub := range subs {
		sub.update(m)
	}

	// log.Printf("Pub(%s), after state: %s", jsonStr, mustJson(nd.state))
	return nil
}

func cleanPaths(paths []string) []string {
	sort.Strings(paths)
	res := make([]string, 0, len(paths))
	lastIdx := -1
	last := ""
	for i, p := range paths {
		if lastIdx >= 0 && strings.HasPrefix(p, last) {
			// It's a dup or a subset, skip
			continue
		}
		lastIdx = i
		last = p
		res = append(res, p)
	}
	return res
}

func mustJson(m map[string]interface{}) string {
	data, err := json.Marshal(m)
	if err != nil {
		panic(fmt.Sprintf("mustJson: %v", err))
	}
	return string(data)
}

func (nd *Node) Sub(paths ...string) (*Sub, error) {
	nd.mu.Lock()
	defer nd.mu.Unlock()
	if nd.stopped {
		return nil, errors.New("node is already stopped")
	}
	// TODO(krasin): validate paths. They must be conforming json rules.
	paths = cleanPaths(paths)
	ch := make(chan string, backlogSize)
	nd.cnt++
	res := &Sub{paths: paths, ch: ch, id: nd.cnt}
	for _, p := range paths {
		nd.subPaths[p] = append(nd.subPaths[p], res)
	}
	nd.subs = append(nd.subs, res)
	// log.Printf("Sub(%q), state: %s", paths, mustJson(nd.state))
	res.update(nd.state)
	return res, nil
}

func (nd *Node) SubString(path string) (*StringSub, error) {
	sub, err := nd.Sub(path)
	if err != nil {
		return nil, err
	}
	return newStringSub(nd, sub, path), nil
}

func removeSub(subs []*Sub, sub *Sub, shouldClose bool) []*Sub {
	for i := 0; i < len(subs); i++ {
		v := subs[i]
		if v != sub {
			continue
		}
		if shouldClose {
			close(v.ch)
		}
		subs[i] = subs[len(subs)-1]
		subs = subs[:len(subs)-1]
		if i == len(subs)-1 {
			break
		}
		i--
	}
	return subs
}

func (nd *Node) Unsub(sub *Sub) {
	nd.mu.Lock()
	defer nd.mu.Unlock()
	if nd.stopped {
		return
	}

	// Dumb: scan everything and delete from everywhere.
	nd.subs = removeSub(nd.subs, sub, true)
	for p, subs := range nd.subPaths {
		nd.subPaths[p] = removeSub(subs, sub, false)
	}
}

type Sub struct {
	id    int
	paths []string
	ch    chan string
}

func (s *Sub) C() <-chan string {
	return s.ch
}

func getIfCan(m map[string]interface{}, pp []string) (interface{}, bool) {
	switch len(pp) {
	case 0:
		return m, true
	case 1:
		val, ok := m[pp[0]]
		return val, ok
	default:
		val, ok := m[pp[0]]
		if !ok || val == nil {
			return nil, false
		}
		switch val.(type) {
		case map[string]interface{}:
			return getIfCan(val.(map[string]interface{}), pp[1:])
		case []interface{}:
			log.Printf("Error: getIfCan: arrays not implemented. Skip.")
			return nil, false
		default:
			// Simple value. We can't go inside.
			return nil, false
		}
	}
}

func mergeObjects(dest, src map[string]interface{}) {
	for k, val := range src {
		if val == nil || dest[k] == nil {
			dest[k] = val
			continue
		}
		switch dest[k].(type) {
		case map[string]interface{}:
			switch val.(type) {
			case map[string]interface{}:
				mergeObjects(dest[k].(map[string]interface{}), val.(map[string]interface{}))
			default:
				dest[k] = val
			}
		default:
			dest[k] = val
		}
	}
}

func setIfCan(m map[string]interface{}, pp []string, val interface{}) {
	// log.Printf("setIfCan(m: %+v, pp: %q, val: %+v", m, pp, val)
	if len(pp) == 1 {
		// This is a top-level path. Set the value directly, unless both sides are object.
		// In the latter case, we need to merge paths
		if val == nil || m[pp[0]] == nil {
			m[pp[0]] = val
			return
		}
		switch val.(type) {
		case bool:
			m[pp[0]] = val
		case float64:
			m[pp[0]] = val
		case string:
			m[pp[0]] = val
		case []interface{}:
			log.Printf("Array values are not supported. Skip.")
		case map[string]interface{}:
			dest := m[pp[0]]
			switch dest.(type) {
			case map[string]interface{}:
				// We need to merge two objects.
				mergeObjects(dest.(map[string]interface{}), val.(map[string]interface{}))
			default:
				m[pp[0]] = val
			}
		default:
			log.Printf("Unsupported value type: %v", val)
		}
		return
	}
	// Deep path.
	mm, ok := m[pp[0]]
	if !ok || mm.(map[string]interface{}) == nil {
		// Create a new object and replace existing value (if any).
		m[pp[0]] = make(map[string]interface{})
		mm = m[pp[0]]
	}
	setIfCan(mm.(map[string]interface{}), pp[1:], val)
}

func assignIfCan(dest, src map[string]interface{}, p string) {
	if p == "" {
		return
	}
	pp := strings.Split(p, ".")
	val, ok := getIfCan(src, pp)
	if !ok {
		// this path is not present in src. Skip.
		return
	}
	setIfCan(dest, pp, val)
}

func (s *Sub) update(m map[string]interface{}) {
	// We need to create a subset of m to only notify about the changes in the subscribed paths.
	res := make(map[string]interface{})
	// TODO(krasin): implement
	for _, p := range s.paths {
		assignIfCan(res, m, p)
	}
	data, err := json.Marshal(res)
	if err != nil {
		panic(fmt.Errorf("update: failed to marshal: %v", err))
	}
	msg := string(data)
	if msg == "{}" {
		// Skip an empty update.
		return
	}
	select {
	case s.ch <- msg:
	default:
		// The destination has lost this update, but we don't want to lock on them anyway.
	}
}

type StringSub struct {
	nd   *Node
	sub  *Sub
	path string
	ch   chan string
}

func (ss *StringSub) C() <-chan string {
	return ss.ch
}

func newStringSub(nd *Node, sub *Sub, path string) *StringSub {
	ss := &StringSub{
		nd:   nd,
		sub:  sub,
		path: path,
		ch:   make(chan string, backlogSize),
	}
	go ss.run()
	return ss
}

func (ss *StringSub) run() {
	defer close(ss.ch)
	for msg := range ss.sub.C() {
		// log.Printf("StringSub got msg: %s\n", msg)
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(msg), &m); err != nil {
			log.Printf("Error: invalid json from Sub. That should never happen, but since it did, we just ignore.")
		}
		pp := strings.Split(ss.path, ".")
		val, ok := getIfCan(m, pp)
		if !ok {
			// this path is not present in src. Skip.
			log.Printf("Error: received unnecessary update; wanted path %q was not found in the message")
			continue
		}
		var str string
		if val != nil {
			str, ok = val.(string)
			if !ok {
				log.Printf("Error: received an update where %q is not a string, but %v", reflect.TypeOf(val))
				str = ""
			}
		}
		ss.ch <- str
	}
}

func (ss *StringSub) Unsub() {
	ss.nd.Unsub(ss.sub)
}
