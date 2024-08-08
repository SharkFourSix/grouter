# grouter
An elegant USSD router in go

### Routers

A router sits between the routing engine and the end function. The job of a router
is to create requests from the incoming web request and provide session management.

The routing engine already takes care of managing the different states that the end
application might have. 

Additional routers can be found in [routers](routers/).

The router must implement the `grouter.Router` interface and provide implementations 
for `grouter.UssdRequest`, and `grouter.UssdSession` interfaces.

### Templating support

The library also supports template usage with custom function bindings.

### Usage

```go
go get github.com/SharkFourSix/grouter
```

### Example

Example [grouter_test.go](grouter_test.go)

```go
package grouter_test

import (
	"net/http"
	"os"
	"testing"
	"text/template"

	"github.com/SharkFourSix/grouter"
	"github.com/SharkFourSix/grouter/routers/at" // include africastalking implementation
)

func TestMain(t *testing.T) {
	e, err := grouter.NewRouterEngine(
		grouter.DebugMode,
		grouter.WithRouter(at.RouterName),
		grouter.WithTemplateFS(os.DirFS("./testdata/templates"), ".", template.FuncMap{}),
	)
	if err != nil {
		t.Fatal(err)
		return
	}
	e.MenuOptions(
		grouter.NewMenuOption("", welcomeScreen, "welcomeScreen",
			grouter.NewMenuOption("1", showAccount, "accountMenu",
				grouter.NewMenuOption("1", accountBalance, "accountBalance",
					grouter.NewMenuOption("#", showAccount, "accountMenu"),
				),
				grouter.NewMenuOption("2", miniStatement, "miniStatement",
					grouter.NewMenuOption("#", showAccount, "accountMenu"),
				),
				grouter.NewMenuOption("3", makeTransfer, "makeTransfer",
					grouter.NewMenuOption("#", showAccount, "accountMenu"),
				),
				grouter.NewMenuOption("#", welcomeScreen, "welcomeScreen"),
			),
			grouter.NewMenuOption("#", endSession, "endSession"),
		),
	)
	http.Handle("/ussd", e)
	http.ListenAndServe(":1234", nil)
}

type transferStep int

const (
	ReadAccount transferStep = iota
	ReadAmount
	ConfirmTransfer
)

func makeTransfer(req grouter.UssdRequest) bool {
	var step transferStep
	if sif, ok := req.Session().Get("transferStep"); ok {
		step = sif.(transferStep)
	} else {
		step = ReadAccount
	}
	switch step {
	case ReadAccount:
		if req.Input() == "" {
			req.Prompt("Enter recipient account number")
		} else {
			req.Session().Set("transferAccount", req.Input())
			req.Session().Set("transferStep", ReadAmount)
			req.Prompt("Enter amount to transfer")
		}
	case ReadAmount:
		if req.Input() == "" {
			req.Prompt("Enter amount to transfer")
		} else {
			req.Session().Set("transferAmount", req.Input())
			req.Session().Set("transferStep", ConfirmTransfer)
			req.PromptWithTemplate(
				"transfer/confirm.tmpl", grouter.TemplateValues{
					"Account": req.Session().MustGet("transferAccount"),
					"Amount":  req.Session().MustGet("transferAmount"),
				},
			)
		}
	case ConfirmTransfer:
		if req.Input() == "" {
			req.PromptWithTemplate(
				"transfer/confirm.tmpl", grouter.TemplateValues{
					"Account": req.Session().MustGet("transferAccount"),
					"Amount":  req.Session().MustGet("transferAmount"),
				},
			)
		} else {
			switch req.Input() {
			case "1":
				req.End(
					"🤙 You transferred %s to %s.",
					req.Session().MustGet("transferAmount"),
					req.Session().MustGet("transferAccount"),
				)
			case "#":
				req.End("Transfer cancelled. Thank you. Come again")
			case "3":
				req.End("You entered a wrong option! Pay attention")
			}
		}
	}
	return true // retain the scrren context for the duration of the interaction
}

func showAccount(req grouter.UssdRequest) bool {
	req.ContinueWithTemplate("account.tmpl", grouter.TemplateValues{"Phone": req.MSISDN()})
	return false
}

func accountBalance(req grouter.UssdRequest) bool {
	req.ContinueWithTemplate("balance.tmpl", grouter.TemplateValues{"Phone": req.MSISDN()})
	return false
}

func miniStatement(req grouter.UssdRequest) bool {
	req.ContinueWithTemplate("statement.tmpl", grouter.TemplateValues{"Phone": req.MSISDN()})
	return false
}

func welcomeScreen(req grouter.UssdRequest) bool {
	req.ContinueWithTemplate("main.tmpl", grouter.TemplateValues{"Phone": req.MSISDN()})
	return false
}

func endSession(req grouter.UssdRequest) bool {
	req.End("Thank you %s. Please come again!", req.MSISDN())
	return false
}

```

You can test the above code with the following USSD simulators

1. https://developers.africastalking.com/simulator
2. https://play.google.com/store/apps/details?id=com.africastalking.sandbox&hl=en
3. https://github.com/nndi-oss/dialoguss