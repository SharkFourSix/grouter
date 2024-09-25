package grouter

type RouterState int

const (
	// Read value as input data.
	//
	// This can be used to handle custom routing inside a route(d) handler.
	//
	// To enter this mode, use any of the UssdRequest.PromptXXX() functions.
	READ_INPUT RouterState = iota
	// Read value as option to for routing.
	//
	// This is the default mode where the routing engine uses the value read
	// to match route handlers.
	//
	// Use any of the UssdRequest.ContinueXXX() functions to enter this mode.
	READ_OPTION
)

// UssdRequest UssdRequest interface to access various states and data from
// the USSD request session.
type UssdRequest interface {
	// MSISDN MSISDN returns the mobile subscriber identification number
	// assigned to the user by their network.
	MSISDN() string
	// Option Option returns the value entererd by the user after calling any
	// of the .Continue functions.
	//
	// The value is intepreted as an option by the routing engine, which is
	// used to match handlers. There is almost no need for handlers to use
	// this value.
	Option() string
	// Input Input returns the value entered by the user after calling any of
	// the .Prompt functions. The value is passed as is to the handler.
	Input() string
	// returns the session associated with this request
	Session() UssdSession
	// Continue This function causes the next input to be treated as an option
	// that will be handled by the routing engine to match a handler
	Continue(text string, args ...any)
	// Prompt This function causes the next input to be treated as input data,
	// which can be obtained using the .Input() function.
	//
	// The original option is also maintained and can be obtained by calling
	// the .Option() function.
	Prompt(text string, args ...any)
	// PromptWithTemplate Prompt using content from a template.
	//
	// Refer to UssdRequest.Prompt() function for more
	PromptWithTemplate(tmplName string, values TemplateValues)
	// ContinueWithTemplate Continue using content from a template.
	//
	// Refer to UssdRequest.Continue() function for more
	ContinueWithTemplate(tmplName string, values TemplateValues)
	// End Ends the session
	End(text string, args ...any)
	EndWithTemplate(tmplName string, values TemplateValues)
	// SetAttribute Set a request attribute.
	//
	// Request attributes are only valid for the duration of the request.
	// To store data for the duration of the session, use UssdRequest.Session() function
	// to use the session storage.
	SetAttribute(key string, value any)

	// GetAttribute Get request attribute
	GetAttribute(key string) any
}
