package BD

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	rrs "github.com/KolobokMysnoy/tmp/general/requestResponseStruct"
)

var (
	IpConnectMongo   = "db"
	PortConnectMongo = "27017"
	Login            = "admin"
	Password         = "password"
)

type RequestRepository struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty"`
	Scheme     string              `bson:"scheme"`
	Method     string              `bson:"method"`
	Host       string              `bson:"host"`
	Path       string              `bson:"path"`
	GetParams  map[string][]string `bson:"get_pa	rams"`
	Headers    http.Header         `bson:"headers"`
	Cookies    []http.Cookie       `bson:"cookies"`
	PostParams map[string][]string `bson:"post_params"`
	Timestamp  time.Time           `bson:"timestamp"`
}

type ResponseRepository struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Code      int                `bson:"code"`
	Message   string             `bson:"message"`
	Headers   http.Header        `bson:"headers"`
	Body      string             `bson:"body"`
	IdRequest primitive.ObjectID `bson:"request_id"`
	Timestamp time.Time          `bson:"timestamp"`
}

func createMongoDBClient() (*mongo.Client, error) {
	clientOptions := options.Client().
		ApplyURI("mongodb://" + Login + ":" + Password + "@" +
			IpConnectMongo + ":" + PortConnectMongo)

	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, err
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

type BD interface {
	SaveResponseRequest(rrs.Response, rrs.Request) error
	GetRequestByID(string) (rrs.Request, error)
	GetAllRequests() ([]rrs.Request, error)
}

type MongoDB struct {
}

func (m MongoDB) SaveResponseRequest(resp rrs.Response, req rrs.Request) error {
	client, err := createMongoDBClient()
	if err != nil {
		return err
	}
	defer client.Disconnect(context.Background())

	db := client.Database("http_logs")
	requestsCollection := db.Collection("requests")
	responsesCollection := db.Collection("responses")

	timeNow := time.Now()

	requestMongo := RequestRepository{
		Method:     req.Method,
		Scheme:     req.Scheme,
		Path:       req.Path,
		Host:       req.Host,
		GetParams:  req.GetParams,
		Headers:    req.Headers,
		Cookies:    req.Cookies,
		PostParams: req.PostParams,
		Timestamp:  timeNow,
	}

	responseMongo := ResponseRepository{
		Code:      resp.Code,
		Message:   resp.Message,
		Headers:   resp.Headers,
		Body:      resp.Body,
		Timestamp: timeNow,
	}

	idOfReq, err := requestsCollection.InsertOne(context.Background(), requestMongo)
	if err != nil {
		return err
	}

	responseMongo.IdRequest = idOfReq.InsertedID.(primitive.ObjectID)
	_, err = responsesCollection.InsertOne(context.Background(), responseMongo)
	if err != nil {
		return err
	}

	return nil
}

func (m MongoDB) GetRequestByID(id string) (rrs.Request, error) {
	client, err := createMongoDBClient()
	if err != nil {
		return rrs.Request{}, err
	}
	defer client.Disconnect(context.Background())

	db := client.Database("http_logs")
	requestsCollection := db.Collection("requests")

	var retrievedRequest RequestRepository
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return rrs.Request{}, err
	}

	err = requestsCollection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&retrievedRequest)
	if err != nil {
		return rrs.Request{}, err
	}

	return rrs.Request{
		Method:     retrievedRequest.Method,
		Path:       retrievedRequest.Path,
		Scheme:     retrievedRequest.Scheme,
		Host:       retrievedRequest.Host,
		GetParams:  retrievedRequest.GetParams,
		Headers:    retrievedRequest.Headers,
		Cookies:    retrievedRequest.Cookies,
		PostParams: retrievedRequest.PostParams,
	}, nil
}

func (m MongoDB) GetAllRequests() ([]rrs.Request, error) {
	client, err := createMongoDBClient()
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(context.Background())

	db := client.Database("http_logs")
	collection := db.Collection("requests")

	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var requests []rrs.Request
	for cursor.Next(context.Background()) {
		var request RequestRepository
		if err := cursor.Decode(&request); err != nil {
			return nil, err
		}
		requests = append(requests, rrs.Request{
			Method:     request.Method,
			Path:       request.Path,
			Scheme:     request.Scheme,
			Host:       request.Host,
			GetParams:  request.GetParams,
			Headers:    request.Headers,
			Cookies:    request.Cookies,
			PostParams: request.PostParams,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return requests, nil
}
