package apigateway

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
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
}

type Pakages struct {
	Pakages []*Pakage
}

type Resolve struct {
	Node    *Node
	Params  []et.Json
	Resolve string
}

// List of routes
var routes *Nodes
var pakages *Pakages
var routesKey = "apigateway/routes"
var pakagesKey = "apigateway/packages"

// Load routes from file
func load() error {
	_routes, err := cache.Get(routesKey, "{routes:[]}")
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(_routes), &routes)
	if err != nil {
		return err
	}

	_pakages, err := cache.Get(pakagesKey, "{pakages:[]}")
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(_pakages), &pakages)
	if err != nil {
		return err
	}

	return nil
}

// Save routes to file
func save() error {
	// Convertion struct to json
	_routes, err := et.Marshal(routes)
	if err != nil {
		return err
	}

	// Save json to cache
	err = cache.Set(routesKey, _routes.ToString(), 0)
	if err != nil {
		return err
	}

	_pakages, err := et.Marshal(pakages)
	if err != nil {
		return err
	}

	err = cache.Set(pakagesKey, _pakages.ToString(), 0)
	if err != nil {
		return err
	}

	return nil
}

// Create a new node from routes
func newNode(tag string, nodes []*Node) (*Node, []*Node) {
	result := &Node{
		_id:     et.UUID(),
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

func findPakage(name string) *Pakage {
	for _, pakage := range pakages.Pakages {
		if pakage.Name == name {
			return pakage
		}
	}

	return nil
}

func newPakage(name string) *Pakage {
	pakage := &Pakage{
		Name:  name,
		Nodes: []*Node{},
	}

	pakages.Pakages = append(pakages.Pakages, pakage)

	return pakage
}

// Add a route to the list
func AddRoute(method, path, resolve, kind, stage, packageName string) {
	node := findNode(method, routes.Routes)
	if node == nil {
		node, routes.Routes = newNode(method, routes.Routes)
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

		pakage := findPakage(packageName)
		if pakage == nil {
			pakage = newPakage(packageName)
			pakage.Nodes = append(pakage.Nodes, node)
		}

		save()
	}
}

// Get a route from the list
func GetResolve(method, path string) *Resolve {
	node := findNode(method, routes.Routes)
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
				result.Resolve = et.Format(result.Resolve, value)
			}
		}
	}

	return result
}

// Init routes
func init() {
	routes = &Nodes{
		Routes: []*Node{},
	}

	pakages = &Pakages{
		Pakages: []*Pakage{},
	}

	load()
}
