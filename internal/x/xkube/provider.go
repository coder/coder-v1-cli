package xkube

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/xerrors"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// load all auth plugins.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

const (
	// ResourceName is the name of the coder resources.
	ResourceName        = "coder"
	WorkspaceSA         = "coder-workspaces"
	saNameAnnotationKey = "kubernetes.io/service-account.name"
	saTokenKey          = "token"
	saCertKey           = "ca.crt"
)

// KubeContext is the data about a kubernetes context.
type KubeContext struct {
	ContextName    string
	Namespace      string
	ClusterAddress string
	Clientset      *kubernetes.Clientset
}

// CurrentKubeContext returns the details of the current kubernetes context.
func CurrentKubeContext() (*KubeContext, error) {
	clientCfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return nil, xerrors.Errorf("loading kubernetes client config: %w", err)
	}
	currentContext := clientCfg.Contexts[clientCfg.CurrentContext]
	cluster := clientCfg.Clusters[currentContext.Cluster]

	restConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		nil,
	).ClientConfig()
	if err != nil {
		return nil, xerrors.Errorf("loading kubernetes rest config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, xerrors.Errorf("loading kubernetes clientset: %w", err)
	}
	return &KubeContext{
		ContextName:    clientCfg.CurrentContext,
		Namespace:      currentContext.Namespace,
		ClusterAddress: cluster.Server,
		Clientset:      clientset,
	}, nil
}

type ServiceAccount struct {
	SAToken   string
	ClusterCA string
}

// CoderServiceAccountFromContext reads the current kube context and fetches the coder workspace provider service account.
func CoderServiceAccountFromContext(ctx context.Context) (*ServiceAccount, error) {
	kctx, err := CurrentKubeContext()
	if err != nil {
		return nil, xerrors.Errorf("getting current kube context: %w", err)
	}

	saSecrets, err := kctx.Clientset.CoreV1().Secrets(kctx.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, xerrors.Errorf("listing kubernetes secrets: %w", err)
	}

	for _, sa := range saSecrets.Items {
		if sa.Annotations[saNameAnnotationKey] == ResourceName {
			return &ServiceAccount{
				SAToken:   string(sa.Data[saTokenKey]),
				ClusterCA: string(sa.Data[saCertKey]),
			}, nil
		}
	}

	return nil, xerrors.Errorf("kubernetes service account secret with annotaion %s=%s not found", saNameAnnotationKey, ResourceName)
}

// InstallWorkspaceProviderResources creates the resources for a workspace provider.
func InstallWorkspaceProviderResources(ctx context.Context, clientset *kubernetes.Clientset, namespace string) error {
	err := installServiceAccount(ctx, clientset, namespace)
	if err != nil {
		return xerrors.Errorf("installing kubernetes service account: %w", err)
	}

	err = installWorkspaceServiceAccount(ctx, clientset, namespace)
	if err != nil {
		return xerrors.Errorf("installing workspace kubernetes service account: %w", err)
	}

	err = installRole(ctx, clientset, namespace)
	if err != nil {
		return xerrors.Errorf("installing kubernetes role: %w", err)
	}

	err = installRoleBinding(ctx, clientset, namespace)
	if err != nil {
		return xerrors.Errorf("installing kubernetes role binding: %w", err)
	}

	return nil
}

// PrettyRules returns a pretty format string of the coder role rules.
func PrettyRules() string {
	var output string
	for _, rule := range roleSpec().Rules {
		// core api group is represented as "" so we handle that formatting
		for i, group := range rule.APIGroups {
			if group == "" {
				rule.APIGroups[i] = `""`
			}
		}
		groups := strings.Join(rule.APIGroups, ",")
		resources := strings.Join(rule.Resources, ",")
		verbs := strings.Join(rule.Verbs, ",")
		output = fmt.Sprintf("%s  - APIGroups: %s\n    Resources: %s\n    Verbs: %s\n", output, groups, resources, verbs)
	}
	return output
}

func installServiceAccount(ctx context.Context, clientset *kubernetes.Clientset, namespace string) error {
	_, err := clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, serviceAccountSpec(), metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			_, err = clientset.CoreV1().ServiceAccounts(namespace).Update(ctx, serviceAccountSpec(), metav1.UpdateOptions{})
			if err != nil {
				return xerrors.Errorf("updating service account: %w", err)
			}
			return nil
		}
		return xerrors.Errorf("creating service account: %w", err)
	}

	return nil
}

func installWorkspaceServiceAccount(ctx context.Context, clientset *kubernetes.Clientset, namespace string) error {
	spec := serviceAccountSpec()
	spec.Name = WorkspaceSA
	_, err := clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			_, err = clientset.CoreV1().ServiceAccounts(namespace).Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return xerrors.Errorf("updating service account: %w", err)
			}
			return nil
		}
		return xerrors.Errorf("creating service account: %w", err)
	}

	return nil
}

func serviceAccountSpec() *v1.ServiceAccount {
	return &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: ResourceName,
		},
	}
}

func installRole(ctx context.Context, clientset *kubernetes.Clientset, namespace string) error {
	_, err := clientset.RbacV1().Roles(namespace).Create(ctx, roleSpec(), metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			_, err = clientset.RbacV1().Roles(namespace).Update(ctx, roleSpec(), metav1.UpdateOptions{})
			if err != nil {
				return xerrors.Errorf("updating role: %w", err)
			}
			return nil
		}
		return xerrors.Errorf("creating role: %w", err)
	}

	return nil
}

func roleSpec() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: ResourceName,
		},
		Rules: []rbacv1.PolicyRule{
			// RW
			{
				APIGroups: []string{"", "networking.k8s.io"},
				Resources: []string{"persistentvolumeclaims", "pods", "secrets", "pods/exec", "pods/log", "events", "networkpolicies"},
				Verbs:     []string{"create", "get", "list", "watch", "update", "patch", "delete", "deletecollection"},
			},
			// RO
			{
				APIGroups: []string{"metrics.k8s.io", "storage.k8s.io"},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

func installRoleBinding(ctx context.Context, clientset *kubernetes.Clientset, namespace string) error {
	_, err := clientset.RbacV1().RoleBindings(namespace).Create(ctx, roleBindingSpec(namespace), metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			_, err = clientset.RbacV1().RoleBindings(namespace).Update(ctx, roleBindingSpec(namespace), metav1.UpdateOptions{})
			if err != nil {
				return xerrors.Errorf("updating role binding: %w", err)
			}
			return nil
		}
		return xerrors.Errorf("creating role binding: %w", err)
	}

	return nil
}

func roleBindingSpec(namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: ResourceName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      ResourceName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     ResourceName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}
