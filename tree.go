package simple

import (
	"fmt"
	"github.com/Unknwon/com"
	"regexp"
	"strings"
)

const (
	_PATTERN_STATIC patternType = iota
	_PATTERN_REGEXP
	_PATTERN_PATH_EXT
	_PATTERN_HODLER
	_PATTERN_MATCH_ALL
)

type patternType int8

type Tree struct {
	parent *Tree

	typ        patternType
	pattern    string
	rawPattern string
	wildcards  []string
	reg        *regexp.Regexp

	subtrees []*Tree
	leaves   []*Leaf
}

func NewSubtree(parent *Tree, pattern string) *Tree {
	typ, rawPattern, wildcards, reg := checkPattern(pattern)
	return &Tree{parent, typ, pattern, rawPattern, wildcards, reg, make([]*Tree, 0, 5), make([]*Leaf, 0, 5)}
}

// 将所有的类型处理掉
func getRawPattern(rawPattern string) string {
	rawPattern = strings.Replace(rawPattern, ":int", "", -1)
	rawPattern = strings.Replace(rawPattern, ":string", "", -1)

	for {
		startIndex := strings.Index(rawPattern, "(")

		if startIndex == -1 {
			break
		}

		closeIndex := strings.Index(rawPattern, ")")
		if closeIndex > -1 {
			rawPattern = rawPattern[:startIndex] + rawPattern[closeIndex:]
		}
	}

	return rawPattern

}

// * 所有
// *.* 路径-后缀路由
// : 占位符
// https://go-macaron.com/docs/middlewares/routing
func checkPattern(pattern string) (typ patternType, rawPattern string, wildcards []string, reg *regexp.Regexp) {
	pattern = strings.TrimLeft(pattern, "?")
	rawPattern = getRawPattern(pattern)

	if pattern == "*" {
		typ = _PATTERN_MATCH_ALL
	} else if pattern == "*.*" {
		typ = _PATTERN_PATH_EXT
	} else if strings.Contains(pattern, ":") {
		typ = _PATTERN_REGEXP
		pattern, wildcards = getWilcards(pattern)
		if pattern == "(.+)" {
			typ = _PATTERN_HODLER
		} else {
			reg = regexp.MustCompile(pattern)
		}
	}
	return typ, rawPattern, wildcards, reg
}

// 循环遍历 找通配符
// 找到通配符 放在一起
func getWilcards(pattern string) (string, []string) {
	wildcards := make([]string, 0, 2)

	var wildcard string

	for {
		wildcard, pattern = getNextWildcard(pattern)

		if len(wildcard) > 0 {
			wildcards = append(wildcards, wildcard)
		} else {
			break
		}
	}

	return pattern, wildcards
}

var wildcardPattern = regexp.MustCompile(`:[a-zA-Z0-9]+`)

func isSpecialRegexp(pattern, regStr string, pos []int) bool {
	return len(pattern) >= pos[1]+len(regStr) && pattern[pos[1]:pos[1]+len(regStr)] == regStr
}

// 找个才是找通配符的 核心代码
// 如果没找到 要的 通配符。直接返回
//func (re *Regexp) FindStringIndex(s string) (loc []int)
// 找到的话 将通配符转化成 正则表达式
func getNextWildcard(pattern string) (wildcard, _ string) {
	pos := wildcardPattern.FindStringIndex(pattern)
	if pos == nil {
		return "", pattern
	}

	wildcard = pattern[pos[0]:pos[1]]

	if len(pattern) == pos[1] {
		return wildcard, strings.Replace(pattern, wildcard, `(.+)`, 1)
	} else if pattern[pos[1]] != '(' {
		switch {
		case isSpecialRegexp(pattern, ":int", pos):
			pattern = strings.Replace(pattern, ":int", "([0-9]+)", 1)
		case isSpecialRegexp(pattern, ":string", pos):
			pattern = strings.Replace(pattern, ":string", "([\\w]+)", 1)
		default:
			return wildcard, strings.Replace(pattern, wildcard, `(.+)`, 1)
		}
	}

	fmt.Println(wildcard)
	return wildcard, pattern[:pos[0]] + pattern[pos[1]:]
}

func NewTree() *Tree {
	return NewSubtree(nil, "")
}

func (t *Tree) Add(pattern string, handle Handle) *Leaf {
	pattern = strings.TrimSuffix(pattern, "/")
	return t.addNextSegment(pattern, handle)
}

func (t *Tree) addLeaf(pattern string, handle Handle) *Leaf {
	for i := 0; i < len(t.leaves); i++ {
		if t.leaves[i].pattern == pattern {
			return t.leaves[i]
		}
	}

	leaf := NewLeaf(t, pattern, handle)

	// 如果使用？ 那就是叶子节点会增加到上面的节点  树嘛，  父节点 当然是 父节点啦
	// /user/?:id 可同时匹配 /user/ 和 /user/123
	if leaf.optional {
		parent := leaf.parent
		if parent.parent != nil {
			parent.parent.addLeaf(parent.pattern, handle)
		} else {
			parent.addLeaf("", handle)
		}
	}

	i := 0
	for ; i < len(t.leaves); i++ {
		if leaf.typ < t.leaves[i].typ {
			break
		}
	}

	if i == len(t.leaves) {
		t.leaves = append(t.leaves, leaf)
	} else {
		t.leaves = append(t.leaves[:i], append([]*Leaf{leaf}, t.leaves[i:]...)...)
	}
	return leaf
}

func NewLeaf(parent *Tree, pattern string, handle Handle) *Leaf {
	typ, rawPattern, wildcards, reg := checkPattern(pattern)

	optional := false
	if len(pattern) > 0 && pattern[0] == '?' {
		optional = true
	}

	return &Leaf{parent, typ, pattern, rawPattern, wildcards, reg, optional, handle}
}

// 将多级路由拆分到子🌲中
// 如果它是根路由的话， 就创建叶子节点
// 如果是多级的话，创建子树
// 这样的好处是 当
/*
	s.Get("/", func() {})

	访问 /s/b 会访问当前的路由体
*/
func (t *Tree) addNextSegment(pattern string, handle Handle) *Leaf {
	pattern = strings.TrimPrefix(pattern, "/")
	i := strings.Index(pattern, "/")
	if i == -1 {
		return t.addLeaf(pattern, handle)
	}

	return t.addSubtree(pattern[:i], pattern[i+1:], handle)
}

// 创建子树
// 如果创建过 就返回
// 判断当前节点的所有子树，如果找到相同的节点 就使用当前节点 继续增加同级

// 如果不存在 创建子树节点，加到中间
// typ 顺序 是路由匹配的先后顺序
func (t *Tree) addSubtree(segment, pattern string, handle Handle) *Leaf {
	for i := 0; i < len(t.subtrees); i++ {
		if t.subtrees[i].pattern == segment {
			return t.subtrees[i].addNextSegment(pattern, handle)
		}
	}

	subtree := NewSubtree(t, segment)
	i := 0
	for ; i < len(t.subtrees); i++ {
		if subtree.typ < t.subtrees[i].typ {
			break
		}
	}

	if i == len(t.subtrees) {
		t.subtrees = append(t.subtrees, subtree)
	} else {
		t.subtrees = append(t.subtrees[:i], append([]*Tree{subtree}, t.subtrees[i:]...)...)
	}
	return subtree.addNextSegment(pattern, handle)
}

func (t *Tree) Match(url string) (Handle, Params, bool) {
	url = strings.TrimPrefix(url, "/")
	url = strings.TrimPrefix(url, "/")

	params := make(Params)
	handle, ok := t.matchNextSegment(0, url, params)
	return handle, params, ok
}

func (t *Tree) matchNextSegment(globLevel int, url string, params Params) (Handle, bool) {
	i := strings.Index(url, "/")
	if i == -1 {
		return t.matchLeaf(globLevel, url, params)
	}
	return t.matchSubtree(globLevel, url[:i], url[i+1:], params)
}

func (t *Tree) matchLeaf(globLevel int, url string, params Params) (Handle, bool) {
	url, err := PathUnescape(url)

	if err != nil {
		return nil, false
	}

	for i := 0; i < len(t.leaves); i++ {
		switch t.leaves[i].typ {
		case _PATTERN_STATIC:
			if t.leaves[i].pattern == url {
				return t.leaves[i].handle, true
			}
		case _PATTERN_REGEXP:
			results := t.leaves[i].reg.FindStringSubmatch(url)
			if len(results)-1 != len(t.leaves[i].wildcards) {
				break
			}

			for j := 0; j < len(t.leaves[i].wildcards); j++ {
				params[t.leaves[i].wildcards[j]] = results[j+1]
			}

			return t.leaves[i].handle, true
		case _PATTERN_PATH_EXT:
			j := strings.LastIndex(url, ".")
			if j > -1 {
				params[":path"] = url[:j]
				params[":ext"] = url[j+1:]
			} else {
				params[":path"] = url
			}
			return t.leaves[i].handle, true
		case _PATTERN_HODLER:
			params[t.leaves[i].wildcards[0]] = url
			return t.leaves[i].handle, true
		case _PATTERN_MATCH_ALL:
			params["*"] = url
			params["*"+com.ToStr(globLevel)] = url
			return t.leaves[i].handle, true
		}
	}
	return nil, false
}

func (t *Tree) matchSubtree(globLevel int, segment, url string, params Params) (Handle, bool) {
	unescapedSegment, err := PathUnescape(segment)
	if err != nil {
		return nil, false
	}

	for i := 0; i < len(t.subtrees); i++ {
		switch t.subtrees[i].typ {
		case _PATTERN_STATIC:
			if t.subtrees[i].pattern == unescapedSegment {
				if handle, ok := t.subtrees[i].matchNextSegment(globLevel, url, params); ok {
					return handle, true
				}
			}
		case _PATTERN_REGEXP:
			results := t.subtrees[i].reg.FindStringSubmatch(unescapedSegment)
			if len(results)-1 != len(t.subtrees[i].wildcards) {
				break
			}

			for j := 0; j < len(t.subtrees[i].wildcards); i++ {
				params[t.subtrees[i].wildcards[j]] = results[j+1]
			}

			if handle, ok := t.subtrees[i].matchNextSegment(globLevel, url, params); ok {
				return handle, true
			}
		case _PATTERN_HODLER:
			if handle, ok := t.subtrees[i].matchNextSegment(globLevel+1, url, params); ok {
				params[t.subtrees[i].wildcards[0]] = unescapedSegment
				return handle, true
			}
		case _PATTERN_MATCH_ALL:
			if handle, ok := t.subtrees[i].matchNextSegment(globLevel+1, url, params); ok {
				params["*"+com.ToStr(globLevel)] = unescapedSegment
				return handle, true
			}
		}
	}

	if len(t.leaves) > 0 {
		leaf := t.leaves[len(t.leaves)-1]
		unescapedURL, err := PathUnescape(segment + "/" + url)
		if err != nil {
			return nil, false
		}

		if leaf.typ == _PATTERN_PATH_EXT {
			j := strings.LastIndex(unescapedURL, ".")
			if j > -1 {
				params[":path"] = unescapedURL[:j]
				params[":ext"] = unescapedURL[j+1:]
			} else {
				params[":path"] = unescapedURL
			}

			return leaf.handle, true
		} else if leaf.typ == _PATTERN_MATCH_ALL {
			params["*"] = unescapedURL
			params["*"+com.ToStr(globLevel)] = unescapedURL
			return leaf.handle, true
		}
	}

	return nil, false

}

type Leaf struct {
	parent *Tree

	typ        patternType
	pattern    string
	rawPattern string
	wildcards  []string
	reg        *regexp.Regexp
	optional   bool

	handle Handle
}
