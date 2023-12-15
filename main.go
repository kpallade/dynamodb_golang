package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// The easiest solution is to create your local instance using the -sharedDb flag, then the access key and secret values don’t matter
func main() {

	dynamodbClient, err := newclient("localhost") // localhost = AWS PROFIL
	if err != nil {
		log.Fatal(err)
	}

	// example table name
	exampleTableName := "Movies"

	// create table
	tableInput := &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("year"),
				AttributeType: types.ScalarAttributeTypeN, // data type descriptor: N == number
			},
			{
				AttributeName: aws.String("title"),
				AttributeType: types.ScalarAttributeTypeS, // data type descriptor: S == string
			},
		},
		KeySchema: []types.KeySchemaElement{ // key: year + title
			{
				AttributeName: aws.String("year"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("title"),
				KeyType:       types.KeyTypeRange,
			},
		},
		TableName: aws.String(exampleTableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	err = createDynamoDBTable(dynamodbClient, exampleTableName, tableInput)
	if err != nil {
		log.Fatal(err)
	}
}

// newclient constructs a new dynamodb client using a default configuration
// and a provided profile name (created via aws configure cmd).
func newclient(profile string) (*dynamodb.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("localhost"),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: "http://localhost:8000"}, nil
			})),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID: "abcd", SecretAccessKey: "a1b2c3", SessionToken: "",
				Source: "Mock credentials used above for local instance",
			},
		}),
	)

	if err != nil {
		return nil, err
	}

	client := dynamodb.NewFromConfig(cfg)
	return client, nil
}

// createDynamoDBTable creates a table in the client's instance
// using the table parameters specified in input.
func createDynamoDBTable(c *dynamodb.Client, tableName string, input *dynamodb.CreateTableInput) error {
	var tableDesc *types.TableDescription
	table, err := c.CreateTable(context.TODO(), input)
	if err != nil {
		log.Printf("Failed to create table %v with error: %v\n", tableName, err)
	} else {
		waiter := dynamodb.NewTableExistsWaiter(c)
		err = waiter.Wait(context.TODO(), &dynamodb.DescribeTableInput{
			TableName: aws.String(tableName)}, 5*time.Minute)
		if err != nil {
			log.Printf("Failed to wait on create table %v with error: %v\n", tableName, err)
		}
		tableDesc = table.TableDescription
	}

	fmt.Println(tableDesc)

	return err
}
