package simple

import (
	"net/http"
	"reflect"
	"strings"

	"fmt"

	"github.com/go-macaron/inject"
)

type Context struct {
	inject.Injector
	handlers []Handler
	action   Handler
	index    int

	*Router
	Req    Request
	Resp   ResponseWriter
	params Params
	Render
	Locale
	Data map[string]interface{}
}

func (c *Context) handler() Handler {
	if c.index < len(c.handlers) {
		return c.handlers[c.index]
	}

	if c.index == len(c.handlers) {
		return c.action
	}

	panic("invalid index for context handler")

}

func (c *Context) Next() {
	c.index += 1
	c.run()
}

func (c *Context) Written() bool {

	return c.Resp.Written()
}

// use 之后 c.handlers
// 任何中间件 都是 handler
func (c *Context) run() {
	fmt.Println(c.params)
	for c.index <= len(c.handlers) {
		vals, err := c.Invoke(c.handler())

		if err != nil {
			panic(err)
		}
		c.index += 1

		if len(vals) > 0 {
			ev := c.GetVal(reflect.TypeOf(ReturnHandler(nil)))
			handlerReturn := ev.Interface().(ReturnHandler)
			handlerReturn(c, vals)
		}

		if c.Written() {
			return
		}
	}
}

type Request struct {
	*http.Request
}

type Locale interface {
	Language() string
	Tr(string, ...interface{}) string
}

type ContextInvoker func(ctx *Context)

func (invoke ContextInvoker) Invoke(params []interface{}) ([]reflect.Value, error) {
	invoke(params[0].(*Context))
	return nil, nil
}

func (ctx *Context) RemoteAddr() string {
	addr := ctx.Req.Header.Get("X-Real-IP")

	if len(addr) == 0 {
		addr = ctx.Req.Header.Get("X-Forwarded-For")
		if addr == "" {
			addr = ctx.Req.RemoteAddr
			if i := strings.LastIndex(addr, ":"); i > -1 {
				addr = addr[:i]
			}
		}
	}
	return addr
}

func (ctx *Context) Params(name string) string {
	if len(name) == 0 {
		return ""
	}

	if len(name) > 1 && name[0] != ':' {
		name = ":" + name
	}

	return ctx.params[name]
}
