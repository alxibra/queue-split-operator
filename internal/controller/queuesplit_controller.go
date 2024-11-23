/*
Copyright 2024.

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
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	messagingv1alpha1 "github.com/alxibra/queue-split-operator/api/v1alpha1"
)

// QueueSplitReconciler reconciles a QueueSplit object
type QueueSplitReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=messaging.yok.travel,resources=queuesplits,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=messaging.yok.travel,resources=queuesplits/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=messaging.yok.travel,resources=queuesplits/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the QueueSplit object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *QueueSplitReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lg := log.FromContext(ctx)

	queuesplit := &messagingv1alpha1.QueueSplit{}
	err := r.Get(ctx, req.NamespacedName, queuesplit)
	if err != nil {
		if errors.IsNotFound(err) {
			lg.Info("Queuesplit instance not found")
			return ctrl.Result{}, nil
		}
		lg.Error(err, "Fetch queuesplit instance error")
		return ctrl.Result{}, err
	}
	lg.Info("Queuesplit found\n")

	existingRs := &appsv1.ReplicaSet{}
	err = r.Get(
		ctx,
		types.NamespacedName{
			Name:      queuesplit.Name,
			Namespace: queuesplit.Namespace,
		},
		existingRs)
	if err != nil && errors.IsNotFound(err) {
		lg.Info("ReplicaSet not found\n")
		rs := buildReplicaSet(*queuesplit)
		if err := controllerutil.SetControllerReference(queuesplit, rs, r.Scheme); err != nil {
			lg.Error(err, "Failed to SetControllerReference")
			return ctrl.Result{}, err
		}
		err = r.Create(ctx, rs)
		if err != nil {
			lg.Error(err, "Failed to create ReplicaSet")
			return ctrl.Result{}, err
		}
		// Requeue to verify creation
		return ctrl.Result{Requeue: true}, nil
	}
	if err != nil {
		lg.Error(err, "Failed to get ReplicaSet")
		return ctrl.Result{}, err
	}

	dcRs := existingRs.DeepCopy()
	dcRs.Spec.Replicas = int32Ptr(int32(queuesplit.Spec.Replicas))
	dcRs.Spec.Template.Spec.Containers[0].Env = buildEnvVars(queuesplit)

	if !reflect.DeepEqual(existingRs.Spec, dcRs.Spec) || !reflect.DeepEqual(existingRs.Annotations, dcRs.Annotations) {
		lg.Info("Updating ReplicaSet")
		err = r.Update(ctx, dcRs)
		if err != nil {
			lg.Error(err, "Failed to update ReplicaSet")
			return ctrl.Result{}, err
		}
		lg.Info("ReplicaSet updated successfully")
		return ctrl.Result{}, nil
	}
	lg.Info("ReplicaSet is already up-to-date")
	return ctrl.Result{}, nil
}

func labels(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       name,
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/managed-by": "queuesplit-controller",
		"app.kubernetes.io/part-of":    "messaging-system",
		"app":                          name,
	}
}

func annotations(name, namespace, version, revision string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by":    "queuesplit-controller",
		"queuesplit.yok.travel/name":      name,
		"queuesplit.yok.travel/namespace": namespace,
	}
}

func buildEnvVars(qs *messagingv1alpha1.QueueSplit) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "GATEWAY_Q",
			Value: qs.Spec.QueueName,
		},
		{
			Name:  "SPLIT_Q_0",
			Value: qs.Spec.Destinations[0].Name,
		},
		{
			Name:  "SPLIT_Q_WEIGHT_0",
			Value: strconv.Itoa(qs.Spec.Destinations[0].Weight),
		},
		{
			Name:  "SPLIT_Q_1",
			Value: qs.Spec.Destinations[1].Name,
		},
		{
			Name:  "SPLIT_Q_WEIGHT_1",
			Value: strconv.Itoa(qs.Spec.Destinations[1].Weight),
		},
		{
			Name:  "PREFETCH_COUNT",
			Value: strconv.Itoa(qs.Spec.PrefetchCount),
		},
		{
			Name: "QUEUE_HOST",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: qs.Spec.SecretName,
					},
					Key: "queue-host",
				},
			},
		},
	}
}

func buildReplicaSet(qs messagingv1alpha1.QueueSplit) *appsv1.ReplicaSet {
	name, namespace := qs.Name, qs.Namespace
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels(name),
			Annotations: annotations(name, namespace, "alpha1.0", "0"),
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: int32Ptr(int32(qs.Spec.Replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels(name),
					Annotations: annotations(name, namespace, "alpha1.0", "0"),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx:1.21.1",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "GATEWAY_Q",
									Value: qs.Spec.QueueName,
								},
								{
									Name:  "SPLIT_Q_0",
									Value: qs.Spec.Destinations[0].Name,
								},
								{
									Name:  "SPLIT_Q_WEIGHT_0",
									Value: strconv.Itoa(qs.Spec.Destinations[0].Weight),
								},
								{
									Name:  "SPLIT_Q_1",
									Value: qs.Spec.Destinations[1].Name,
								},
								{
									Name:  "SPLIT_Q_WEIGHT_1",
									Value: strconv.Itoa(qs.Spec.Destinations[1].Weight),
								},
								{
									Name:  "PREFETCH_COUNT",
									Value: strconv.Itoa(qs.Spec.PrefetchCount),
								},
								{
									Name: "QUEUE_HOST", // Key from the dynamically specified secret
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: qs.Spec.SecretName, // Use the secretName from the QueueSplit spec
											},
											Key: "queue-host", // The specific key in the secret
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return rs
}

func int32Ptr(i int32) *int32 { return &i }

// SetupWithManager sets up the controller with the Manager.
func (r *QueueSplitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&messagingv1alpha1.QueueSplit{}).
		Complete(r)
}
