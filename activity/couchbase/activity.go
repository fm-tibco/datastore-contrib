package couchbase

import (
	"fmt"

	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/metadata"
	"gopkg.in/couchbase/gocb.v1"
)

const (
	methodGet    = "Get"
	methodInsert = "Insert"
	methodUpsert = "Upsert"
	methodRemove = "Remove"
)

func init() {
	_ = activity.Register(&Activity{}, New)
}
func New(ctx activity.InitContext) (activity.Activity, error) {
	s := &Settings{}
	err := metadata.MapToStruct(ctx.Settings(), s, true)
	if err != nil {
		return nil, err
	}
	act := &Activity{settings: s}
	return act, nil
}

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

type Activity struct {
	settings *Settings
}

// Metadata returns the activity's metadata
func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

// Eval implements api.Activity.Eval - Logs the Message
func (a *Activity) Eval(ctx activity.Context) (done bool, err error) {
	logger := ctx.Logger()

	input := &Input{}

	err = ctx.GetInputObject(input)
	if err != nil {
		return false, err
	}

	cluster, connectError := gocb.Connect(a.settings.Server)
	if connectError != nil {
		logger.Errorf("Connection error: %v", connectError)
		return false, connectError
	}

	cluster.Authenticate(gocb.PasswordAuthenticator{
		Username: a.settings.Username,
		Password: a.settings.Password,
	})

	bucket, openBucketError := cluster.OpenBucket(a.settings.BucketName, a.settings.BucketPassword)
	if openBucketError != nil {
		logger.Errorf("Error while opening the bucked with the specified credentials: %v", openBucketError)
		return false, openBucketError
	}

	defer bucket.Close()

	output := &Output{}

	switch input.Method {
	case methodInsert:
		cas, methodError := bucket.Insert(input.Key, input.Data, uint32(input.Expiry))
		if methodError != nil {
			logger.Errorf("Insert error: %v", methodError)
			return false, methodError
		}
		output.Data = cas
		ctx.SetOutputObject(output)
		return true, nil
	case methodUpsert:
		cas, methodError := bucket.Upsert(input.Key, input.Data, uint32(input.Expiry))
		if methodError != nil {
			logger.Errorf("Upsert error: %v", methodError)
			return false, methodError
		}
		output.Data = cas
		ctx.SetOutputObject(output)
		return true, nil
	case methodRemove:
		cas, methodError := bucket.Remove(input.Key, 0)
		if methodError != nil {
			logger.Errorf("Remove error: %v", methodError)
			return false, methodError
		}
		output.Data = cas
		ctx.SetOutputObject(output)
		return true, nil
	case methodGet:
		var document interface{}
		_, methodError := bucket.Get(input.Key, &document)
		if methodError != nil {
			logger.Errorf("Get error: %v", methodError)
			return false, methodError
		}
		output.Data = document
		ctx.SetOutputObject(output)
		return true, nil
	default:
		logger.Errorf("Method %v not recognized.", input.Method)
		return false, fmt.Errorf("method %v not recognized", input.Method)
	}

	//return true, nil
}