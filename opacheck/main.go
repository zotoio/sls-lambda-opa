package main

import (

	"fmt"
	"strings"
	"errors"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/events"
	//"github.com/open-policy-agent/opa/rego"
)

func main() {
	fmt.Println("cold start")
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
