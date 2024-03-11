package apigateway

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/cgalvisleon/et/et"
)

type Node struct {
	_id     string
	Tag     string
	Resolve et.Json
	Handler http.HandlerFunc
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

type Routes interface {
}

type Router interface {
	http.Handler
	Routes

	// Use appends one or more middlewares onto the Router stack.
	Use(middlewares ...func(http.Handler) http.Handler)

	// With adds inline middlewares for an endpoint handler.
	With(middlewares ...func(http.Handler) http.Handler) Router

	// Group adds a new inline-Router along the current routing
	// path, with a fresh middleware stack for the inline-Router.
	Group(fn func(r Router)) Router

	// Route mounts a sub-Router along a `pattern`` string.
	Route(pattern string, fn func(r Router)) Router

	// Mount attaches another http.Handler along ./pattern/*
	Mount(pattern string, h http.Handler)

	// Handle and HandleFunc adds routes for `pattern` that matches
	// all HTTP methods.
	Handle(pattern string, h http.Handler)
	HandleFunc(pattern string, h http.HandlerFunc)

	// Method and MethodFunc adds routes for `pattern` that matches
	// the `method` HTTP method.
	Method(method, pattern string, h http.Handler)
	MethodFunc(method, pattern string, h http.HandlerFunc)

	// HTTP-method routing along `pattern`
	Connect(pattern string, h http.HandlerFunc)
	Delete(pattern string, h http.HandlerFunc)
	Get(pattern string, h http.HandlerFunc)
	Head(pattern string, h http.HandlerFunc)
	Options(pattern string, h http.HandlerFunc)
	Patch(pattern string, h http.HandlerFunc)
	Post(pattern string, h http.HandlerFunc)
	Put(pattern string, h http.HandlerFunc)
	Trace(pattern string, h http.HandlerFunc)

	// NotFound defines a handler to respond whenever a route could
	// not be found.
	NotFound(h http.HandlerFunc)

	// MethodNotAllowed defines a handler to respond whenever a method is
	// not allowed.
	MethodNotAllowed(h http.HandlerFunc)
}

// List of routes
var routes *Nodes
var pakages *Pakages

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
	}
}

func AddHandleMethod(method, path string, handlerFn http.HandlerFunc) {
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
			"method": method,
			"kind":   "HANDLER",
		}
		node.Handler = handlerFn
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
}
