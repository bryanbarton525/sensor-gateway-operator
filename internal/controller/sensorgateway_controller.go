/*
Copyright 2025.

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

package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	iotv1alpha1 "github.com/bryanbarton525/sensor-gateway-operator/api/v1alpha1"
)

// SensorGatewayReconciler reconciles a SensorGateway object
type SensorGatewayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=iot.iambarton.com,resources=sensorgateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=iot.iambarton.com,resources=sensorgateways/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=iot.iambarton.com,resources=sensorgateways/finalizers,verbs=update

func (r *SensorGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling SensorGateway", "name", req.Name, "namespace", req.Namespace)

	// Fetch the SensorGateway instance
	sensorGateway := &iotv1alpha1.SensorGateway{}
	if err := r.Get(ctx, req.NamespacedName, sensorGateway); err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			log.Info("SensorGateway resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get SensorGateway")
		return ctrl.Result{}, err
	}
	// Check if the deployment already exists, if not create a new one
	found := &iotv1alpha1.SensorGateway{}
	err := r.Get(ctx, types.NamespacedName{Name: sensorGateway.Name, Namespace: sensorGateway.Namespace}, found)
	if err != nil && k8serrors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForGateway(ctx, sensorGateway)
		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err := r.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create Deployment")
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}
	// TODO : Update the deployment if necessary
	// Deployment already exists - don't requeue
	log.Info("Skip reconcile: Deployment already exists", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
	return ctrl.Result{}, nil
}

// deploymentForGateway returns a SensorGateway Deployment object
func (r *SensorGatewayReconciler) deploymentForGateway(ctx context.Context, gw *iotv1alpha1.SensorGateway) *appsv1.Deployment {
	labels := map[string]string{
		"app":           "sensor-gateway",
		"controller_cr": gw.Name,
	}
	replicas := int32(1)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gw.Name,
			Namespace: gw.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: gw.Spec.Image,
						Name:  "sensor-gateway-pod",
						Env: []corev1.EnvVar{
							{
								Name:  "MQTT_BROKER_URL",
								Value: gw.Spec.BrokerURL,
							},
							{
								Name:  "MQTT_BROKER_PORT",
								Value: gw.Spec.BrokerPort,
							},
							{
								Name:  "MQTT_TOPIC",
								Value: gw.Spec.Topic,
							},
						},
					}},
				},
			},
		},
	}
	err := ctrl.SetControllerReference(gw, dep, r.Scheme)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to set controller reference")
		return nil
	}
	return dep
}

// SetupWithManager sets up the controller with the Manager.
func (r *SensorGatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iotv1alpha1.SensorGateway{}). // Ensure that the controller watches for changes to SensorGateway resources
		Owns(&appsv1.Deployment{}).        // Ensure that the controller watches for changes to Deployment resources
		Named("sensorgateway").            // Set a name for the controller
		Complete(r)                        // Complete the controller setup with the reconciler
}
