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
	// EventRecorder record.EventRecorder
}

//+kubebuilder:rbac:groups=csye7125-fall2023-group07.operator.souvikdinda.me,resources=httpchecks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=csye7125-fall2023-group07.operator.souvikdinda.me,resources=httpchecks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=csye7125-fall2023-group07.operator.souvikdinda.me,resources=httpchecks/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods/log,verbs=get;list;watch

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

	// eventRecorder := r.EventRecorderFor("httpcheck-controller")
	log := log.FromContext(ctx)
	fmt.Println("Reconciling HttpCheck")
	httpCheck := &csye7125fall2023group07v1.HttpCheck{}

	if err := r.Get(ctx, req.NamespacedName, httpCheck); err != nil {
		if client.IgnoreNotFound(err) != nil {
			// r.EventRecorder.Event(httpCheck, "Warning", "NotFound", fmt.Sprintf("HttpCheck %s not found", req.NamespacedName))
			log.Error(err, "unable to fetch HttpCheck")
			return ctrl.Result{}, err
		}
	}

	myFilnalizer := "csye7125-fall2023-group07.operator.souvikdinda.me/finalizer"

	if httpCheck.ObjectMeta.DeletionTimestamp.IsZero() {

		if !controllerutil.ContainsFinalizer(httpCheck, myFilnalizer) {
			// r.EventRecorder.Event(httpCheck, "Normal", "AddFinalizer", "Adding Finalizer for the HttpCheck")
			log.Info("Adding Finalizer for the HttpCheck")
			controllerutil.AddFinalizer(httpCheck, myFilnalizer)
			if err := r.Update(ctx, httpCheck); err != nil {
				// r.EventRecorder.Event(httpCheck, "Warning", "UpdateError", fmt.Sprintf("Failed to update HttpCheck with finalizer: %s", err))
				log.Error(err, "unable to update HttpCheck")
				return ctrl.Result{}, err
			}
		}

		if err := r.reconcileCreateOrUpdate(ctx, httpCheck); err != nil {
			// r.EventRecorder.Event(httpCheck, "Warning", "ReconcileError", fmt.Sprintf("Failed to reconcile HttpCheck: %s", err))
			log.Error(err, "unable to reconcile HttpCheck")
			return ctrl.Result{}, err
		}

	} else {

		if controllerutil.ContainsFinalizer(httpCheck, myFilnalizer) {
			// r.EventRecorder.Event(httpCheck, "Normal", "DeleteFinalizer", "Deleting HttpCheck Finalizer")
			log.Info("Deleting HttpCheck Finalizer")
			if err := r.cleanupResources(ctx, httpCheck); err != nil {
				// r.EventRecorder.Event(httpCheck, "Warning", "CleanupError", fmt.Sprintf("Failed to cleanup resources: %s", err))
				log.Error(err, "unable to cleanup resources")
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(httpCheck, myFilnalizer)
			if err := r.Update(ctx, httpCheck); err != nil {
				// r.EventRecorder.Event(httpCheck, "Warning", "UpdateError", fmt.Sprintf("Failed to update HttpCheck with finalizer: %s", err))
				log.Error(err, "unable to update HttpCheck")
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// Create or update the associated resources
func (r *HttpCheckReconciler) reconcileCreateOrUpdate(ctx context.Context, httpCheck *csye7125fall2023group07v1.HttpCheck) error {
	log := log.FromContext(ctx)

	configMap := newConfigMap(httpCheck)

	if err := ctrl.SetControllerReference(httpCheck, configMap, r.Scheme); err != nil {
		// r.EventRecorder.Event(httpCheck, "Warning", "SetOwnerReferenceError", fmt.Sprintf("Failed to set owner reference on ConfigMap: %s", err))
		log.Error(err, "unable to set owner reference on ConfigMap")
		return err
	}

	if err := r.Update(ctx, configMap); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, configMap); err != nil {
				// r.EventRecorder.Event(httpCheck, "Warning", "CreateError", fmt.Sprintf("Failed to create ConfigMap: %s", err))
				log.Error(err, "unable to create ConfigMap")
				return err
			}
		} else {
			// r.EventRecorder.Event(httpCheck, "Warning", "UpdateError", fmt.Sprintf("Failed to update ConfigMap: %s", err))
			log.Error(err, "unable to update ConfigMap")
			return err
		}
	}

	cronJob := newCronJob(httpCheck, configMap)

	if err := ctrl.SetControllerReference(httpCheck, cronJob, r.Scheme); err != nil {
		// r.EventRecorder.Event(httpCheck, "Warning", "SetOwnerReferenceError", fmt.Sprintf("Failed to set owner reference on CronJob: %s", err))
		log.Error(err, "unable to set owner reference on CronJob")
		return err
	}

	if httpCheck.Spec.IsPaused {
		// r.EventRecorder.Event(httpCheck, "Normal", "HttpCheckPaused", "HttpCheck is paused")
		log.Info("HttpCheck is paused")
		suspend := true
		cronJob.Spec.Suspend = &suspend
	} else {
		// r.EventRecorder.Event(httpCheck, "Normal", "HttpCheckNotPaused", "HttpCheck is not paused")
		log.Info("HttpCheck is not paused")
		suspend := false
		cronJob.Spec.Suspend = &suspend
	}

	if err := r.Update(ctx, cronJob); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, cronJob); err != nil {
				// r.EventRecorder.Event(httpCheck, "Warning", "CreateError", fmt.Sprintf("Failed to create CronJob: %s", err))
				log.Error(err, "unable to create CronJob")
				return err
			}
		} else {
			// r.EventRecorder.Event(httpCheck, "Warning", "UpdateError", fmt.Sprintf("Failed to update CronJob: %s", err))
			log.Error(err, "unable to update CronJob")
			return err
		}
	}

	cronJobStatus := &batchv1.CronJob{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(cronJob), cronJobStatus); err != nil {
		// r.EventRecorder.Event(httpCheck, "Warning", "GetError", fmt.Sprintf("Failed to get CronJob status: %s", err))
		log.Error(err, "unable to get CronJob status")
		return err
	}

	if cronJobStatus.Status.LastScheduleTime != nil {
		httpCheck.Status.LastExecutionTime = metav1.Time{Time: cronJobStatus.Status.LastScheduleTime.Time}
	}

	if cronJobStatus.Spec.Suspend != nil && *cronJobStatus.Spec.Suspend {
		// r.EventRecorder.Event(httpCheck, "Normal", "HttpCheckPaused", "HttpCheck is paused due to suspend flag on CronJob")
		httpCheck.Status.CronJobStatus = "Paused"
	} else {
		// r.EventRecorder.Event(httpCheck, "Normal", "HttpCheckRunning", "HttpCheck is running")
		httpCheck.Status.CronJobStatus = "Running"
	}

	if err := r.Status().Update(ctx, httpCheck); err != nil {
		// r.EventRecorder.Event(httpCheck, "Warning", "UpdateError", fmt.Sprintf("Failed to update HttpCheck status: %s", err))
		log.Error(err, "unable to update HttpCheck status")
		return err
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
		return err
	}

	if err := r.Delete(ctx, configMap); err != nil && !errors.IsNotFound(err) {
		// r.EventRecorder.Event(httpCheck, "Warning", "DeleteError", fmt.Sprintf("Failed to delete ConfigMap: %s", err))
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
		// r.EventRecorder.Event(httpCheck, "Warning", "DeleteError", fmt.Sprintf("Failed to delete CronJob: %s", err))
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
			"NAME":                 httpCheck.Spec.Name,
			"HEALTH_CHECK_URL":     httpCheck.Spec.Uri,
			"NUM_RETRIES":          strconv.Itoa(httpCheck.Spec.NumRetries),
			"RESPONSE_STATUS_CODE": strconv.Itoa(httpCheck.Spec.ResponseStatusCode),
			"KAFKA_BROKERS":        "infra-kafka-controller-0.infra-kafka-controller-headless.kafka.svc.cluster.local:9092,infra-kafka-controller-1.infra-kafka-controller-headless.kafka.svc.cluster.local:9092,infra-kafka-controller-2.infra-kafka-controller-headless.kafka.svc.cluster.local:9092",
			"KAFKA_TOPIC":          "healthcheck",
		},
	}
}

func newCronJob(httpCheck *csye7125fall2023group07v1.HttpCheck, configMap *corev1.ConfigMap) *batchv1.CronJob {
	backOffLimit := int32(httpCheck.Spec.NumRetries)

	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-cronjob", httpCheck.Spec.Name),
			Namespace: httpCheck.Namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: fmt.Sprintf("*/%d * * * *", httpCheck.Spec.CheckInterval),
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					BackoffLimit: &backOffLimit,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							ServiceAccountName: "webapp-operator-controller-manager",
							Containers: []corev1.Container{
								{
									Name:  "httpcheck",
									Image: "quay.io/csye-7125/producerapp:latest",
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
							ImagePullSecrets: []corev1.LocalObjectReference{
								{Name: "webapp-operator-quay-creds"},
							},
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
