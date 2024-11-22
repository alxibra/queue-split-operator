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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	lg := log.Log

	queuesplit := &messagingv1alpha1.QueueSplit{}
	err := r.Get(context.Background(), req.NamespacedName, queuesplit)
	if err != nil {
		if errors.IsNotFound(err) {
			lg.Info("Queuesplit instance not found")
			return ctrl.Result{}, nil
		}
		lg.Error(err, "Fetch queuesplit instance error")
		return ctrl.Result{}, err
	}
	lg.Info("Queuesplit found\n")
	if queuesplit.ObjectMeta.DeletionTimestamp.IsZero() {
		lg.Info("To Be Delete\n")
	}
	existingRs := &appsv1.ReplicaSet{}
	err = r.Get(
		context.Background(),
		types.NamespacedName{
			Name:      queuesplit.Name,
			Namespace: queuesplit.Namespace,
		},
		existingRs)
	if err != nil && errors.IsNotFound(err) {
		lg.Info("ReplicaSet not found\n")
		rs := buildReplicaSet()
		err = r.Create(context.Background(), rs)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	lg.Info("ReplicaSet found\n")

	if !existingRs.ObjectMeta.DeletionTimestamp.IsZero() {
		lg.Info("Resource is being deleted", "name", existingRs.Name)
	}

	return ctrl.Result{}, nil
}

func buildReplicaSet() *appsv1.ReplicaSet {
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: int32Ptr(3), // Desired number of replicas
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-app",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-app",
					},
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
