# sls-lambda-opa

Experiment embedding Open Policy Agent (https://www.openpolicyagent.org/) within a GoLang Api Gateway Authorizer Lambda function, deployed via Serverless framework.  See this for info on AWS Api Gateway Authorizer Lambdas: https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-use-lambda-authorizer.html

## Install
1. go https://golang.org/doc/install#install
2. dep https://github.com/golang/dep
3. serverless.com framework and configure for aws lambda https://serverless.com/framework/docs/providers/aws/guide/installation/
4. Pull this repo and `chmod 755 *.sh`

## Deploy
```
./slsDeploy.sh
```
This will compile the go files, and then deploy 3 Lambdas and associated API gateway.
- opacheck (the authorizer)
- gold target (accessible to gold role only (?role=gold)
- silver target (accessible to silver and gold roles)

## Test
Tail logs as below in one shell, and then use some of the following:

```
curl -H Authorization:allow https://[apiId].execute-api.ap-southeast-2.amazonaws.com/dev/gold?role=gold
curl -H Authorization:allow https://[apiId].execute-api.ap-southeast-2.amazonaws.com/dev/silver?role=gold
curl -H Authorization:allow https://[apiId].execute-api.ap-southeast-2.amazonaws.com/dev/gold?role=silver
```

The OPA policy in summary is:
- deny by default
- role 'gold' can access /silver and /gold
- role 'silver' can access /silver but not /gold
- other roles cannot access either

## Tail logs
```
./slsLogs opacheck
```
