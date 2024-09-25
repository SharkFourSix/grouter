package at

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/SharkFourSix/grouter"
	cmap "github.com/orcaman/concurrent-map"
)

const RouterName = "AfricasTalkingUSSDRouter"

var (
	instance router
)

func init() {
	grouter.RegisterRouter(RouterName, &instance)
}

type router struct {
}

type requestData struct {
	Text        string
	SessionId   string
	ServiceCode string
	PhoneNumber string
	NetworkCode string
}

type binder func(r *requestData, value string)

var (
	binders map[string]binder = map[string]binder{
		"text":        func(r *requestData, value string) { r.Text = value },
		"sessionId":   func(r *requestData, value string) { r.SessionId = value },
		"serviceCode": func(r *requestData, value string) { r.ServiceCode = value },
		"phoneNumber": func(r *requestData, value string) { r.PhoneNumber = value },
		"networkCode": func(r *requestData, value string) { r.NetworkCode = value },
	}
)

func (r *router) CreateRequest(resp *grouter.BufferedResponse, req *http.Request, store grouter.Storage) (grouter.UssdRequest, error) {
	var (
		request = new(requestData)
		form    url.Values
	)
	if err := req.ParseForm(); err != nil {
		return nil, err
	}
	form = req.Form
	bind := func(name string, b binder) error {
		if form.Has(name) {
			b(request, form.Get(name))
			return nil
		} else {
			return fmt.Errorf("missing form field: `%s`", name)
		}
	}
	for field, Binder := range binders {
		if err := bind(field, Binder); err != nil {
			return nil, err
		}
	}

	sess := store.Get(request.SessionId)
	if grouter.IsEmptyText(request.Text) {
		// New session
		sess = &africasTalkingUssdSession{
			startTime:             time.Now(),
			readPointer:           0,
			store:                 cmap.New(),
			id:                    request.SessionId,
			state:                 grouter.READ_OPTION,
			autoAdjustReadPointer: false,
		}
		store.Set(request.SessionId, sess)
	} else {
		if sess == nil {
			return nil, fmt.Errorf("session %s not found", request.SessionId)
		} else {
			sess.(*africasTalkingUssdSession).Read(request)
		}
	}
	ussdRequest := ussd_request{
		resp: resp,
		data: request,
		req:  req,
		attr: map[string]any{},
		sess: sess.(*africasTalkingUssdSession),
	}
	return &ussdRequest, nil
}

type ussd_request struct {
	resp *grouter.BufferedResponse
	req  *http.Request
	data *requestData
	sess *africasTalkingUssdSession
	attr map[string]any
}

func (r *ussd_request) Session() grouter.UssdSession {
	return r.sess
}

func (r *ussd_request) MSISDN() string {
	return r.data.PhoneNumber
}

func (r *ussd_request) Option() string {
	return r.sess.option
}

func (r *ussd_request) Input() string {
	return r.sess.input
}

func (r *ussd_request) Continue(text string, args ...any) {
	r.sess.state = grouter.READ_OPTION
	_, _ = fmt.Fprintf(r.resp, "CON %s\n", fmt.Sprintf(text, args...))
}

func (r *ussd_request) ContinueWithTemplate(tmplName string, values grouter.TemplateValues) {
	r.sess.state = grouter.READ_OPTION
	r.resp.RenderContinueTemplate(tmplName, values)
}

func (r *ussd_request) Prompt(text string, args ...any) {
	r.sess.state = grouter.READ_INPUT
	_, _ = fmt.Fprintf(r.resp, "CON %s\n", fmt.Sprintf(text, args...))
}

func (r *ussd_request) PromptWithTemplate(tmplName string, values grouter.TemplateValues) {
	r.sess.state = grouter.READ_INPUT
	r.resp.RenderContinueTemplate(tmplName, values)
}

func (r *ussd_request) End(text string, args ...any) {
	_, _ = fmt.Fprintf(r.resp, "END %s\n", fmt.Sprintf(text, args...))
}

func (r *ussd_request) EndWithTemplate(tmplName string, values grouter.TemplateValues) {
	r.resp.RenderEndTemplate(tmplName, values)
}

func (r *ussd_request) SetAttribute(key string, value any) {
	r.attr[key] = value
}

func (r *ussd_request) GetAttribute(key string) any {
	return r.attr[key]
}
