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
	"reflect"

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

//+kubebuilder:rbac:groups=iot.iambarton.com,resources=sensorgateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=iot.iambarton.com,resources=sensorgateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *SensorGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciliation loop started")

	// 1. Fetch the SensorGateway resource
	gateway := &iotv1alpha1.SensorGateway{}
	if err := r.Get(ctx, req.NamespacedName, gateway); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("SensorGateway resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get SensorGateway")
		return ctrl.Result{}, err
	}

	// 2. Check if the Deployment already exists
	foundDeployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: gateway.Name, Namespace: gateway.Namespace}, foundDeployment)

	// 3. Deployment does not exist - CREATE IT
	if err != nil && k8serrors.IsNotFound(err) {
		logger.Info("Deployment not found. Creating a new one.")
		desiredDeployment, err := r.deploymentForGateway(ctx, gateway)
		if err != nil {
			logger.Error(err, "Failed to create desired Deployment")
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, desiredDeployment); err != nil {
			logger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", desiredDeployment.Namespace, "Deployment.Name", desiredDeployment.Name)
			return ctrl.Result{}, err
		}
		logger.Info("Successfully created new Deployment.")
		// Requeue to check status after creation
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		// Some other error occurred when trying to fetch the deployment
		logger.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// 4. Deployment EXISTS - UPDATE IT if necessary
	logger.Info("Deployment found. Checking for drift.")
	desiredDeployment, err := r.deploymentForGateway(ctx, gateway)
	if err != nil {
		logger.Error(err, "Failed to create desired Deployment")
		return ctrl.Result{}, err
	}

	// Compare the pod template spec of the found vs desired deployment
	if !reflect.DeepEqual(foundDeployment.Spec.Template.Spec, desiredDeployment.Spec.Template.Spec) {
		logger.Info("Drift detected. Updating Deployment spec.")
		foundDeployment.Spec.Template.Spec = desiredDeployment.Spec.Template.Spec
		if err := r.Update(ctx, foundDeployment); err != nil {
			logger.Error(err, "Failed to update Deployment", "Deployment.Namespace", foundDeployment.Namespace, "Deployment.Name", foundDeployment.Name)
			return ctrl.Result{}, err
		}
		logger.Info("Successfully updated Deployment.")
		return ctrl.Result{Requeue: true}, nil
	}

	logger.Info("Reconciliation finished. No changes required.")
	return ctrl.Result{}, nil
}

// deploymentForGateway returns a SensorGateway Deployment object
func (r *SensorGatewayReconciler) deploymentForGateway(ctx context.Context, gw *iotv1alpha1.SensorGateway) (*appsv1.Deployment, error) {
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
							{
								Name:  "SENSOR_TYPE",
								Value: gw.Spec.SensorType,
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
		return nil, err
	}
	return dep, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SensorGatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iotv1alpha1.SensorGateway{}). // Ensure that the controller watches for changes to SensorGateway resources
		Owns(&appsv1.Deployment{}).        // Ensure that the controller watches for changes to Deployment resources
		Named("sensorgateway").            // Set a name for the controller
		Complete(r)                        // Complete the controller setup with the reconciler
}
