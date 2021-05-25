package xkube

import (
	"cdr.dev/coder-cli/coder-sdk"
	"context"
	"golang.org/x/xerrors"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	// load all auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

const (
	saNameAnnotationKey = "kubernetes.io/service-account.name"
	saName = "coder"
	saTokenKey = "token"
	saCertKey  = "ca.crt"
)

func ApplyKubeConfigFromContext(ctx context.Context, req *coder.WorkspaceProviderKubernetesCreateRequest) (*coder.WorkspaceProviderKubernetesCreateRequest, error) {
	clientCfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return nil, err
	}
	currentContext := clientCfg.Contexts[clientCfg.CurrentContext]
	cluster := clientCfg.Clusters[currentContext.Cluster]

	req.DefaultNamespace = currentContext.Namespace
	req.ClusterAddress = cluster.Server

	restConfig, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	saSecrets, err := clientset.CoreV1().Secrets(req.DefaultNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, sa := range saSecrets.Items {
		if sa.Annotations[saNameAnnotationKey] == saName {
			req.SAToken = string(sa.Data[saTokenKey])
			req.ClusterCA = string(sa.Data[saCertKey])

			return req, nil
		}
	}

	return nil, xerrors.Errorf("kubernetes service account secret with annotaion %s=%s not found", saNameAnnotationKey, saName)
}

func CreateWorkspaceProviderResources(ctx context.Context) error {
	clientCfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return err
	}
	namespace := clientCfg.Contexts[clientCfg.CurrentContext].Namespace

	restConfig, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: saName,
		},
	}
	sa, err = clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	role := &rbacv1.Role{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Rules:      nil,
	}
	role, err = clientset.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	rb := &rbacv1.RoleBinding{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Subjects:   nil,
		RoleRef:    rbacv1.RoleRef{},
	}
	rb, err = clientset.RbacV1().RoleBindings(namespace).Create(ctx, rb, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}