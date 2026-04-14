/*
Copyright 2026 The Klutch Bind Authors.

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

package appclusterbinding

import (
	"context"
	"fmt"
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	dynamicinformer "k8s.io/client-go/dynamic/dynamicinformer"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	rbacinformers "k8s.io/client-go/informers/rbac/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	examplebackendv1alpha1 "github.com/anynines/klutchio/bind/contrib/example-backend/apis/examplebackend/v1alpha1"
	"github.com/anynines/klutchio/bind/contrib/example-backend/exporttemplate"
	bindv1alpha1 "github.com/anynines/klutchio/bind/pkg/apis/bind/v1alpha1"
	bindclient "github.com/anynines/klutchio/bind/pkg/client/clientset/versioned"
	bindinformers "github.com/anynines/klutchio/bind/pkg/client/informers/externalversions/bind/v1alpha1"
)

const (
	controllerName             = "klutch-bind-example-backend-appclusterbinding"
	appClusterBindingFinalizer = "klutch.anynines.com/appclusterbinding-cleanup"

	indexByKubeconfigSecret = "byKubeconfigSecret"
)

var appClusterBindingGVR = schema.GroupVersionResource{
	Group:    bindv1alpha1.GroupName,
	Version:  bindv1alpha1.GroupVersion,
	Resource: "appclusterbindings",
}

// NewController returns a new controller for AppClusterBindings.
func NewController(
	config *rest.Config,
	secretInformer coreinformers.SecretInformer,
	roleInformer rbacinformers.RoleInformer,
	roleBindingInformer rbacinformers.RoleBindingInformer,
	deploymentInformer appsinformers.DeploymentInformer,
	apiServiceBindingInformer bindinformers.APIServiceBindingInformer,
) (*Controller, error) {
	queue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), controllerName)

	logger := klog.Background().WithValues("controller", controllerName)

	config = rest.CopyConfig(config)
	config = rest.AddUserAgent(config, controllerName)

	bindClient, err := bindclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	templateIndex := exporttemplate.NewCatalogue(config)

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, 30*time.Minute)
	informer := factory.ForResource(appClusterBindingGVR).Informer()

	indexers := cache.Indexers{
		indexByKubeconfigSecret: indexAppClusterBindingByKubeconfigSecret,
	}
	if err := informer.AddIndexers(indexers); err != nil {
		return nil, err
	}

	c := &Controller{
		queue: queue,

		bindClient:                bindClient,
		dynamicClient:             dynamicClient,
		informerFactory:           factory,
		appClusterBindingInformer: informer,
		appClusterBindingIndexer:  informer.GetIndexer(),

		secretLister: secretInformer.Lister(),

		reconciler: reconciler{
			kubeClient: kubeClient,
			getSecret: func(ns, name string) (*corev1.Secret, error) {
				return secretInformer.Lister().Secrets(ns).Get(name)
			},
			listServiceBindings: func(ctx context.Context, labelSelector string) (*bindv1alpha1.APIServiceBindingList, error) {
				return bindClient.KlutchBindV1alpha1().APIServiceBindings("").List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
			},
			createServiceBinding: func(ctx context.Context, binding *bindv1alpha1.APIServiceBinding) (*bindv1alpha1.APIServiceBinding, error) {
				return bindClient.KlutchBindV1alpha1().APIServiceBindings(binding.Namespace).Create(ctx, binding, metav1.CreateOptions{})
			},
			updateServiceBinding: func(ctx context.Context, binding *bindv1alpha1.APIServiceBinding) (*bindv1alpha1.APIServiceBinding, error) {
				return bindClient.KlutchBindV1alpha1().APIServiceBindings(binding.Namespace).Update(ctx, binding, metav1.UpdateOptions{})
			},
			deleteServiceBinding: func(ctx context.Context, namespace, name string) error {
				return bindClient.KlutchBindV1alpha1().APIServiceBindings(namespace).Delete(ctx, name, metav1.DeleteOptions{})
			},
			templateFor: func(ctx context.Context, group, resource string) (examplebackendv1alpha1.APIServiceExportTemplate, error) {
				return templateIndex.TemplateFor(ctx, group, resource)
			},
			listAPIServiceExportRequests: func(ctx context.Context, namespace, labelSelector string) (*bindv1alpha1.APIServiceExportRequestList, error) {
				return bindClient.KlutchBindV1alpha1().APIServiceExportRequests(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
			},
			createAPIServiceExportRequest: func(ctx context.Context, namespace string, req *bindv1alpha1.APIServiceExportRequest) (*bindv1alpha1.APIServiceExportRequest, error) {
				return bindClient.KlutchBindV1alpha1().APIServiceExportRequests(namespace).Create(ctx, req, metav1.CreateOptions{})
			},
			deleteAPIServiceExportRequest: func(ctx context.Context, namespace, name string) error {
				return bindClient.KlutchBindV1alpha1().APIServiceExportRequests(namespace).Delete(ctx, name, metav1.DeleteOptions{})
			},
			getServiceAccount: func(ctx context.Context, namespace, name string) (*corev1.ServiceAccount, error) {
				return kubeClient.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
			},
			createServiceAccount: func(ctx context.Context, namespace string, sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
				return kubeClient.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{})
			},
			deleteServiceAccount: func(ctx context.Context, namespace, name string) error {
				return kubeClient.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{})
			},
			getRole: func(ctx context.Context, namespace, name string) (*rbacv1.Role, error) {
				return kubeClient.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
			},
			createRole: func(ctx context.Context, namespace string, role *rbacv1.Role) (*rbacv1.Role, error) {
				return kubeClient.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
			},
			updateRole: func(ctx context.Context, namespace string, role *rbacv1.Role) (*rbacv1.Role, error) {
				return kubeClient.RbacV1().Roles(namespace).Update(ctx, role, metav1.UpdateOptions{})
			},
			deleteRole: func(ctx context.Context, namespace, name string) error {
				return kubeClient.RbacV1().Roles(namespace).Delete(ctx, name, metav1.DeleteOptions{})
			},
			getClusterRole: func(ctx context.Context, name string) (*rbacv1.ClusterRole, error) {
				return kubeClient.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
			},
			createClusterRole: func(ctx context.Context, role *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
				return kubeClient.RbacV1().ClusterRoles().Create(ctx, role, metav1.CreateOptions{})
			},
			updateClusterRole: func(ctx context.Context, role *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
				return kubeClient.RbacV1().ClusterRoles().Update(ctx, role, metav1.UpdateOptions{})
			},
			deleteClusterRole: func(ctx context.Context, name string) error {
				return kubeClient.RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{})
			},
			getClusterRoleBinding: func(ctx context.Context, name string) (*rbacv1.ClusterRoleBinding, error) {
				return kubeClient.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
			},
			createClusterRoleBinding: func(ctx context.Context, workingClusterBinding *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
				return kubeClient.RbacV1().ClusterRoleBindings().Create(ctx, workingClusterBinding, metav1.CreateOptions{})
			},
			updateClusterRoleBinding: func(ctx context.Context, workingClusterBinding *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
				return kubeClient.RbacV1().ClusterRoleBindings().Update(ctx, workingClusterBinding, metav1.UpdateOptions{})
			},
			deleteClusterRoleBinding: func(ctx context.Context, name string) error {
				return kubeClient.RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{})
			},
			getRoleBinding: func(ctx context.Context, namespace, name string) (*rbacv1.RoleBinding, error) {
				return kubeClient.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
			},
			createRoleBinding: func(ctx context.Context, namespace string, rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
				return kubeClient.RbacV1().RoleBindings(namespace).Create(ctx, rb, metav1.CreateOptions{})
			},
			updateRoleBinding: func(ctx context.Context, namespace string, rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
				return kubeClient.RbacV1().RoleBindings(namespace).Update(ctx, rb, metav1.UpdateOptions{})
			},
			deleteRoleBinding: func(ctx context.Context, namespace, name string) error {
				return kubeClient.RbacV1().RoleBindings(namespace).Delete(ctx, name, metav1.DeleteOptions{})
			},
			getNamespace: func(ctx context.Context, name string) (*corev1.Namespace, error) {
				return kubeClient.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
			},
			createNamespace: func(ctx context.Context, namespace *corev1.Namespace) (*corev1.Namespace, error) {
				return kubeClient.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
			},
			deleteNamespace: func(ctx context.Context, name string) error {
				return kubeClient.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
			},
			getClusterBinding: func(ctx context.Context, namespace, name string) (*bindv1alpha1.ClusterBinding, error) {
				return bindClient.KlutchBindV1alpha1().ClusterBindings(namespace).Get(ctx, name, metav1.GetOptions{})
			},
			createClusterBinding: func(ctx context.Context, namespace string, binding *bindv1alpha1.ClusterBinding) (*bindv1alpha1.ClusterBinding, error) {
				return bindClient.KlutchBindV1alpha1().ClusterBindings(namespace).Create(ctx, binding, metav1.CreateOptions{})
			},
			updateClusterBinding: func(ctx context.Context, namespace string, binding *bindv1alpha1.ClusterBinding) (*bindv1alpha1.ClusterBinding, error) {
				return bindClient.KlutchBindV1alpha1().ClusterBindings(namespace).Update(ctx, binding, metav1.UpdateOptions{})
			},
			createSecret: func(ctx context.Context, namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
				return kubeClient.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
			},
			updateSecret: func(ctx context.Context, namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
				return kubeClient.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
			},
			getDeployment: func(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
				return kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			},
			createDeployment: func(ctx context.Context, namespace string, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
				return kubeClient.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
			},
			updateDeployment: func(ctx context.Context, namespace string, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
				return kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
			},
			deleteDeployment: func(ctx context.Context, namespace, name string) error {
				return kubeClient.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
			},
		},
	}

	if _, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueAppClusterBinding(logger, obj)
		},
		UpdateFunc: func(_, newObj interface{}) {
			c.enqueueAppClusterBinding(logger, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			c.enqueueAppClusterBinding(logger, obj)
		},
	}); err != nil {
		return nil, err
	}

	if _, err := secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueSecret(logger, obj)
		},
		UpdateFunc: func(_, newObj interface{}) {
			c.enqueueSecret(logger, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			c.enqueueSecret(logger, obj)
		},
	}); err != nil {
		return nil, err
	}

	// Watch Role and RoleBinding for drift detection
	if _, err := roleInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, newObj interface{}) {
			c.enqueueRBACResourceOwner(logger, newObj)
		},
	}); err != nil {
		return nil, err
	}

	if _, err := roleBindingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, newObj interface{}) {
			c.enqueueRBACResourceOwner(logger, newObj)
		},
	}); err != nil {
		return nil, err
	}

	// Watch Deployment for drift detection
	if _, err := deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, newObj interface{}) {
			c.enqueueRBACResourceOwner(logger, newObj)
		},
	}); err != nil {
		return nil, err
	}

	// Watch APIServiceBinding for drift detection
	if _, err := apiServiceBindingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueAPIServiceBinding(logger, obj)
		},
		UpdateFunc: func(_, newObj interface{}) {
			c.enqueueAPIServiceBinding(logger, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			c.enqueueAPIServiceBinding(logger, obj)
		},
	}); err != nil {
		return nil, err
	}

	return c, nil
}

// Controller reconciles AppClusterBindings.
type Controller struct {
	queue workqueue.RateLimitingInterface

	bindClient    bindclient.Interface
	dynamicClient dynamic.Interface

	informerFactory           dynamicinformer.DynamicSharedInformerFactory
	appClusterBindingInformer cache.SharedIndexInformer
	appClusterBindingIndexer  cache.Indexer

	secretLister corelisters.SecretLister

	reconciler
}

func (c *Controller) enqueueAppClusterBinding(logger klog.Logger, obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	logger.V(2).Info("queueing AppClusterBinding", "key", key)
	c.queue.Add(key)
}

func (c *Controller) enqueueSecret(logger klog.Logger, obj interface{}) {
	secretKey, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	bindings, err := c.appClusterBindingIndexer.ByIndex(indexByKubeconfigSecret, secretKey)
	if err != nil && !errors.IsNotFound(err) {
		utilruntime.HandleError(err)
		return
	} else if errors.IsNotFound(err) {
		return
	}

	for _, obj := range bindings {
		bindingKey, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			utilruntime.HandleError(err)
			continue
		}
		logger.V(2).Info("queueing AppClusterBinding", "key", bindingKey, "reason", "Secret", "SecretKey", secretKey)
		c.queue.Add(bindingKey)
	}
}

func (c *Controller) enqueueRBACResourceOwner(logger klog.Logger, obj interface{}) {
	meta, ok := obj.(metav1.Object)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("expected metav1.Object but got %T", obj))
		return
	}

	// Check OwnerReferences for AppClusterBinding
	for _, ref := range meta.GetOwnerReferences() {
		if ref.APIVersion == "klutch.anynines.com/v1alpha1" && ref.Kind == "AppClusterBinding" {
			key := meta.GetNamespace() + "/" + ref.Name
			logger.V(2).Info("queueing AppClusterBinding due to RBAC resource change", "key", key, "resource", meta.GetName())
			c.queue.Add(key)
			return
		}
	}
}

func (c *Controller) enqueueAPIServiceBinding(logger klog.Logger, obj interface{}) {
	meta, ok := obj.(metav1.Object)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("expected metav1.Object but got %T", obj))
		return
	}

	labels := meta.GetLabels()
	bindingName := labels[appClusterBindingNameLabel]
	bindingNamespace := labels[appClusterBindingNamespaceLabel]
	if bindingName == "" || bindingNamespace == "" {
		return
	}

	key := bindingNamespace + "/" + bindingName
	logger.V(2).Info("queueing AppClusterBinding due to APIServiceBinding change", "key", key, "binding", meta.GetName())
	c.queue.Add(key)
}

// Start starts the controller, which stops when ctx.Done() is closed.
func (c *Controller) Start(ctx context.Context, numThreads int) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	logger := klog.FromContext(ctx).WithValues("controller", controllerName)
	logger.Info("Starting controller")
	defer logger.Info("Shutting down controller")

	c.informerFactory.Start(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), c.appClusterBindingInformer.HasSynced)

	for i := 0; i < numThreads; i++ {
		go wait.UntilWithContext(ctx, c.startWorker, time.Second)
	}

	<-ctx.Done()
}

func (c *Controller) startWorker(ctx context.Context) {
	defer utilruntime.HandleCrash()

	for c.processNextWorkItem(ctx) {
	}
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	k, quit := c.queue.Get()
	if quit {
		return false
	}
	key := k.(string)

	logger := klog.FromContext(ctx).WithValues("key", key)
	ctx = klog.NewContext(ctx, logger)
	logger.V(2).Info("processing key")

	defer c.queue.Done(key)

	if err := c.process(ctx, key); err != nil {
		utilruntime.HandleError(fmt.Errorf("%q controller failed to sync %q, err: %w", controllerName, key, err))
		c.queue.AddRateLimited(key)
		return true
	}
	c.queue.Forget(key)
	return true
}

func (c *Controller) process(ctx context.Context, key string) error {
	obj, exists, err := c.appClusterBindingIndexer.GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("unexpected object type %T", obj)
	}

	binding := &bindv1alpha1.AppClusterBinding{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, binding); err != nil {
		return err
	}

	if binding.DeletionTimestamp.IsZero() {
		added, err := c.ensureFinalizer(ctx, binding)
		if err != nil {
			return err
		}
		if added {
			return nil
		}
	} else if hasFinalizer(binding.Finalizers, appClusterBindingFinalizer) {
		if err := c.ensureDeleted(ctx, binding); err != nil {
			return err
		}

		removed, err := c.removeFinalizer(ctx, binding)
		if err != nil {
			return err
		}
		if removed {
			return nil
		}
	}

	oldStatus := binding.Status.DeepCopy()

	if binding.DeletionTimestamp.IsZero() {
		if err := c.reconcile(ctx, binding); err != nil {
			return err
		}
	}

	if oldStatus != nil && reflect.DeepEqual(*oldStatus, binding.Status) {
		return nil
	}

	statusData, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&binding.Status)
	if err != nil {
		return err
	}

	updated := u.DeepCopy()
	if err := unstructured.SetNestedField(updated.Object, statusData, "status"); err != nil {
		return err
	}

	_, err = c.dynamicClient.Resource(appClusterBindingGVR).Namespace(binding.Namespace).UpdateStatus(ctx, updated, metav1.UpdateOptions{})
	return err
}

func (c *Controller) ensureFinalizer(ctx context.Context, binding *bindv1alpha1.AppClusterBinding) (bool, error) {
	if hasFinalizer(binding.Finalizers, appClusterBindingFinalizer) {
		return false, nil
	}

	updated := binding.DeepCopy()
	updated.Finalizers = append(updated.Finalizers, appClusterBindingFinalizer)
	updatedData, err := runtime.DefaultUnstructuredConverter.ToUnstructured(updated)
	if err != nil {
		return false, err
	}

	if _, err := c.dynamicClient.Resource(appClusterBindingGVR).Namespace(binding.Namespace).Update(ctx, &unstructured.Unstructured{Object: updatedData}, metav1.UpdateOptions{}); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Controller) removeFinalizer(ctx context.Context, binding *bindv1alpha1.AppClusterBinding) (bool, error) {
	if !hasFinalizer(binding.Finalizers, appClusterBindingFinalizer) {
		return false, nil
	}

	updated := binding.DeepCopy()
	updated.Finalizers = removeFinalizer(updated.Finalizers, appClusterBindingFinalizer)
	updatedData, err := runtime.DefaultUnstructuredConverter.ToUnstructured(updated)
	if err != nil {
		return false, err
	}

	if _, err := c.dynamicClient.Resource(appClusterBindingGVR).Namespace(binding.Namespace).Update(ctx, &unstructured.Unstructured{Object: updatedData}, metav1.UpdateOptions{}); err != nil {
		return false, err
	}

	return true, nil
}

func hasFinalizer(finalizers []string, target string) bool {
	for _, finalizer := range finalizers {
		if finalizer == target {
			return true
		}
	}

	return false
}

func removeFinalizer(finalizers []string, target string) []string {
	if len(finalizers) == 0 {
		return finalizers
	}

	updated := make([]string, 0, len(finalizers))
	for _, finalizer := range finalizers {
		if finalizer == target {
			continue
		}
		updated = append(updated, finalizer)
	}

	return updated
}

func indexAppClusterBindingByKubeconfigSecret(obj interface{}) ([]string, error) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, nil
	}

	ns, _, err := unstructured.NestedString(u.Object, "spec", "kubeconfigSecretRef", "namespace")
	if err != nil {
		return nil, err
	}
	name, _, err := unstructured.NestedString(u.Object, "spec", "kubeconfigSecretRef", "name")
	if err != nil {
		return nil, err
	}
	if ns == "" || name == "" {
		return nil, nil
	}
	return []string{ns + "/" + name}, nil
}
