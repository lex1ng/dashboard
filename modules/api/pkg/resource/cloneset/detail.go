package cloneset

import (
	"context"
	kruise "github.com/openkruise/kruise-api/apps/v1alpha1"
	kruiseclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricapi "k8s.io/dashboard/api/pkg/integration/metric/api"
	"k8s.io/dashboard/api/pkg/resource/common"
	"k8s.io/dashboard/errors"
	"k8s.io/klog/v2"
)

type CloneSetDetail struct {
	CloneSet `json:",inline"`

	Errors []error `json:"errors"`
}

func GetCloneSetDetail(kruiseClient kruiseclientset.Interface, client kubernetes.Interface, metricClient metricapi.MetricClient, namespace, name string) (*CloneSetDetail, error) {
	klog.V(4).Infof("Getting details of %s cloneset in %s namespace", name, namespace)

	ss, err := kruiseClient.AppsV1alpha1().CloneSets(namespace).Get(context.TODO(), name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	podInfo, err := getCloneSetPodInfo(kruiseClient, client, ss)
	nonCriticalErrors, criticalError := errors.ExtractErrors(err)
	if criticalError != nil {
		return nil, criticalError
	}

	ssDetail := getCloneSetDetail(ss, podInfo, nonCriticalErrors)
	return &ssDetail, nil
}

func getCloneSetDetail(cloneSet *kruise.CloneSet, podInfo *common.PodInfo, nonCriticalErrors []error) CloneSetDetail {
	return CloneSetDetail{
		CloneSet: toCloneSet(cloneSet, podInfo),
		Errors:   nonCriticalErrors,
	}
}
