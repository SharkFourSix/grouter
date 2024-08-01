package grouter

type RouterState int

const (
	// Read value as input data
	READ_INPUT RouterState = iota
	// Read value as option to for routing
	READ_OPTION
)

type UssdRequest interface {
	MSISDN() string
	Option() string
	Input() string
	// returns the session associated with this request
	Session() UssdSession
	// Continues the session
	Continue(text string, args ...any)
	Prompt(text string, args ...any)
	PromptWithTemplate(tmplName string, values TemplateValues)
	ContinueWithTemplate(tmplName string, values TemplateValues)
	// Ends the session
	End(text string, args ...any)
	EndWithTemplate(tmplName string, values TemplateValues)
	// Return or sets the request's IO state
	// Set a request attribute
	SetAttribute(key string, value any)

	// Get request attribute
	GetAttribute(key string) any
}
