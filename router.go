package simple

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

var _HTTP_METHODS = map[string]bool{
	"GET":  true,
	"POST": true,
}

type Route struct {
	router *Router
	leaf   *Leaf
}

type Router struct {
	m        *Simple
	autoHead bool
	routers  map[string]*Tree
	*routeMap
	namedRoutes map[string]*Leaf

	groups              []group
	notFound            http.HandlerFunc
	internalServerError func(*Context, error)

	handlerWapper func(Handler) Handler
}

func (r *Router) NotFound(handlers ...Handler) {
	handlers = validateAndWrapHandlers(handlers)

	r.notFound = func(rw http.ResponseWriter, req *http.Request) {
		c := r.m.createContext(rw, req)
		c.handlers = make([]Handler, 0, len(r.m.handlers)+len(handlers))
		c.handlers = append(c.handlers, r.m.handlers)
		c.handlers = append(c.handlers, handlers)
		c.run()
	}
}

func (r *Router) InternalServerError(handlers ...Handler) {
	handlers = validateAndWrapHandlers(handlers)
	r.internalServerError = func(c *Context, err error) {
		c.index = 0
		c.handlers = handlers
		c.Map(err)
		c.run()
	}
}

func (r *Router) Handle(method string, pattern string, handlers []Handler) *Route {
	if len(r.groups) > 0 {

	}
	handlers = validateAndWrapHandlers(handlers, r.handlerWapper)
	return r.handle(method, pattern, func(resp http.ResponseWriter, req *http.Request, params Params) {
		c := r.m.createContext(resp, req)
		c.params = params
		c.handlers = make([]Handler, 0, len(r.m.handlers)+len(handlers))
		c.handlers = append(c.handlers, r.m.handlers...)
		c.handlers = append(c.handlers, handlers...)
		c.run()
	})
}

func (r *Router) handle(method string, pattern string, handle Handle) *Route {
	method = strings.ToUpper(method)
	var leaf *Leaf

	if leaf = r.getLeaf(method, pattern); leaf != nil {
		return &Route{r, leaf}
	}

	if !_HTTP_METHODS[method] && method != "*" {
		panic("unknown http method")
	}

	methods := make(map[string]bool)
	if method == "*" {
		for m := range _HTTP_METHODS {
			methods[m] = true
		}
	} else {
		methods[method] = true
	}

	for m := range methods {
		if t, ok := r.routers[m]; ok {
			leaf = t.Add(pattern, handle)
		} else {
			t := NewTree()
			leaf = t.Add(pattern, handle)
			r.routers[m] = t
		}

		r.add(m, pattern, leaf)
	}
	return &Route{r, leaf}

}

func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if t, ok := r.routers[req.Method]; ok {
		leaf := r.getLeaf(req.Method, req.URL.Path)
		if leaf != nil {
			leaf.handle(rw, req, nil)
			return
		}

		h, p, ok := t.Match(req.URL.EscapedPath())
		if ok {
			if splat, ok := p["*0"]; ok {
				p["*"] = splat
			}
			h(rw, req, p)
			return
		}
	}

	r.NotFound(rw, req)
}

func (r *Router) Get(pattern string, h ...Handler) (leaf *Route) {
	leaf = r.Handle("GET", pattern, h)
	return leaf
}

func (r *Router) Post(pattern string, h ...Handler) (leaf *Route) {
	return r.Handle("POST", pattern, h)
}

type routeMap struct {
	lock   sync.RWMutex
	routes map[string]map[string]*Leaf
}

type group struct {
	pattern  string
	handlers []Handler
}

func NewRouter() *Router {
	return &Router{
		routers:     make(map[string]*Tree),
		routeMap:    NewRouteMap(),
		namedRoutes: make(map[string]*Leaf),
	}
}

func NewRouteMap() *routeMap {
	rm := &routeMap{
		routes: make(map[string]map[string]*Leaf),
	}

	for m := range _HTTP_METHODS {
		rm.routes[m] = make(map[string]*Leaf)
	}

	return rm
}

func (rm *routeMap) getLeaf(method, pattern string) *Leaf {
	rm.lock.RLock()
	defer rm.lock.RUnlock()
	fmt.Println(rm.routes)
	return rm.routes[method][pattern]
}

func (rm *routeMap) add(method, pattern string, leaf *Leaf) {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	rm.routes[method][pattern] = leaf
}

type Handle func(http.ResponseWriter, *http.Request, Params)
type Params map[string]string
