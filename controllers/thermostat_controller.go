/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	automationv1 "github.com/pbochynski/thermostat/api/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ThermostatReconciler reconciles a Thermostat object
type ThermostatReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	sensorField = ".spec.sensors"
)

//+kubebuilder:rbac:groups=automation.demo.pb,resources=thermostats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=automation.demo.pb,resources=thermostats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=automation.demo.pb,resources=thermostats/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Thermostat object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ThermostatReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Thermostat reconciliation happened")

	var thermostat automationv1.Thermostat
	if err := r.Get(ctx, req.NamespacedName, &thermostat); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// TODO(user): your logic here
	var minTemp *resource.Quantity
	for _, item := range thermostat.Spec.Sensors {
		var sensor automationv1.Sensor
		if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: item}, &sensor); err == nil {
			if minTemp == nil {
				minTemp = &sensor.Status.Temperature
			} else if minTemp.Cmp(sensor.Status.Temperature) == 1 {
				minTemp = &sensor.Status.Temperature
			}
		}
	}
	if minTemp != nil {
		thermostat.Status.Temperature = *minTemp
		if thermostat.Status.Temperature.Cmp(thermostat.Spec.Temperature) == -1 {
			thermostat.Status.Heat = true
		}
		if thermostat.Status.Temperature.Cmp(thermostat.Spec.Temperature) == 1 {
			thermostat.Status.Heat = false
		}
	} else {
		thermostat.Status.Heat = false
	}

	r.Status().Update(ctx, &thermostat)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ThermostatReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &automationv1.Thermostat{}, sensorField, func(rawObj client.Object) []string {
		thermostat := rawObj.(*automationv1.Thermostat)
		return thermostat.Spec.Sensors
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&automationv1.Thermostat{}).
		Watches(&source.Kind{Type: &automationv1.Sensor{}},
			handler.EnqueueRequestsFromMapFunc(r.findThermostatsForSensor),
		).Complete(r)
}

func (r *ThermostatReconciler) findThermostatsForSensor(sensor client.Object) []reconcile.Request {
	attachedThermostats := &automationv1.ThermostatList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(sensorField, sensor.GetName()),
		Namespace:     sensor.GetNamespace(),
	}
	err := r.List(context.TODO(), attachedThermostats, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(attachedThermostats.Items))
	for i, item := range attachedThermostats.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}
