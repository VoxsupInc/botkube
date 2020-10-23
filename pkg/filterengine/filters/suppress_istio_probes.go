package filters

import (
	"reflect"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/utils"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// SuppressIstioProbes suppresses istio proxy probe errors
type SuppressIstioProbes struct {
	Description string
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(SuppressIstioProbes{
		Description: "Suppresses istio proxy readiness probe errors.",
	})
}

// Run filters and modifies event struct
func (s SuppressIstioProbes) Run(object interface{}, event *events.Event) {
	if event.Kind != "Pod" {
		return
	}

	var eventObj coreV1.Event
	err := utils.TransformIntoTypedObject(object.(*unstructured.Unstructured), &eventObj)
	if err != nil {
		log.Errorf("Unable to tranform object type: %v, into type: %v", reflect.TypeOf(object), reflect.TypeOf(eventObj))
		return
	}

	if eventObj.InvolvedObject != (coreV1.ObjectReference{}) && eventObj.InvolvedObject.FieldPath == "spec.containers{istio-proxy}" {
		if event.Type == config.ErrorEvent && event.Reason == "Unhealthy" {
			// istio readiness check is failed, so skipping
			event.Skip = true
		}
	}
}

// Describe filter
func (s SuppressIstioProbes) Describe() string {
	return s.Description
}
