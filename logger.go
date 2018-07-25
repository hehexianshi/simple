package simple

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"time"
)

var (
	ColorLog      = true
	LogTimeFormat = "2006-01-02 15:04:05"
)

func Logger() Handler {
	return func(ctx *Context, log *log.Logger) {
		start := time.Now()

		log.Printf("%s: Started %s %s for %s", time.Now().Format(LogTimeFormat), ctx.Req.Method, ctx.Req.RequestURI, ctx.RemoteAddr())
		rw := ctx.Resp.(ResponseWriter)
		ctx.Next()

		content := fmt.Sprintf("%s: Completed %s %s %v %s in %v", time.Now().Format(LogTimeFormat), ctx.Req.Method, ctx.Req.RequestURI, rw.Status(), http.StatusText(rw.Status()), time.Since(start))
		if ColorLog {
			switch rw.Status() {
			case 200, 201, 202:
				content = fmt.Sprintf("\033[1;32m%s\033[0m", content)
			case 301, 302:
				content = fmt.Sprintf("\033[1;37m%s\033[0m", content)
			case 304:
				content = fmt.Sprintf("\033[1;33m%s\033[0m", content)
			case 401, 403:
				content = fmt.Sprintf("\033[4;31m%s\033[0m", content)
			case 404:
				content = fmt.Sprintf("\033[1;31m%s\033[0m", content)
			case 500:
				content = fmt.Sprintf("\033[1;36m%s\033[0m", content)
			}
		}

		log.Println(content)
	}
}

type LoggerInvoker func(ctx *Context, log *log.Logger)

func (invoke LoggerInvoker) Invoke(params []interface{}) ([]reflect.Value, error) {
	invoke(params[0].(*Context), params[1].(*log.Logger))
	return nil, nil
}
