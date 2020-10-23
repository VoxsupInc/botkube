package filters

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/filterengine"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/utils"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var kubectlBinary = "/usr/local/bin/kubectl"

// SuppressStoppedPodErrors suppresses istio proxy probe errors
type SuppressStoppedPodErrors struct {
	Description string
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(SuppressStoppedPodErrors{
		Description: "Suppresses errors from pods that are stopped.",
	})
}

// Run filters and modifies event struct
func (s SuppressStoppedPodErrors) Run(object interface{}, event *events.Event) {
	if event.Kind != "Pod" {
		return
	}

	var eventObj coreV1.Event
	err := utils.TransformIntoTypedObject(object.(*unstructured.Unstructured), &eventObj)
	if err != nil {
		log.Errorf("Unable to tranform object type: %v, into type: %v", reflect.TypeOf(object), reflect.TypeOf(eventObj))
		return
	}

	if event.Type == config.ErrorEvent && event.Reason == "Unhealthy" {
		var args = []string{"get", "pod", eventObj.InvolvedObject.Name, "-n", eventObj.InvolvedObject.Namespace, "-o=json"}
		runner := execute.NewCommandRunner(kubectlBinary, args)
		out, err := runner.Run()
		if err != nil {
			// probably pod is deleted
			event.Skip = true
			return
		}

		var podObj coreV1.Pod

		err = json.Unmarshal([]byte(out), &podObj)

		if err != nil {
			log.Error("Unable to tranform json into coreV1.Pod")
			return
		}

		if podObj.DeletionTimestamp == nil {
			// no timestamp means pod can be running
			return
		}

		dTimestamp := podObj.DeletionTimestamp
		dSec := *podObj.DeletionGracePeriodSeconds

		if time.Now().After(dTimestamp.Add(-time.Duration(dSec) * time.Second)) {
			// even occurred during termination period.
			event.Skip = true
			return
		}
	}
}

// Describe filter
func (s SuppressStoppedPodErrors) Describe() string {
	return s.Description
}
