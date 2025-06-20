package cloneset

import (
	"context"

	kruise "github.com/openkruise/kruise-api/apps/v1alpha1"
	kruiseclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricapi "k8s.io/dashboard/api/pkg/integration/metric/api"
	"k8s.io/dashboard/api/pkg/resource/common"
	"k8s.io/dashboard/api/pkg/resource/dataselect"
	"k8s.io/dashboard/api/pkg/resource/event"
	"k8s.io/dashboard/api/pkg/resource/pod"
	"k8s.io/dashboard/errors"
	"k8s.io/klog/v2"
)

func GetCloneSetPods(kruiseClient kruiseclientset.Interface, client kubernetes.Interface, metricClient metricapi.MetricClient,
	dsQuery *dataselect.DataSelectQuery, name, namespace string) (*pod.PodList, error) {

	klog.V(4).Infof("Getting replication controller %s pods in namespace %s", name, namespace)

	pods, err := getRawCloneSetPods(kruiseClient, client, name, namespace)
	if err != nil {
		return pod.EmptyPodList, err
	}

	events, err := event.GetPodsEvents(client, namespace, pods)
	nonCriticalErrors, criticalError := errors.ExtractErrors(err)
	if criticalError != nil {
		return nil, criticalError
	}

	podList := pod.ToPodList(pods, events, nonCriticalErrors, dsQuery, metricClient)
	return &podList, nil
}
func getRawCloneSetPods(kruiseClient kruiseclientset.Interface, client kubernetes.Interface, name, namespace string) ([]v1.Pod, error) {
	cloneSet, err := kruiseClient.AppsV1alpha1().CloneSets(namespace).Get(context.TODO(), name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	channels := &common.ResourceChannels{
		PodList: common.GetPodListChannel(client, common.NewSameNamespaceQuery(namespace), 1),
	}

	podList := <-channels.PodList.List
	if err := <-channels.PodList.Error; err != nil {
		return nil, err
	}

	return common.FilterPodsByControllerRef(cloneSet, podList.Items), nil
}

func getCloneSetPodInfo(kruiseClient kruiseclientset.Interface, client kubernetes.Interface, cloneSet *kruise.CloneSet) (*common.PodInfo, error) {
	pods, err := getRawCloneSetPods(kruiseClient, client, cloneSet.Name, cloneSet.Namespace)
	if err != nil {
		return nil, err
	}

	podInfo := common.GetPodInfo(cloneSet.Status.Replicas, cloneSet.Spec.Replicas, pods)
	return &podInfo, nil
}
