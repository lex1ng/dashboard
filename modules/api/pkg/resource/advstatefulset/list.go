package advstatefulset

import (
	kruise "github.com/openkruise/kruise-api/apps/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/dashboard/api/pkg/resource/event"
	"k8s.io/dashboard/errors"
	"k8s.io/klog/v2"

	kruiseclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	metricapi "k8s.io/dashboard/api/pkg/integration/metric/api"
	"k8s.io/dashboard/api/pkg/resource/common"
	"k8s.io/dashboard/api/pkg/resource/dataselect"
	"k8s.io/dashboard/types"
)

type AdvStatefulSetList struct {
	ListMeta types.ListMeta `json:"listMeta"`

	Status            common.ResourceStatus `json:"status"`
	AdvStatefulSets   []AdvStatefulSet      `json:"advStatefulSets"`
	CumulativeMetrics []metricapi.Metric    `json:"cumulativeMetrics"`

	Errors []error `json:"errors"`
}

type AdvStatefulSet struct {
	ObjectMeta          types.ObjectMeta `json:"objectMeta"`
	TypeMeta            types.TypeMeta   `json:"typeMeta"`
	Pods                common.PodInfo   `json:"podInfo"`
	ContainerImages     []string         `json:"containerImages"`
	InitContainerImages []string         `json:"initContainerImages"`
}

func GetAdvStatefulSetList(kruiseClient kruiseclientset.Interface, client kubernetes.Interface, nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery, metricClient metricapi.MetricClient) (*AdvStatefulSetList, error) {
	klog.V(4).Infof("Getting list of all stateful sets in the cluster")

	channels := &common.ResourceChannels{
		AdvStatefulSetList: common.GetAdvStatefulSetListChannel(kruiseClient, nsQuery, 1),
		PodList:            common.GetPodListChannel(client, nsQuery, 1),
		EventList:          common.GetEventListChannel(client, nsQuery, 1),
	}

	return GetAdvStatefulSetListFromChannels(channels, dsQuery, metricClient)
}

func GetAdvStatefulSetListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery,
	metricClient metricapi.MetricClient) (*AdvStatefulSetList, error) {

	statefulSets := <-channels.AdvStatefulSetList.List
	err := <-channels.AdvStatefulSetList.Error
	nonCriticalErrors, criticalError := errors.ExtractErrors(err)
	if criticalError != nil {
		return nil, criticalError
	}

	pods := <-channels.PodList.List
	err = <-channels.PodList.Error
	nonCriticalErrors, criticalError = errors.AppendError(err, nonCriticalErrors)
	if criticalError != nil {
		return nil, criticalError
	}

	events := <-channels.EventList.List
	err = <-channels.EventList.Error
	nonCriticalErrors, criticalError = errors.AppendError(err, nonCriticalErrors)
	if criticalError != nil {
		return nil, criticalError
	}

	ssList := toAdvStatefulSetList(statefulSets.Items, pods.Items, events.Items, nonCriticalErrors, dsQuery, metricClient)
	ssList.Status = getStatus(statefulSets, pods.Items, events.Items)
	return ssList, nil
}

func toAdvStatefulSetList(statefulSets []kruise.StatefulSet, pods []v1.Pod, events []v1.Event, nonCriticalErrors []error,
	dsQuery *dataselect.DataSelectQuery, metricClient metricapi.MetricClient) *AdvStatefulSetList {

	statefulSetList := &AdvStatefulSetList{
		AdvStatefulSets: make([]AdvStatefulSet, 0),
		ListMeta:        types.ListMeta{TotalItems: len(statefulSets)},
		Errors:          nonCriticalErrors,
	}

	cachedResources := &metricapi.CachedResources{
		Pods: pods,
	}
	ssCells, metricPromises, filteredTotal := dataselect.GenericDataSelectWithFilterAndMetrics(
		toCells(statefulSets), dsQuery, cachedResources, metricClient)
	statefulSets = fromCells(ssCells)
	statefulSetList.ListMeta = types.ListMeta{TotalItems: filteredTotal}

	for _, statefulSet := range statefulSets {
		matchingPods := common.FilterPodsByControllerRef(&statefulSet, pods)
		podInfo := common.GetPodInfo(statefulSet.Status.Replicas, statefulSet.Spec.Replicas, matchingPods)
		podInfo.Warnings = event.GetPodsEventWarnings(events, matchingPods)
		statefulSetList.AdvStatefulSets = append(statefulSetList.AdvStatefulSets, toAdvStatefulSet(&statefulSet, &podInfo))
	}

	cumulativeMetrics, err := metricPromises.GetMetrics()
	statefulSetList.CumulativeMetrics = cumulativeMetrics
	if err != nil {
		statefulSetList.CumulativeMetrics = make([]metricapi.Metric, 0)
	}

	return statefulSetList
}

func toAdvStatefulSet(statefulSet *kruise.StatefulSet, podInfo *common.PodInfo) AdvStatefulSet {
	return AdvStatefulSet{
		ObjectMeta:          types.NewObjectMeta(statefulSet.ObjectMeta),
		TypeMeta:            types.NewTypeMeta(types.ResourceKindAdvStatefulSet),
		ContainerImages:     common.GetContainerImages(&statefulSet.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&statefulSet.Spec.Template.Spec),
		Pods:                *podInfo,
	}
}
