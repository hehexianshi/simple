package simple

import (
	"encoding/base64"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

type StaticOptions struct {
	Prefix      string
	SkipLogging bool
	IndexFile   string
	Expires     func() string
	ETag        bool
	FileSystem  http.FileSystem
}

type staticFileSystem struct {
	dir *http.Dir
}

func Static(directory string, staticOpt ...StaticOptions) Handler {
	opt := prepareStaticOptions(directory, staticOpt)
	return func(ctx *Context, log *log.Logger) {
		staticHandler(ctx, log, opt)
	}
}

func prepareStaticOptions(dir string, options []StaticOptions) StaticOptions {
	var opt StaticOptions

	if len(options) > 0 {
		opt = options[0]
	}

	return prepareStaticOption(dir, opt)
}

func prepareStaticOption(dir string, opt StaticOptions) StaticOptions {
	if len(opt.IndexFile) == 0 {
		opt.IndexFile = "index.html"
	}

	if opt.Prefix != "" {
		if opt.Prefix[0] != '/' {
			opt.Prefix = "/" + opt.Prefix
		}

		opt.Prefix = strings.TrimRight(opt.Prefix, "/")
	}

	if opt.FileSystem == nil {
		opt.FileSystem = newStaticFileSystem(dir)
	}
	return opt
}

type staticMap struct {
	lock sync.RWMutex
	data map[string]*http.Dir
}

func (sm *staticMap) Set(dir *http.Dir) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	sm.data[string(*dir)] = dir
}

func (sm *staticMap) Get(name string) *http.Dir {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	return sm.data[name]
}

func (sm *staticMap) Delete(name string) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	delete(sm.data, name)
}

var statics = staticMap{sync.RWMutex{}, map[string]*http.Dir{}}

func newStaticFileSystem(directory string) staticFileSystem {
	if !filepath.IsAbs(directory) {
		directory = filepath.Join(Root, directory)
	}

	dir := http.Dir(directory)
	statics.Set(&dir)
	return staticFileSystem{&dir}
}

func (fs staticFileSystem) Open(name string) (http.File, error) {
	return fs.dir.Open(name)
}

func staticHandler(ctx *Context, log *log.Logger, opt StaticOptions) bool {
	if ctx.Req.Method != "GET" && ctx.Req.Method != "HEAD" {
		return false
	}

	file := ctx.Req.URL.Path

	if opt.Prefix != "" {
		if !strings.HasPrefix(file, opt.Prefix) {
			return false
		}
		file = file[len(opt.Prefix):]
		if file != "" && file[0] != '/' {
			return false
		}
	}

	f, err := opt.FileSystem.Open(file)
	if err != nil {
		return false
	}
	defer f.Close()
	fi, err := f.Stat()

	if err != nil {
		return true
	}

	if fi.IsDir() {
		if !strings.HasSuffix(ctx.Req.URL.Path, "/") {
			http.Redirect(ctx.Resp, ctx.Req.Request, ctx.Req.URL.Path+"/", http.StatusFound)
			return true
		}

		file = path.Join(file, opt.IndexFile)
		f, err = opt.FileSystem.Open(file)

		if err != nil {
			return false
		}

		defer f.Close()
		fi, err = f.Stat()
		if err != nil || fi.IsDir() {
			return true
		}
	}

	if !opt.SkipLogging {
		log.Println("[static] seving " + file)
	}

	if opt.Expires != nil {
		ctx.Resp.Header().Set("Expires", opt.Expires())
	}

	if opt.ETag {
		tag := GenerateETag(string(fi.Size()), fi.Name(), fi.ModTime().UTC().Format(http.TimeFormat))
		ctx.Resp.Header().Set("ETag", tag)
	}

	http.ServeContent(ctx.Resp, ctx.Req.Request, file, fi.ModTime(), f)

	return true

}

func GenerateETag(fileSize, fileName, modTime string) string {
	etag := fileSize + fileName + modTime
	return base64.StdEncoding.EncodeToString([]byte(etag))
}
