package simple

import (
	"fmt"
	"github.com/Unknwon/com"
	"github.com/go-macaron/inject"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
)

type Simple struct {
	inject.Injector
	befores  []BeforeHandler
	handlers []Handler
	action   Handler

	hasURLPrefix bool
	urlPrefix    string
	*Router

	logger *log.Logger
}

type Handler interface{}
type BeforeHandler func(rw http.ResponseWriter, req *http.Request) bool
type handlerFuncInvoker func(http.ResponseWriter, *http.Request)

// 如果是 fastInvoker 然后 inject 去调用Ivoker, 如果设置的话 就会重写Invoker
// 核心调用在 inject
func (invoke handlerFuncInvoker) Invoke(params []interface{}) ([]reflect.Value, error) {

	return nil, nil
}

type internalServerErrorInvoker func(rw http.ResponseWriter, err error)

func newWithLogger(out io.Writer) *Simple {
	m := &Simple{
		Injector: inject.New(),
		action:   func() {},
		Router:   NewRouter(),
		logger:   log.New(out, "[Simple] ", 0),
	}

	m.Router.m = m
	// inject 里的Map
	m.Map(m.logger)
	m.Map(defaultReturnHandler())

	// 使用 http 的notFound
	m.NotFound(http.NotFound)
	m.InternalServerError(func(rw http.ResponseWriter, err error) {
		http.Error(rw, err.Error(), 500)
	})
	fmt.Println(1)
	return m
}

func New() *Simple {
	return newWithLogger(os.Stdout)
}

func Classic() *Simple {
	m := New()
	m.Use(Logger())
	m.Use(Recovery())
	m.Use(Static("public"))
	return m
}

const (
	DEV  = "development"
	PROD = "production"
	TEST = "test"
)

var (
	Env     = DEV
	envLock sync.Mutex
	Root    string
)

// 创建上下文
func (m *Simple) createContext(rw http.ResponseWriter, req *http.Request) *Context {
	c := &Context{
		Injector: inject.New(),
		handlers: m.handlers,
		action:   m.action,
		index:    0,
		Router:   m.Router,
		Req:      Request{req},
		Resp:     NewResponseWriter(req.Method, rw),
		Render:   &DummyRender{rw},
		Data:     make(map[string]interface{}),
	}

	c.SetParent(m)
	c.Map(c)
	c.MapTo(c.Resp, (*http.ResponseWriter)(nil))
	c.Map(req)
	return c
}

func (m *Simple) Use(handlers Handler) {
	handlers = validateAndWrapHandler(handlers)
	m.handlers = append(m.handlers, handlers)
}

func GetDefaultListenInfo() (string, int) {
	host := os.Getenv("HOST")
	if len(host) == 0 {
		host = "0.0.0.0"
	}

	port := com.StrTo(os.Getenv("PORT")).MustInt()
	if port == 0 {
		port = 4000
	}

	return host, port
}

func (m *Simple) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if m.hasURLPrefix {
		req.URL.Path = strings.TrimPrefix(req.URL.Path, m.urlPrefix)
	}

	for _, h := range m.befores {
		if h(rw, req) {
			return
		}
	}

	m.Router.ServeHTTP(rw, req)
}

func (m *Simple) Run(args ...interface{}) {
	host, port := GetDefaultListenInfo()

	if len(args) == 1 {
		switch arg := args[0].(type) {
		case string:
			host = arg
		case int:
			port = arg
		}
	} else if len(args) >= 2 {
		if arg, ok := args[0].(string); ok {
			host = arg
		}

		if arg, ok := args[0].(int); ok {
			port = arg
		}
	}

	addr := host + ":" + com.ToStr(port)
	logger := m.GetVal(reflect.TypeOf(m.logger)).Interface().(*log.Logger)
	logger.Printf("listening on %s  (%s) \n", addr, safeEnv())
	logger.Fatalln(http.ListenAndServe(addr, m))
}

func safeEnv() string {
	envLock.Lock()
	defer envLock.Unlock()

	return Env
}

// 判断是否为Func
// 在反射中函数和方法的类型（Type）都是reflect.Func

// 如果handler 不是 isFastInvoker
// 转化成相应的invoker
func validateAndWrapHandler(h Handler) Handler {
	if reflect.TypeOf(h).Kind() != reflect.Func {
		// 此处 如此处理 欠妥当
		panic("func must bu callable function")
	}

	if !inject.IsFastInvoker(h) {
		switch v := h.(type) {
		case func(*Context):
			return ContextInvoker(v)
		case func(*Context, *log.Logger):
			return LoggerInvoker(v)
		case func(http.ResponseWriter, *http.Request):
			return handlerFuncInvoker(v)
		case func(http.ResponseWriter, error):
			return internalServerErrorInvoker(v)
		}
	}

	return h

}

// 循环 传过来的handlers, 将handler 转化， 再使用 wrappers 包裹
// 没太懂 这快是干嘛用的
func validateAndWrapHandlers(handlers []Handler, wrappers ...func(Handler) Handler) []Handler {
	var wrapper func(Handler) Handler
	if len(wrappers) > 0 {
		wrapper = wrappers[0]
	}

	wrappedHandlers := make([]Handler, len(handlers))
	for i, h := range handlers {
		h = validateAndWrapHandler(h)
		if wrapper != nil && !inject.IsFastInvoker(h) {
			h = wrapper(h)
		}

		wrappedHandlers[i] = h
	}

	return wrappedHandlers
}
