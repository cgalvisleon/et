package gateway

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/cgalvisleon/elvis/cache"
	"github.com/cgalvisleon/elvis/et"
	"github.com/cgalvisleon/elvis/strs"
	"github.com/cgalvisleon/elvis/utility"
)

type Node struct {
	_id     string
	Tag     string
	Resolve et.Json
	Nodes   []*Node
}

type Nodes struct {
	Routes []*Node
}

type Pakage struct {
	Name  string
	Nodes []*Node
	Count int
}

type Pakages struct {
	Pakages []*Pakage
}

type Resolve struct {
	Node    *Node
	Params  []et.Json
	Resolve string
}

type Handlers map[string]http.HandlerFunc

// Create new router
func newRouters() *Nodes {
	return &Nodes{
		Routes: []*Node{},
	}
}

// Create new pakages
func newPakages() *Pakages {
	return &Pakages{
		Pakages: []*Pakage{},
	}
}

// Create new handlers
func newHandlers() Handlers {
	return make(Handlers)
}

// Create a new node from routes
func newNode(tag string, nodes []*Node) (*Node, []*Node) {
	result := &Node{
		_id:     utility.UUID(),
		Tag:     tag,
		Resolve: et.Json{},
		Nodes:   []*Node{},
	}

	nodes = append(nodes, result)

	return result, nodes
}

// Find a node from routes
func findNode(tag string, nodes []*Node) *Node {
	for _, node := range nodes {
		if node.Tag == tag {
			return node
		}
	}

	return nil
}

func findResolve(tag string, nodes []*Node, route *Resolve) (*Node, *Resolve) {
	node := findNode(tag, nodes)
	if node == nil {
		// Define regular expression
		regex := regexp.MustCompile(`^\{.*\}$`)
		// Find node by regular expression
		for _, n := range nodes {
			if regex.MatchString(n.Tag) {
				if route == nil {
					route = &Resolve{
						Params: []et.Json{},
					}
				}
				route.Node = n
				route.Params = append(route.Params, et.Json{n.Tag: tag})
				return n, route
			}
		}
	} else if route == nil {
		route = &Resolve{
			Node:   node,
			Params: []et.Json{},
		}
	} else {
		route.Node = node
	}

	return node, route
}

// Load routes from file
func (s *HttpServer) LoadRouter() error {
	_routes, err := cache.Get(s.routesKey, "{routes:[]}")
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(_routes), &s.routes)
	if err != nil {
		return err
	}

	_pakages, err := cache.Get(s.pakagesKey, "{pakages:[]}")
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(_pakages), &s.pakages)
	if err != nil {
		return err
	}

	return nil
}

// Save routes to file
func (s *HttpServer) Save() error {
	// Convertion struct to json
	_routes, err := et.Marshal(s.routes)
	if err != nil {
		return err
	}

	// Save json to cache
	err = cache.Set(s.routesKey, _routes.ToString(), 0)
	if err != nil {
		return err
	}

	_pakages, err := et.Marshal(s.pakages)
	if err != nil {
		return err
	}

	err = cache.Set(s.pakagesKey, _pakages.ToString(), 0)
	if err != nil {
		return err
	}

	return nil
}

// Find a pakage by name
func (s *HttpServer) findPakage(name string) *Pakage {
	for _, pakage := range s.pakages.Pakages {
		if pakage.Name == name {
			return pakage
		}
	}

	return nil
}

// Create a new pakage
func (s *HttpServer) newPakage(name string) *Pakage {
	pakage := &Pakage{
		Name:  name,
		Nodes: []*Node{},
	}

	s.pakages.Pakages = append(s.pakages.Pakages, pakage)

	return pakage
}

// Add a route to the list
func (s *HttpServer) AddRoute(method, path, resolve, kind, stage, packageName string) {
	node := findNode(method, s.routes.Routes)
	if node == nil {
		node, s.routes.Routes = newNode(method, s.routes.Routes)
	}

	tags := strings.Split(path, "/")
	for _, tag := range tags {
		if len(tag) > 0 {
			find := findNode(tag, node.Nodes)
			if find == nil {
				node, node.Nodes = newNode(tag, node.Nodes)
			} else {
				node = find
			}
		}
	}

	if node != nil {
		node.Resolve = et.Json{
			"method":  method,
			"kind":    kind,
			"stage":   stage,
			"resolve": resolve,
		}

		pakage := s.findPakage(packageName)
		if pakage == nil {
			pakage = s.newPakage(packageName)
		}
		pakage.Nodes = append(pakage.Nodes, node)
		pakage.Count = len(pakage.Nodes)
	}
}

func (s *HttpServer) AddHandleMethod(method, path string, handlerFn http.HandlerFunc, packageName string) {
	node := findNode(method, s.routes.Routes)
	if node == nil {
		node, s.routes.Routes = newNode(method, s.routes.Routes)
	}

	tags := strings.Split(path, "/")
	for _, tag := range tags {
		if len(tag) > 0 {
			find := findNode(tag, node.Nodes)
			if find == nil {
				node, node.Nodes = newNode(tag, node.Nodes)
			} else {
				node = find
			}
		}
	}

	if node != nil {
		node.Resolve = et.Json{
			"method": method,
			"kind":   "HANDLER",
		}
		s.handlers[node._id] = handlerFn

		pakage := s.findPakage(packageName)
		if pakage == nil {
			pakage = s.newPakage(packageName)
		}
		pakage.Nodes = append(pakage.Nodes, node)
		pakage.Count = len(pakage.Nodes)
	}
}

// Get a route from the list
func (s *HttpServer) GetResolve(method, path string) *Resolve {
	node := findNode(method, s.routes.Routes)
	if node == nil {
		return nil
	}

	var result *Resolve
	tags := strings.Split(path, "/")
	for _, tag := range tags {
		if len(tag) > 0 {
			node, result = findResolve(tag, node.Nodes, result)
			if node == nil {
				return nil
			}
		}
	}

	if result != nil {
		result.Resolve = node.Resolve.Str("resolve")
		for _, param := range result.Params {
			for key, value := range param {
				result.Resolve = strings.Replace(result.Resolve, key, "%v", -1)
				result.Resolve = strs.Format(result.Resolve, value)
			}
		}
	}

	return result
}
