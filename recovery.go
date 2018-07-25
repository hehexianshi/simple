package simple

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"

	"github.com/go-macaron/inject"
)

const (
	panicHtml = `<html></html>`
)

var (
	dunno     = []byte("???")
	dot       = []byte(".")
	slash     = []byte("/")
	conterDot = []byte(",")
)

func stack(skip int) []byte {
	buf := new(bytes.Buffer)

	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)

		if !ok {
			break
		}

		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}

			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}

		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.Bytes()
}

func Recovery() Handler {
	return func(c *Context, log *log.Logger) {

		defer func() {
			if err := recover(); err != nil {
				stack := stack(3)
				log.Printf("PANIC: %s\n%s", err, stack)

				val := c.GetVal(inject.InterfaceOf((*http.ResponseWriter)(nil)))
				res := val.Interface().(http.ResponseWriter)

				var body []byte
				if Env == DEV {
					res.Header().Set("Content-Type", "text/html")
					body = []byte(fmt.Sprintf(panicHtml, err, err, stack))
				}

				res.WriteHeader(http.StatusInternalServerError)
				if nil != body {
					res.Write(body)
				}
			}
		}()

		c.Next()
	}
}

func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}

	name := []byte(fn.Name())
	if lastslash := bytes.LastIndex(name, slash); lastslash >= 0 {
		name = name[lastslash+1:]
	}

	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}

	name = bytes.Replace(name, conterDot, dot, -1)
	return name

}

func source(lines [][]byte, n int) []byte {
	n--
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}
