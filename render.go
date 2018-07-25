package simple

import (
	"net/http"
)

type Render interface {
	http.ResponseWriter
	SetResponseWrite(http.ResponseWriter)

	JSON(int, interface{})
}

func renderNotRegistered() {
	panic("middleware render hasn't been registered")
}

type DummyRender struct {
	http.ResponseWriter
}

func (r *DummyRender) JSON(int, interface{}) {
	renderNotRegistered()
}

func (r *DummyRender) SetResponseWrite(http.ResponseWriter) {
	renderNotRegistered()
}
