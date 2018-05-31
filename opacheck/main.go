package main

import (

	"fmt"
	"strings"
	"errors"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/events"
	"github.com/open-policy-agent/opa/rego"
	"os"
	"context"
	"github.com/open-policy-agent/opa/ast"
)

func main() {
	fmt.Println("cold start")

	ctx := context.Background()

	// Define a simple policy.
	module := `
		package example
		default allow = false
		allow {
			input.identity = "admin"
		}
		allow {
			input.method = "GET"
		}
	`

	// Parse the module. The first argument is used as an identifier in error messages.
	parsed, err := ast.ParseModule("example.rego", module)
	if err != nil {
		// Handle error.
	}

	// Create a new compiler and compile the module. The keys are used as
	// identifiers in error messages.
	compiler := ast.NewCompiler()
	compiler.Compile(map[string]*ast.Module{
		"example.rego": parsed,
	})

	if compiler.Failed() {
		// Handle error. Compilation errors are stored on the compiler.
		panic(compiler.Errors)
	}

	// Create a new query that uses the compiled policy from above.
	rego := rego.New(
		rego.Query("data.example.allow"),
		rego.Compiler(compiler),
		rego.Input(
			map[string]interface{}{
				"identity": "bob",
				"method":   "GET",
			},
		),
	)

	// Run evaluation.
	rs, err := rego.Eval(ctx)

	if err != nil {
		// Handle error.
	}

	// Inspect results.
	fmt.Println("len:", len(rs))
	fmt.Println("value:", rs[0].Expressions[0].Value)

	os.Exit(0)

	lambda.Start(Handler)
}

func Handler(event events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {

	fmt.Println("handle event")

	switch token := strings.ToLower(event.Headers["Authorization"]); token {
	case "allow":
		return generatePolicy("user", "Allow", event.MethodArn), nil
	case "deny":
		return generatePolicy("user", "Deny", event.MethodArn), nil
	case "unauthorized":
		return events.APIGatewayCustomAuthorizerResponse{}, errors.New("Unauthorized")
	default:
		return events.APIGatewayCustomAuthorizerResponse{}, errors.New("Error: Invalid token")
	}
}

/**
	Generate IAM policy document
 */
func generatePolicy(principalId string, effect string, resource string) events.APIGatewayCustomAuthorizerResponse {
	authResponse := events.APIGatewayCustomAuthorizerResponse{PrincipalID: principalId}

	if effect != "" && resource != "" {
		authResponse.PolicyDocument = events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   effect,
					Resource: []string{resource},
				},
			},
		}
	}

	authResponse.Context = map[string]interface{}{}
	return authResponse
}
