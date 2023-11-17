/*
Copyright 2023.

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
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	csye7125fall2023group07v1 "github.com/csye7125-fall2023-group07/webapp-operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HttpCheckReconciler reconciles a HttpCheck object
type HttpCheckReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	configMapFinalizer = "configmap.csye7125-fall2023-group07.operator.souvikdinda.me/finalizer"
	cronJobFinalizer   = "cronjob.csye7125-fall2023-group07.operator.souvikdinda.me/finalizer"
)

//+kubebuilder:rbac:groups=csye7125-fall2023-group07.operator.souvikdinda.me,resources=httpchecks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=csye7125-fall2023-group07.operator.souvikdinda.me,resources=httpchecks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=csye7125-fall2023-group07.operator.souvikdinda.me,resources=httpchecks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HttpCheck object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *HttpCheckReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	httpCheck := &csye7125fall2023group07v1.HttpCheck{}

	if err := r.Get(ctx, req.NamespacedName, httpCheck); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "unable to fetch HttpCheck")
			return ctrl.Result{}, err
		}
	}

	myFilnalizer := "csye7125-fall2023-group07.operator.souvikdinda.me/finalizer"

	if httpCheck.ObjectMeta.DeletionTimestamp.IsZero() {

		if !controllerutil.ContainsFinalizer(httpCheck, myFilnalizer) {
			log.Info("Adding Finalizer for the HttpCheck")
			controllerutil.AddFinalizer(httpCheck, myFilnalizer)
			if err := r.Update(ctx, httpCheck); err != nil {
				log.Error(err, "unable to update HttpCheck")
				return ctrl.Result{}, err
			}
		}

		if err := r.reconcileCreateOrUpdate(ctx, httpCheck); err != nil {
			log.Error(err, "unable to reconcile HttpCheck")
			return ctrl.Result{}, err
		}

	} else {

		if controllerutil.ContainsFinalizer(httpCheck, myFilnalizer) {
			log.Info("Deleting HttpCheck Finalizer")
			if err := r.cleanupResources(ctx, httpCheck); err != nil {
				log.Error(err, "unable to cleanup resources")
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(httpCheck, myFilnalizer)
			if err := r.Update(ctx, httpCheck); err != nil {
				log.Error(err, "unable to update HttpCheck")
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *HttpCheckReconciler) reconcileCreateOrUpdate(ctx context.Context, httpCheck *csye7125fall2023group07v1.HttpCheck) error {
	log := log.FromContext(ctx)

	configMap := newConfigMap(httpCheck)

	if err := ctrl.SetControllerReference(httpCheck, configMap, r.Scheme); err != nil {
		log.Error(err, "unable to set owner reference on ConfigMap")
		return err
	}

	if err := r.Update(ctx, configMap); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, configMap); err != nil {
				log.Error(err, "unable to create ConfigMap")
				return err
			}
		} else {
			log.Error(err, "unable to update ConfigMap")
			return err
		}
	}

	cronJob := newCronJob(httpCheck, configMap)

	if err := ctrl.SetControllerReference(httpCheck, cronJob, r.Scheme); err != nil {
		log.Error(err, "unable to set owner reference on CronJob")
		return err
	}

	if err := r.Update(ctx, cronJob); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, cronJob); err != nil {
				log.Error(err, "unable to create CronJob")
				return err
			}
		} else {
			log.Error(err, "unable to update CronJob")
			return err
		}
	}

	return nil
}

func (r *HttpCheckReconciler) cleanupResources(ctx context.Context, httpCheck *csye7125fall2023group07v1.HttpCheck) error {
	log := log.FromContext(ctx)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-configmap", httpCheck.Spec.Name),
			Namespace: httpCheck.Namespace,
		},
	}

	if err := r.Update(ctx, configMap); err != nil {
		log.Error(err, "unable to remove finalizers from ConfigMap")
		return err
	}

	if err := r.Delete(ctx, configMap); err != nil && !errors.IsNotFound(err) {
		log.Error(err, "unable to delete ConfigMap")
		return err
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-cronjob", httpCheck.Spec.Name),
			Namespace: httpCheck.Namespace,
		},
	}

	if err := r.Delete(ctx, cronJob); err != nil && !errors.IsNotFound(err) {
		log.Error(err, "unable to delete CronJob")
		return err
	}

	return nil
}

func newConfigMap(httpCheck *csye7125fall2023group07v1.HttpCheck) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-configmap", httpCheck.Spec.Name),
			Namespace: httpCheck.Namespace,
		},
		Data: map[string]string{
			"name":                      httpCheck.Spec.Name,
			"uri":                       httpCheck.Spec.Uri,
			"num_retries":               strconv.Itoa(httpCheck.Spec.NumRetries),
			"response_status_code":      strconv.Itoa(httpCheck.Spec.ResponseStatusCode),
			"check_interval_in_seconds": strconv.Itoa(httpCheck.Spec.CheckInterval),
		},
	}
}

func newCronJob(httpCheck *csye7125fall2023group07v1.HttpCheck, configMap *corev1.ConfigMap) *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-cronjob", httpCheck.Spec.Name),
			Namespace: httpCheck.Namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: fmt.Sprintf("*/%d * * * *", httpCheck.Spec.CheckInterval),
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "httpcheck",
									Image: "souvikdinda/csye7125-fall2023-group07:latest",
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 8080,
											Protocol:      "TCP",
										},
									},
									EnvFrom: []corev1.EnvFromSource{
										{
											ConfigMapRef: &corev1.ConfigMapEnvSource{
												LocalObjectReference: corev1.LocalObjectReference{Name: configMap.Name},
											},
										},
									},
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *HttpCheckReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&csye7125fall2023group07v1.HttpCheck{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&batchv1.CronJob{}).
		Complete(r)
}
