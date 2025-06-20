package cloneset

import (
	kruise "github.com/openkruise/kruise-api/apps/v1alpha1"
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

type CloneSetList struct {
	ListMeta types.ListMeta `json:"listMeta"`

	Status            common.ResourceStatus `json:"status"`
	CloneSets         []CloneSet            `json:"cloneSets"`
	CumulativeMetrics []metricapi.Metric    `json:"cumulativeMetrics"`

	Errors []error `json:"errors"`
}

type CloneSet struct {
	ObjectMeta          types.ObjectMeta `json:"objectMeta"`
	TypeMeta            types.TypeMeta   `json:"typeMeta"`
	Pods                common.PodInfo   `json:"podInfo"`
	ContainerImages     []string         `json:"containerImages"`
	InitContainerImages []string         `json:"initContainerImages"`
}

func GetCloneSetList(kruiseClient kruiseclientset.Interface, client kubernetes.Interface, nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery, metricClient metricapi.MetricClient) (*CloneSetList, error) {
	klog.V(4).Infof("Getting list of all clonesets in the cluster")

	channels := &common.ResourceChannels{
		CloneSetList: common.GetCloneSetListChannel(kruiseClient, nsQuery, 1),
		PodList:      common.GetPodListChannel(client, nsQuery, 1),
		EventList:    common.GetEventListChannel(client, nsQuery, 1),
	}

	return GetCloneSetListFromChannels(channels, dsQuery, metricClient)
}

func GetCloneSetListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery,
	metricClient metricapi.MetricClient) (*CloneSetList, error) {

	cloneSets := <-channels.CloneSetList.List
	err := <-channels.CloneSetList.Error
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

	ssList := toCloneSetList(cloneSets.Items, pods.Items, events.Items, nonCriticalErrors, dsQuery, metricClient)
	ssList.Status = getStatus(cloneSets, pods.Items, events.Items)
	return ssList, nil
}

func toCloneSetList(cloneSets []kruise.CloneSet, pods []v1.Pod, events []v1.Event, nonCriticalErrors []error,
	dsQuery *dataselect.DataSelectQuery, metricClient metricapi.MetricClient) *CloneSetList {

	cloneSetList := &CloneSetList{
		CloneSets: make([]CloneSet, 0),
		ListMeta:  types.ListMeta{TotalItems: len(cloneSets)},
		Errors:    nonCriticalErrors,
	}

	cachedResources := &metricapi.CachedResources{
		Pods: pods,
	}
	ssCells, metricPromises, filteredTotal := dataselect.GenericDataSelectWithFilterAndMetrics(
		toCells(cloneSets), dsQuery, cachedResources, metricClient)
	cloneSets = fromCells(ssCells)
	cloneSetList.ListMeta = types.ListMeta{TotalItems: filteredTotal}

	for _, cloneSet := range cloneSets {
		matchingPods := common.FilterPodsByControllerRef(&cloneSet, pods)
		podInfo := common.GetPodInfo(cloneSet.Status.Replicas, cloneSet.Spec.Replicas, matchingPods)
		podInfo.Warnings = event.GetPodsEventWarnings(events, matchingPods)
		cloneSetList.CloneSets = append(cloneSetList.CloneSets, toCloneSet(&cloneSet, &podInfo))
	}

	cumulativeMetrics, err := metricPromises.GetMetrics()
	cloneSetList.CumulativeMetrics = cumulativeMetrics
	if err != nil {
		cloneSetList.CumulativeMetrics = make([]metricapi.Metric, 0)
	}

	return cloneSetList
}

func toCloneSet(cloneSet *kruise.CloneSet, podInfo *common.PodInfo) CloneSet {
	return CloneSet{
		ObjectMeta:          types.NewObjectMeta(cloneSet.ObjectMeta),
		TypeMeta:            types.NewTypeMeta(types.ResourceKindCloneSet),
		ContainerImages:     common.GetContainerImages(&cloneSet.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&cloneSet.Spec.Template.Spec),
		Pods:                *podInfo,
	}
}
