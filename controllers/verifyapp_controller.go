/*


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

	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	a "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "github.com/mlu-ibm/demo-operator/api/v1"
)

// VerifyAppReconciler reconciles a VerifyApp object
type VerifyAppReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps.ibm.com,resources=verifyapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.ibm.com,resources=verifyapps/status,verbs=get;update;patch

func (r *VerifyAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("verifyapp", req.NamespacedName)

	log.Info("Processing VerifyAppReconciler.")
	verifyApp := &appsv1.VerifyApp{}
	err := r.Client.Get(ctx, req.NamespacedName, verifyApp)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found
			log.Info("VerifyApp resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request
		log.Error(err, "failed to get VerifyApp")
		return ctrl.Result{}, err
	}

	found := &a.Deployment{}

	err = r.Client.Get(ctx, types.NamespacedName{Name: verifyApp.Name, Namespace: verifyApp.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		dep := r.deployVerifyApp(verifyApp)
		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Client.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}

		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// Check desired amount of deployments.
	size := verifyApp.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Client.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}

		// Spec updated - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *VerifyAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.VerifyApp{}).
		Complete(r)
}

func (c *VerifyAppReconciler) deployVerifyApp(ha *appsv1.VerifyApp) *a.Deployment {
	replicas := ha.Spec.Size
	labels := map[string]string{"app": "mock-containers"}
	image := ha.Spec.Image
	dep := &a.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ha.Name,
			Namespace: ha.Namespace,
		},
		Spec: a.DeploymentSpec{
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
						Image: image,
						Name:  ha.Name,
					}},
				},
			},
		},
	}
	ctrl.SetControllerReference(ha, dep, c.Scheme)
	return dep

}
