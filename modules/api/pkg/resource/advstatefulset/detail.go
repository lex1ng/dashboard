package advstatefulset

import (
	"context"
	kruise "github.com/openkruise/kruise-api/apps/v1beta1"
	kruiseclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricapi "k8s.io/dashboard/api/pkg/integration/metric/api"
	"k8s.io/dashboard/api/pkg/resource/common"
	"k8s.io/dashboard/errors"
	"k8s.io/klog/v2"
)

type AdvStatefulSetDetail struct {
	AdvStatefulSet `json:",inline"`

	Errors []error `json:"errors"`
}

func GetAdvStatefulSetDetail(kruiseClient kruiseclientset.Interface, client kubernetes.Interface, metricClient metricapi.MetricClient, namespace, name string) (*AdvStatefulSetDetail, error) {
	klog.V(4).Infof("Getting details of %s statefulset in %s namespace", name, namespace)

	ss, err := kruiseClient.AppsV1beta1().StatefulSets(namespace).Get(context.TODO(), name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	podInfo, err := getAdvStatefulSetPodInfo(kruiseClient, client, ss)
	nonCriticalErrors, criticalError := errors.ExtractErrors(err)
	if criticalError != nil {
		return nil, criticalError
	}

	ssDetail := getAdvStatefulSetDetail(ss, podInfo, nonCriticalErrors)
	return &ssDetail, nil
}

func getAdvStatefulSetDetail(statefulSet *kruise.StatefulSet, podInfo *common.PodInfo, nonCriticalErrors []error) AdvStatefulSetDetail {
	return AdvStatefulSetDetail{
		AdvStatefulSet: toAdvStatefulSet(statefulSet, podInfo),
		Errors:         nonCriticalErrors,
	}
}
