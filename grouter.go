package grouter

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"slices"
	"strings"
	"sync"
	"text/template"
	"time"
)

type Logger interface {
	Printf(string, ...any)
}

// The main routing engine (Layer 0 router)
type Engine struct {
	Log              Logger
	Debug            bool
	NotFound         RouteHandler
	router           UssdRouter
	options          []*MenuOption
	Storage          Storage
	ihandler         int
	templateMap      map[string]*template.Template
	stateCache       *stateCache
	indexScreen      string
	storageFrequency time.Duration
	storageEviction  time.Duration
}

func (e Engine) currentHandler() string {
	if e.ihandler >= 0 {
		h := e.options[e.ihandler]
		return fmt.Sprintf("handler(options=%s,name=%s,ptr=%v)", h.code, h.name, h.handler)
	} else {
		return "handler(option=,name=)"
	}
}

type MenuOption struct {
	code         string
	handler      RouteHandler
	name         string
	sub          []*MenuOption
	parentScreen string
}

// Responsible for creating USSD requests
type UssdRouter interface {
	// Creates parses the incoming http request and creates a USSD request from it.
	//
	// # Session Management
	//
	// The router is responsible for providing and attaching a stage. The storage parameter
	// can be used to store or retrieve the session.
	//
	// # Returning responses
	//
	//	resp *BufferedResponse
	// should be used to buffer output from the handlers, through the UssdRequest interface.
	//
	// The routing engine utilizes this to create a final response back to the client.
	CreateRequest(resp *BufferedResponse, req *http.Request, storage Storage) (UssdRequest, error)
}

// USSD Request handler. Return true to remain in the same screen context
// or false to indicate to the routing engine to advance the context.
//
// A screen context confines menu options to a set, allowing the same option
// values to be used without conflicts.
type RouteHandler func(request UssdRequest) bool

type RouterOption func(r *Engine) error

func NewRouterEngine(options ...RouterOption) (*Engine, error) {
	r := Engine{
		Log:              &defaultLogger{shutup: true},
		storageFrequency: 30 * time.Second,
		storageEviction:  2 * time.Minute,
		NotFound: func(request UssdRequest) bool {
			request.End("Invalid option")
			return false
		},
		stateCache:  newStateCache(30*time.Second, 2*time.Minute),
		templateMap: map[string]*template.Template{},
	}
	for _, opt := range options {
		if err := opt(&r); err != nil {
			return nil, err
		}
	}
	r.Storage = NewInMemorySessionStorage(r.storageFrequency, r.storageEviction)
	if r.router == nil {
		return nil, ErrRouterNotFound
	}
	return &r, nil
}

func (e *Engine) mapOption(opt *MenuOption, parent *MenuOption) {
	if opt != nil {
		if parent != nil {
			opt.parentScreen = parent.name
		}
		e.options = append(e.options, opt)
		for _, subOpt := range opt.sub {
			e.mapOption(subOpt, opt)
		}
	}
}

func (e *Engine) MenuOptions(opts ...*MenuOption) {
	// create map
	for _, opt := range opts {
		if opt.code == "" {
			if e.indexScreen != "" {
				panic(fmt.Errorf("index screen is already set to `%s`", e.indexScreen))
			} else {
				e.indexScreen = opt.name
			}
		}
		e.mapOption(opt, nil)
	}
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	e.RouteFromHttpRequest(w, req)
}

func (e *Engine) RouteFromHttpRequest(w http.ResponseWriter, req *http.Request) {
	end := func(text string) {
		w.WriteHeader(200)
		_, _ = fmt.Fprintf(w, "END %s\n", text)
	}
	defer func() {
		if p := recover(); p != nil {
			e.Log.Printf("error: %v. handler info : %s", p, e.currentHandler())
			end("Session terminated due to internal error")
		}
	}()
	var writer BufferedResponse
	request, err := e.router.CreateRequest(&writer, req, e.Storage)
	if err != nil {
		e.Log.Printf("error creating request: %v", err)
		end("Session closed")
		return
	} else {
		// get current screen
		screen, _ := e.stateCache.get(request.Session().ID())
		index := slices.IndexFunc(e.options, func(mo *MenuOption) bool {
			return mo.code == request.Option() && mo.parentScreen == screen
		})
		e.Log.Printf("screen=%s, index=%d, option=%s, input=%s", screen, index, request.Option(), request.Input())
		if index != -1 {
			e.ihandler = index
			opt := e.options[index]
			e.Log.Printf("matched-handler=%s", opt.name)
			if !opt.handler(request) {
				e.stateCache.set(request.Session().ID(), opt.name)
			}
		} else {
			e.NotFound(request)
		}
		if writer.buf.Len() == 0 && IsEmptyText(writer.templateName) {
			e.Log.Printf("session ended because there was no response from handler `%s`. Make sure to call request.EndXXX or ContinueXXX", e.currentHandler())
			end("Unexpected end of session")
		} else {
			if !IsEmptyText(writer.templateName) {
				if tmpl, ok := e.templateMap[writer.templateName]; !ok {
					panic(fmt.Errorf("%s: template not found `%s`", e.currentHandler(), writer.templateName))
				} else {
					if writer.end {
						writer.Printf("END ")
					} else {
						writer.Printf("CON ")
					}
					err := tmpl.Execute(&writer, writer.values)
					if err != nil {
						e.Log.Printf(err.Error())
						panic(err)
					}
				}
			}
			_, err = w.Write(writer.buf.Bytes())
			if err != nil {
				e.Log.Printf(err.Error())
			}
		}
	}
}

func NewMenuOption(code string, h RouteHandler, name string, sub ...*MenuOption) *MenuOption {
	if IsEmptyText(name) {
		panic(fmt.Errorf("option: name cannot be blank"))
	}
	return &MenuOption{code: code, handler: h, name: name, sub: sub}
}

var (
	WithSessionTimes = func(probeFrequency, timeToEviction time.Duration) RouterOption {
		return func(r *Engine) error {
			r.storageFrequency = probeFrequency
			r.storageEviction = timeToEviction
			return nil
		}
	}
	DebugMode = func(r *Engine) error {
		r.Debug = true
		switch v := r.Log.(type) {
		case *defaultLogger:
			v.shutup = false
		}
		return nil
	}
	WithRouter = func(routerName string) RouterOption {
		return func(r *Engine) error {
			if instance, ok := registry.Load(routerName); ok {
				r.router = instance.(UssdRouter)
			} else {
				return ErrRouterNotFound
			}
			return nil
		}
	}

	WithTemplateFS = func(fsys fs.FS, root string, funcs template.FuncMap) RouterOption {
		return func(r *Engine) error {
			err := fs.WalkDir(fsys, root, func(filepath string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				ext := path.Ext(strings.ToLower(d.Name()))
				if !d.IsDir() && ext == ".tmpl" && len(d.Name()) >= 6 {
					fd, err := fsys.Open(path.Join(root, filepath))
					if err != nil {
						return err
					}
					defer fd.Close()
					b, err := io.ReadAll(fd)
					if err != nil {
						return err
					}
					templateName := strings.TrimPrefix(path.Join(root, filepath), root)
					tmpl, err := template.New(d.Name()).Funcs(funcs).Parse(string(b))
					if err != nil {
						return err
					}
					r.templateMap[templateName] = tmpl
				}
				return nil
			})
			return err
		}
	}
)

type defaultLogger struct {
	shutup bool
}

func (l defaultLogger) Printf(format string, args ...any) {
	if !l.shutup {
		fmt.Printf("[%s]: %s", time.Now().UTC().Format(time.RFC3339), fmt.Sprintf(format, args...))
		fmt.Println()
	}
}

var (
	registry sync.Map
)

// Registers a router. Must be called in package `init`
func RegisterRouter(name string, router UssdRouter) {
	registry.LoadOrStore(name, router)
}
