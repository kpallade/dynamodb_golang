package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynoObject represents an object in dynamoDB.
// Used to represent key value data such as keys, table items...
type DynoNotation map[string]types.AttributeValue

// Movie represents our domain entity.
type Movie struct {
	Year       int
	Title      string
	Phase      string
	HasFavreau bool
}

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

	// add one item
	movies := getMovieList()
	err = putItem(dynamodbClient, exampleTableName, movies[0])
	if err != nil {
		log.Fatal(err)
	}
	// add multiple items
	err = putItems(dynamodbClient, exampleTableName, movies[1:])
	if err != nil {
		log.Fatal(err)
	}

	// get item
	movieTitle := "Avengers: Endgame"
	movieYear := 2019
	titleAttr, _ := attributevalue.Marshal(movieTitle)
	yearAttr, _ := attributevalue.Marshal(movieYear)

	item, err := getItem(dynamodbClient, exampleTableName, DynoNotation{"title": titleAttr, "year": yearAttr})
	if err != nil {
		log.Fatal(err)
	}

	var movie Movie
	// unmarshal item
	err = attributevalue.UnmarshalMap(item, &movie)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("GET ITEM: Search for movie title `%s` from `%d` returned %v\n\n", movieTitle, movieYear, movie)

	// range query
	keyExpr := expression.Key("year").Equal(expression.Value(yearAttr))
	expr, err := expression.NewBuilder().WithKeyCondition(keyExpr).Build()
	if err != nil {
		log.Fatal(err)
	}
	query, err := dynamodbClient.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:                 aws.String(exampleTableName),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
	})
	if err != nil {
		log.Fatal(err)
	}

	returnMovies := []Movie{}
	// unmarshal list of items
	err = attributevalue.UnmarshalListOfMaps(query.Items, &returnMovies)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("RANGE QUERY: Search for movies by `%v returned %v\n\n", movieYear, returnMovies)

	// scan query
	hasFav := false
	hasFavAttr, _ := attributevalue.Marshal(hasFav) // flip this to true or false

	params := &dynamodb.ScanInput{
		TableName:                aws.String("Movies"),
		ProjectionExpression:     nil, // not provided, query will return all attributes
		ExpressionAttributeNames: nil,
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":hasFav": hasFavAttr,
		},
		FilterExpression: aws.String("hasFavreau = :hasFav"),
	}
	scan, err := dynamodbClient.Scan(context.TODO(), params)
	if err != nil {
		log.Fatal(err)
	}
	err = attributevalue.UnmarshalListOfMaps(scan.Items, &returnMovies)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("SCAN QUERY: Search for movies by `hasFavreau==%v` returned %v\n\n", hasFav, returnMovies)

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

// putItem inserts an item (key + attributes) in to a dynamodb table.
func putItem(c *dynamodb.Client, tableName string, item DynoNotation) (err error) {
	_, err = c.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(tableName), Item: item,
	})
	if err != nil {
		return err
	}

	return nil
}

// putItems batch inserts multiple items in to a dynamodb table.
func putItems(c *dynamodb.Client, tableName string, items []DynoNotation) (err error) {
	// dynamodb batch limit is 25
	if len(items) > 25 {
		return fmt.Errorf("Max batch size is 25, attempted `%d`", len(items))
	}

	// create requests
	writeRequests := make([]types.WriteRequest, len(items))
	for i, item := range items {
		writeRequests[i] = types.WriteRequest{PutRequest: &types.PutRequest{Item: item}}
	}

	// write batch
	_, err = c.BatchWriteItem(
		context.TODO(),
		&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{tableName: writeRequests},
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func getMovieList() (movies []DynoNotation) {
	list := []struct {
		year       int
		title      string
		phase      string // new movie attribute
		hasFavreau bool   // new movie attribute
	}{
		{year: 2008, phase: "I", hasFavreau: true, title: "Iron Man"},
		{year: 2008, phase: "I", hasFavreau: false, title: "The Incredible Hulk"},
		{year: 2010, phase: "I", hasFavreau: true, title: "Iron Man 2"},
		{year: 2011, phase: "I", hasFavreau: false, title: "Thor"},
		{year: 2011, phase: "I", hasFavreau: false, title: "Captain America: The First Avenger"},
		{year: 2012, phase: "I", hasFavreau: false, title: "Marvel's The Avengers"},
		{year: 2013, phase: "II", hasFavreau: true, title: "Iron Man 3"},
		{year: 2008, phase: "II", hasFavreau: false, title: "Thor: The Dark World"},
		{year: 2013, phase: "II", hasFavreau: false, title: "Captain America: The Winter Soldier"},
		{year: 2014, phase: "II", hasFavreau: false, title: "Guardians of the Galaxy"},
		{year: 2014, phase: "II", hasFavreau: false, title: "Avengers: Age of Ultron"},
		{year: 2015, phase: "II", hasFavreau: false, title: "Ant-Man"},
		{year: 2016, phase: "III", hasFavreau: false, title: "Captain America: Civil War"},
		{year: 2016, phase: "III", hasFavreau: false, title: "Doctor Strange"},
		{year: 2017, phase: "III", hasFavreau: false, title: "Guardians of the Galaxy Vol. 2"},
		{year: 2017, phase: "III", hasFavreau: true, title: "Spider-Man: Homecoming"},
		{year: 2017, phase: "III", hasFavreau: false, title: "Thor: Ragnarok"},
		{year: 2018, phase: "III", hasFavreau: false, title: "Black Panther"},
		{year: 2018, phase: "III", hasFavreau: true, title: "Avengers: Infinity War"},
		{year: 2018, phase: "III", hasFavreau: false, title: "Ant-Man and the Wasp"},
		{year: 2019, phase: "III", hasFavreau: false, title: "Captian Marvel"},
		{year: 2019, phase: "III", hasFavreau: false, title: "Avengers: Endgame"},
		{year: 2019, phase: "III", hasFavreau: true, title: "Spider-Man: Far From Home"},
	}

	for _, m := range list {
		movies = append(movies, DynoNotation{
			"year":       unsafeToAttrValue(m.year),
			"title":      unsafeToAttrValue(m.title),
			"phase":      unsafeToAttrValue(m.phase),
			"hasFavreau": unsafeToAttrValue(m.hasFavreau),
		})
	}

	return movies
}

func unsafeToAttrValue(in interface{}) types.AttributeValue {
	val, err := attributevalue.Marshal(in)
	if err != nil {
		log.Fatalf("could not marshal value `%v` with error: %v", in, err)
	}

	return val
}

// getItem returns an item if found based on the key provided.
// the key could be either a primary or composite key and values map.
func getItem(c *dynamodb.Client, tableName string, key DynoNotation) (item DynoNotation, err error) {
	resp, err := c.GetItem(context.TODO(), &dynamodb.GetItemInput{Key: key, TableName: aws.String(tableName)})
	if err != nil {
		return nil, err
	}

	return resp.Item, nil
}
