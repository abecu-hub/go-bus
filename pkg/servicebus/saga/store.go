package saga

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type Saga struct {
	ID            primitive.ObjectID     `bson:"_id"`
	CorrelationID string                 `bson:"correlationId"`
	Type          string                 `bson:"type"`
	State         map[string]interface{} `bson:"state"`
	IsCompleted   bool                   `bson:"isCompleted"`
	LockTime      *time.Time             `bson:"lockTime,omitempty"`
}

type Session interface {
	Commit() error
	Close()
}

type MongoSession struct {
	mongo.Session
	context.Context
}

func (session *MongoSession) Commit() error {
	err := session.CommitTransaction(session.Context)
	if err != nil {
		return err
	}
	return nil
}

func (session *MongoSession) Close() {
	session.Session.EndSession(session.Context)
}

func CreateMongoSession(ctx context.Context, session mongo.Session) Session {
	return &MongoSession{session, ctx}
}

type Store interface {
	SagaExists(correlationId string, sagaType string) (bool, error)
	RequestSaga(correlationId string, sagaType string) (*Context, error)
	CreateSaga(correlationId string, sagaType string) error
	UpdateState(session Session, correlationId string, sagaType string, state map[string]interface{}) error
	CompleteSaga(session Session, correlationId string, sagaType string) error
	DeleteSaga(correlationId string, sagaType string) error
}

type MongoStore struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func (store *MongoStore) ensureCompoundIndex() error {

	index := mongo.IndexModel{
		Keys:    bson.D{{"correlationId", 1}, {"type", 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := store.collection.Indexes().CreateOne(context.Background(), index)
	if err != nil {
		return err
	}
	return nil
}

func CreateMongoStore(client *mongo.Client, database string, collection string) (Store, error) {
	store := &MongoStore{
		client:     client,
		collection: client.Database(database).Collection(collection),
	}
	err := store.ensureCompoundIndex()
	if err != nil {
		return nil, err
	}
	return store, nil
}

func (store *MongoStore) SagaExists(correlationId string, sagaType string) (bool, error) {
	result := store.collection.FindOne(context.Background(), bson.M{"correlationId": correlationId, "type": sagaType})
	if result.Err() == nil {
		return true, nil
	}

	if result.Err() == mongo.ErrNoDocuments {
		return false, nil
	}
	return false, result.Err()
}

//Request a saga from the MongoDB store and put a transaction lock on the saga. If an error occurs, the session will be closed and all transactions will be dismissed.
func (store *MongoStore) RequestSaga(correlationId string, sagaType string) (*Context, error) {
	maxDuration := 15 * time.Second
	session, err := store.client.StartSession(&options.SessionOptions{DefaultMaxCommitTime: &maxDuration})
	if err != nil {
		panic(err)
	}

	err = session.StartTransaction()
	if err != nil {
		panic(err)
	}

	saga := new(Saga)
	ctx := context.Background()
	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		_, err := store.collection.UpdateOne(sc, bson.M{"correlationId": correlationId, "type": sagaType}, bson.M{"$set": bson.M{"locktime": time.Now().UTC()}})
		if err != nil {
			if err = session.AbortTransaction(sc); err != nil {
				panic(err)
			}
			panic(err)
		}

		err = store.collection.FindOne(sc, bson.M{"correlationId": correlationId, "type": sagaType}).Decode(saga)
		if err != nil {
			if err = session.AbortTransaction(sc); err != nil {
				panic(err)
			}
			panic(err)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	sagaContext := CreateContext(store, CreateMongoSession(ctx, session))
	sagaContext.State = saga.State
	sagaContext.Type = saga.Type
	sagaContext.CorrelationId = saga.CorrelationID
	sagaContext.IsCompleted = saga.IsCompleted

	return sagaContext, nil
}

//Create a new saga with the given correlationId and sagaType in the MongoStore
func (store *MongoStore) CreateSaga(correlationId string, sagaType string) error {
	saga := &Saga{
		ID:            primitive.NewObjectID(),
		CorrelationID: correlationId,
		Type:          sagaType,
		State:         nil,
		IsCompleted:   false,
	}
	_, err := store.collection.InsertOne(context.Background(), saga)
	if err != nil {
		return err
	}
	return nil
}

func (store *MongoStore) UpdateState(session Session, correlationId string, sagaType string, state map[string]interface{}) error {
	mongoSession := session.(*MongoSession)
	err := mongo.WithSession(mongoSession.Context, mongoSession.Session, func(sc mongo.SessionContext) error {
		_, err := store.collection.UpdateOne(sc, bson.M{"correlationId": correlationId, "type": sagaType}, bson.M{"$set": bson.M{"state": state}})
		//Update failed, abort transaction
		if err != nil {
			if err = mongoSession.Session.AbortTransaction(sc); err != nil {
				return err
			}
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (store *MongoStore) CompleteSaga(session Session, correlationId string, sagaType string) error {
	mongoSession := session.(*MongoSession)
	err := mongo.WithSession(mongoSession.Context, mongoSession.Session, func(sc mongo.SessionContext) error {
		_, err := store.collection.UpdateOne(sc, bson.M{"correlationId": correlationId, "type": sagaType}, bson.M{"$set": bson.M{"isCompleted": true}})
		//Update failed, abort transaction
		if err != nil {
			if err = mongoSession.Session.AbortTransaction(sc); err != nil {
				panic(err)
			}
			panic(err)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return nil
}

func (store *MongoStore) DeleteSaga(correlationId string, sagaType string) error {
	panic("Not implemented")
}
