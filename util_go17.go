package simple

import "net/url"

// PathUnescape unescapes a path. Ideally, this function would use
// url.PathUnescape(..), but the function was not introduced until go1.8.
func PathUnescape(s string) (string, error) {
	return url.QueryUnescape(s)
}
