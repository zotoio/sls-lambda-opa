package main

import (

	"bytes"
	"strings"
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/util"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/storage/inmem"
	"time"
	"log"
)

var module string
var store storage.Store
var compiler *ast.Compiler
var ctx context.Context

func main() {
	log.Println("cold start")

	// just compile OPA policies once per container
	compileOpaPolicy()

	//os.Exit(0)
	lambda.Start(Handler)
}

func track(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

func compileOpaPolicy() {
	defer track(time.Now(), "compileOpaPolicy()")

	ctx = context.Background()

	// Policy.  This could be pulled from opa server on cold start, or s3.
	module = `
		package rolemapper

		import data.roles

		default allow = false

		allow {
			input.operation = "GET"
			role_has_permission[input.role_name]
		}

		role_has_permission[role_name] {
			r = roles[_]
			r.name = role_name
			match_with_wildcard(r.operations, input.operation)
			match_with_wildcard(r.resources, input.resource)
		}
		match_with_wildcard(allowed, value) {
			allowed[_] = "*"
		}
		match_with_wildcard(allowed, value) {
			allowed[_] = value
		}
	`

	// Data store. This could be pulled from opa server on cold start, or s3.
	store = inmem.NewFromReader(bytes.NewBufferString(`{
		"roles": [
			{
				"resources": ["/gold", "/silver"],
				"operations": ["GET"],
				"name": "gold"
			},
			{
				"resources": ["/silver"],
				"operations": ["GET"],
				"name": "silver"
			},
			{
				"resources": ["*"],
				"operations": ["*"],
				"name": "admin"
			}
		]
	}`))

	// Parse the module. The first argument is used as an identifier in error messages.
	parsed, err := ast.ParseModule("rolemapper.rego", module)
	if err != nil {
		// Handle error.
	}

	// Create a new compiler and compile the module. The keys are used as
	// identifiers in error messages.
	compiler = ast.NewCompiler()
	compiler.Compile(map[string]*ast.Module{
		"rolemapper.rego": parsed,
	})

	if compiler.Failed() {
		// Handle error. Compilation errors are stored on the compiler.
		panic(compiler.Errors)
	}
}

func checkOpaPolicy(roleName string, resourcePath string, operation string) (result bool) {
	defer track(time.Now(), "checkOpaPolicy()")

	// Create a new query that uses the compiled policy from above.
	reg := rego.New(
		rego.Query("data.rolemapper.allow"),
		rego.Store(store),
		rego.Compiler(compiler),
		rego.Input(
			map[string]interface{}{
				"resource": resourcePath,
				"role_name": roleName,
				"operation": operation,
			},
		),
	)

	// Run evaluation.
	rs, err := reg.Eval(ctx)

	if err != nil {
		log.Println("err:", err)
	}

	// Inspect results.
	log.Println("len:", len(rs))

	if len(rs) != 1 {
		return false
	}

	result = util.Compare(rs[0].Expressions[0].Value, true) == 0

	log.Println("res:", result)

	return result

}

func Handler(event events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	defer track(time.Now(), "Handler()")
	log.Println("handle event")

	/**
		check Authorization header first - just using 'allow' and 'deny' value to simulate for now
		JWT validation can replace this to check signing/exp, and then auth caching can be enabled in apigw
	*/
	var headerCheckOk = false
	switch token := strings.ToLower(event.Headers["Authorization"]); token {
	case "allow":
		log.Println("Auth header forcing allow, on to OPA check..")
		headerCheckOk = true
	case "deny":
		log.Println("Auth header forcing deny")
		return generateIAMPolicy("user", "Deny", event.MethodArn), nil
	default:
		log.Println("Auth header invalid: ", token)
		return generateIAMPolicy("user", "Deny", event.MethodArn), nil
	}

	/**
		check request against OPA policy - force the role to either 'gold' or 'silver'
		gold = can access /gold and /silver
		silver = can access /silver but not /gold
		..any other role denied
		role would be taken from JWT claims once signing, expiry etc is verified
	*/
	roleName := event.QueryStringParameters["role"] // just using QS to test
	resourcePath := event.Path
	log.Println("roleName: ", roleName)
	log.Println("resourcePath: ", resourcePath)
	if headerCheckOk && checkOpaPolicy(roleName, resourcePath, event.HTTPMethod) {
		log.Println("OPA policy check ok - request allowed")
		return generateIAMPolicy("user", "Allow", event.MethodArn), nil
	} else {
		log.Println("failed OPA policy check")
		return generateIAMPolicy("user", "Deny", event.MethodArn), nil
	}

}

/**
	Generate IAM policy document
 */
func generateIAMPolicy(principalId string, effect string, resource string) events.APIGatewayCustomAuthorizerResponse {
	defer track(time.Now(), "generateIAMPolicy()")
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
