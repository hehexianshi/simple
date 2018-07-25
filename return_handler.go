package simple

import (
	"net/http"
	"reflect"

	"github.com/go-macaron/inject"
)

type ReturnHandler func(*Context, []reflect.Value)

func canDeref(val reflect.Value) bool {
	return val.Kind() == reflect.Interface || val.Kind() == reflect.Ptr
}

func isError(val reflect.Value) bool {
	_, ok := val.Interface().(error)
	return ok
}

func isByteSlice(val reflect.Value) bool {
	return val.Kind() == reflect.Slice && val.Type().Elem().Kind() == reflect.Uint8
}

// 返回输出 数据。所有的return 走这
func defaultReturnHandler() ReturnHandler {
	return func(ctx *Context, vals []reflect.Value) {
		rv := ctx.GetVal(inject.InterfaceOf((*http.ResponseWriter)(nil)))
		resp := rv.Interface().(http.ResponseWriter)
		var respVal reflect.Value
		if len(vals) > 1 && vals[0].Kind() == reflect.Int {
			resp.WriteHeader(int(vals[0].Int()))
			respVal = vals[1]
		} else if len(vals) > 0 {
			respVal = vals[0]

			if isError(respVal) {
				err := respVal.Interface().(error)
				if err != nil {
					ctx.internalServerError(ctx, err)
				}
				return
			} else if canDeref(respVal) {
				if respVal.IsNil() {
					return
				}
			}
		}

		if canDeref(respVal) {
			respVal = respVal.Elem()
		}

		if isByteSlice(respVal) {
			resp.Write(respVal.Bytes())
		} else {
			resp.Write([]byte(respVal.String()))
		}
	}
}
