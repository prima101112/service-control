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

// Note : all this using int32 to make it easy to use in kubernetes api.
// since if we just use int the compiler machine will be determinde the size 32 or 64.
// so set default to 32 make it more maintainable and sure

package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appv1alpha1 "github.com/prima101112/service-control/api/v1alpha1"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=app.teraskula.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.teraskula.com,resources=applications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=app.teraskula.com,resources=applications/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Application instance
	app := &appv1alpha1.Application{}
	err := r.Get(ctx, req.NamespacedName, app)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Application resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Application")
		return ctrl.Result{}, err
	}

	// Reconcile Deployment : required
	if err := r.reconcileDeployment(ctx, app); err != nil {
		log.Error(err, "Failed to reconcile Deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Service : required
	if err := r.reconcileService(ctx, app); err != nil {
		log.Error(err, "Failed to reconcile Service")
		return ctrl.Result{}, err
	}

	// Reconcile Ingress : required (if host is configured or explicit ingress config exists)
	if app.Spec.Host != "" || app.Spec.Ingress != nil {
		if err := r.reconcileIngress(ctx, app); err != nil {
			log.Error(err, "Failed to reconcile Ingress")
			return ctrl.Result{}, err
		}
	}

	// Reconcile HPA (just if configured)
	if app.Spec.HPA != nil {
		if err := r.reconcileHPA(ctx, app); err != nil {
			log.Error(err, "Failed to reconcile HPA")
			return ctrl.Result{}, err
		}
	}

	// Update status
	if err := r.updateStatus(ctx, app); err != nil {
		log.Error(err, "Failed to update Application status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// reconcile deployments
func (r *ApplicationReconciler) reconcileDeployment(ctx context.Context, app *appv1alpha1.Application) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-deployment", app.Name),
			Namespace: app.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		// Set defaults
		replicas := int32(1)
		if app.Spec.Replicas != nil {
			replicas = *app.Spec.Replicas
		}

		port := int32(8080)
		if app.Spec.Port != nil {
			port = *app.Spec.Port
		}

		// define deployment spec in this area we could add some default sidecar container or other things
		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": app.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": app.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  app.Name,
							Image: fmt.Sprintf("%s:%s", app.Spec.Image, app.Spec.Tag),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: port,
								},
							},
							Env: app.Spec.Env,
						},
					},
				},
			},
		}

		// Add resources if specified
		if app.Spec.Resources != nil {
			deployment.Spec.Template.Spec.Containers[0].Resources = *app.Spec.Resources
		}

		// Add health checks if specified
		if app.Spec.HealthCheck != nil {
			healthPath := "/health"
			if app.Spec.HealthCheck.Path != "" {
				healthPath = app.Spec.HealthCheck.Path
			}

			healthPort := port
			if app.Spec.HealthCheck.Port != nil {
				healthPort = *app.Spec.HealthCheck.Port
			}

			initialDelay := int32(10)
			if app.Spec.HealthCheck.InitialDelaySeconds != nil {
				initialDelay = *app.Spec.HealthCheck.InitialDelaySeconds
			}

			period := int32(10)
			if app.Spec.HealthCheck.PeriodSeconds != nil {
				period = *app.Spec.HealthCheck.PeriodSeconds
			}

			probe := &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: healthPath,
						Port: intstr.FromInt32(healthPort),
					},
				},
				InitialDelaySeconds: initialDelay,
				PeriodSeconds:       period,
			}

			deployment.Spec.Template.Spec.Containers[0].LivenessProbe = probe
			deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = probe
		}

		return controllerutil.SetControllerReference(app, deployment, r.Scheme)
	})

	return err
}

// service reconcile its basically service creation or update which will be update if application is updated
func (r *ApplicationReconciler) reconcileService(ctx context.Context, app *appv1alpha1.Application) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-service", app.Name),
			Namespace: app.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {

		// Set defaults ports to make the service port is standardize. but we could also open this to the application spec as configurable options.
		serviceType := corev1.ServiceTypeClusterIP
		servicePort := int32(80)
		targetPort := int32(8080)

		// Replace if in application level service spec is defined
		if app.Spec.Service != nil {
			if app.Spec.Service.Type != "" {
				serviceType = app.Spec.Service.Type
			}
			if app.Spec.Service.Port != nil {
				servicePort = *app.Spec.Service.Port
			}
		}

		if app.Spec.Port != nil {
			targetPort = *app.Spec.Port
		}

		// Define service spec
		service.Spec = corev1.ServiceSpec{
			Type: serviceType,
			Selector: map[string]string{
				"app": app.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       servicePort,
					TargetPort: intstr.FromInt32(targetPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		}

		// Add NodePort if specified
		if app.Spec.Service != nil && app.Spec.Service.NodePort != nil {
			service.Spec.Ports[0].NodePort = *app.Spec.Service.NodePort
		}

		// Add annotations if specified
		if app.Spec.Service != nil && app.Spec.Service.Annotations != nil {
			service.Annotations = app.Spec.Service.Annotations
		}

		return controllerutil.SetControllerReference(app, service, r.Scheme)
	})

	return err
}

// ingress reconcile TODO cert still not tested yet
func (r *ApplicationReconciler) reconcileIngress(ctx context.Context, app *appv1alpha1.Application) error {
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-ingress", app.Name),
			Namespace: app.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		host := app.Spec.Host
		if app.Spec.Ingress != nil && app.Spec.Ingress.Host != "" {
			host = app.Spec.Ingress.Host
		}

		path := "/"
		if app.Spec.Ingress != nil && app.Spec.Ingress.Path != "" {
			path = app.Spec.Ingress.Path
		}

		pathType := networkingv1.PathTypePrefix
		servicePort := int32(80)
		if app.Spec.Service != nil && app.Spec.Service.Port != nil {
			servicePort = *app.Spec.Service.Port
		}

		// define ingress spec
		ingress.Spec = networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: host, // from spec if specified
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     path,
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: fmt.Sprintf("%s-service", app.Name), // related service
											Port: networkingv1.ServiceBackendPort{
												Number: servicePort,
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

		// Set default ingress class to nginx
		ingressClassName := "nginx"

		// Configure ingress class and additional settings if explicit ingress config exists
		if app.Spec.Ingress != nil && app.Spec.Ingress.IngressClassName != nil {
			ingressClassName = *app.Spec.Ingress.IngressClassName
		}
		ingress.Spec.IngressClassName = &ingressClassName

		// Configure additional settings if explicit ingress config exists
		if app.Spec.Ingress != nil {

			// Add TLS if specified
			if app.Spec.Ingress.TLSSecretName != "" {
				ingress.Spec.TLS = []networkingv1.IngressTLS{
					{
						Hosts:      []string{host},
						SecretName: app.Spec.Ingress.TLSSecretName,
					},
				}
			}

			// Add annotations if specified
			if app.Spec.Ingress.Annotations != nil {
				ingress.Annotations = app.Spec.Ingress.Annotations
			}
		}

		return controllerutil.SetControllerReference(app, ingress, r.Scheme)
	})

	return err
}

// HPA reconcile
func (r *ApplicationReconciler) reconcileHPA(ctx context.Context, app *appv1alpha1.Application) error {
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-hpa", app.Name),
			Namespace: app.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, hpa, func() error {
		minReplicas := int32(1)
		if app.Spec.HPA.MinReplicas != nil {
			minReplicas = *app.Spec.HPA.MinReplicas
		}

		var metrics []autoscalingv2.MetricSpec

		// Add CPU metric if specified
		if app.Spec.HPA.TargetCPUUtilizationPercentage != nil {
			metrics = append(metrics, autoscalingv2.MetricSpec{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: corev1.ResourceCPU,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: app.Spec.HPA.TargetCPUUtilizationPercentage,
					},
				},
			})
		}

		// Add Memory metric if specified
		if app.Spec.HPA.TargetMemoryUtilizationPercentage != nil {
			metrics = append(metrics, autoscalingv2.MetricSpec{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: corev1.ResourceMemory,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: app.Spec.HPA.TargetMemoryUtilizationPercentage,
					},
				},
			})
		}

		// Default to CPU 75% if no metrics specified
		if len(metrics) == 0 {
			cpuTarget := int32(75)
			metrics = append(metrics, autoscalingv2.MetricSpec{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: corev1.ResourceCPU,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: &cpuTarget,
					},
				},
			})
		}

		// Define HPA spec
		hpa.Spec = autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       fmt.Sprintf("%s-deployment", app.Name), // related deployment
			},
			MinReplicas: &minReplicas,
			MaxReplicas: app.Spec.HPA.MaxReplicas,
			Metrics:     metrics,
		}

		return controllerutil.SetControllerReference(app, hpa, r.Scheme)
	})

	return err
}

func (r *ApplicationReconciler) updateStatus(ctx context.Context, app *appv1alpha1.Application) error {
	// Get deployment status
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-deployment", app.Name),
		Namespace: app.Namespace,
	}, deployment)

	if err == nil {
		app.Status.Replicas = deployment.Status.Replicas
		app.Status.ReadyReplicas = deployment.Status.ReadyReplicas
		app.Status.DeploymentReady = deployment.Status.ReadyReplicas == deployment.Status.Replicas
	}

	// Set phase
	if app.Status.DeploymentReady {
		app.Status.Phase = "Running"
	} else {
		app.Status.Phase = "Pending"
	}

	// Check service
	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-service", app.Name),
		Namespace: app.Namespace,
	}, service)
	app.Status.ServiceReady = err == nil

	// Check ingress if configured
	if app.Spec.Host != "" || app.Spec.Ingress != nil {
		ingress := &networkingv1.Ingress{}
		err = r.Get(ctx, types.NamespacedName{
			Name:      fmt.Sprintf("%s-ingress", app.Name),
			Namespace: app.Namespace,
		}, ingress)
		app.Status.IngressReady = err == nil
	}

	// check if configure in application spec or we could set default here
	if app.Spec.HPA != nil {
		hpa := &autoscalingv2.HorizontalPodAutoscaler{}
		err = r.Get(ctx, types.NamespacedName{
			Name:      fmt.Sprintf("%s-hpa", app.Name),
			Namespace: app.Namespace,
		}, hpa)
		app.Status.HPAReady = err == nil
	}

	return r.Status().Update(ctx, app)
}

// SetupWithManager sets up the controller with the Manager
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.Application{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Named("application").
		Complete(r)
}
