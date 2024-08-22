package pkg

import (
	"context"
	"time"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"k8s.io/metrics/pkg/apis/custom_metrics"

	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider/helpers"
)

type CustomMetricsProvider interface {
	  provider.ExternalMetricsProvider

    ListAllMetrics() []provider.CustomMetricInfo

    GetMetricByName(ctx context.Context, name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error)
    GetMetricBySelector(ctx context.Context, namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error)
}


type clusterProvider struct {
    client dynamic.Interface
		clientset *kubernetes.Clientset
    mapper apimeta.RESTMapper

    // just increment values when they're requested
    values map[provider.CustomMetricInfo]int64
}

func NewProvider(client dynamic.Interface, mapper apimeta.RESTMapper, clientset *kubernetes.Clientset) CustomMetricsProvider {
	return &clusterProvider{
		client: client,
		mapper: mapper,
		clientset: clientset,
		values: make(map[provider.CustomMetricInfo]int64),
	}
}


func (p *clusterProvider) ListAllMetrics() []provider.CustomMetricInfo {
	return []provider.CustomMetricInfo{
		{
			Metric: "controlplanes",
			Namespaced: false,
		},
		{
			Metric: "workers",
			Namespaced: false,
		},
	}
}

func (p *clusterProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	return []provider.ExternalMetricInfo{
		{
			Metric: "controlplanes",
		},
		{
			Metric: "workers",
		},
	}
}

func (p *clusterProvider)	GetExternalMetric(ctx context.Context, 
namespace string,
metricSelector labels.Selector,
info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
    value, err := p.valueForNodes(info)
    if err != nil {
        return nil, err
    }
		var list external_metrics.ExternalMetricValueList
		item := external_metrics.ExternalMetricValue{
				MetricName: info.Metric,
        Timestamp:       metav1.Time{time.Now()},
        Value:           *resource.NewQuantity(value, resource.DecimalSI),
    }
		list.Items = append(list.Items, item)
		return &list, nil
}

// valueFor fetches a value from the fake list and increments it.
func (p *clusterProvider) valueFor(info provider.CustomMetricInfo) (int64, error) {
	nodeList, err := p.clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	klog.InfoS("get value for")
	if err != nil {
		return 0, err
	}
    return int64(len(nodeList.Items)), nil
}

func (p *clusterProvider) valueForNodes(info provider.ExternalMetricInfo) (int64, error) {
	var opt metav1.ListOptions
		nodeList, err := p.clientset.CoreV1().Nodes().List(context.TODO(), opt)
		if err != nil {
			return 0, err
		}
	var count int64
		for _, node := range nodeList.Items {
			_, ok := node.Labels["node-role.kubernetes.io/control-plane"]
			if (ok && info.Metric == "controlplanes") || (
				!ok && info.Metric != "controlplanes"){
				count++
			}
		}
    return int64(count), nil
}

// metricFor constructs a result for a single metric value.
func (p *clusterProvider) metricFor(value int64, name types.NamespacedName, info provider.CustomMetricInfo) (*custom_metrics.MetricValue, error) {
    // construct a reference referring to the described object
    objRef, err := helpers.ReferenceFor(p.mapper, name, info)
    if err != nil {
        return nil, err
    }

    return &custom_metrics.MetricValue{
        DescribedObject: objRef,
        Metric: custom_metrics.MetricIdentifier{
                Name:  info.Metric,
        },
        // you'll want to use the actual timestamp in a real adapter
        Timestamp:       metav1.Time{time.Now()},
        Value:           *resource.NewMilliQuantity(value*100, resource.DecimalSI),
    }, nil
}


func (p *clusterProvider) GetMetricByName(ctx context.Context, name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {
    value, err := p.valueFor(info)
    if err != nil {
        return nil, err
    }
    return p.metricFor(value, name, info)
}



func (p *clusterProvider) GetMetricBySelector(ctx context.Context, namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
    totalValue, err := p.valueFor(info)
    if err != nil {
        return nil, err
    }

    names, err := helpers.ListObjectNames(p.mapper, p.client, namespace, selector, info)
    if err != nil {
        return nil, err
    }

    res := make([]custom_metrics.MetricValue, len(names))
    for i, name := range names {
        value, err := p.metricFor(100*totalValue/int64(len(res)), types.NamespacedName{Namespace: namespace, Name: name}, info)
        if err != nil {
            return nil, err
        }
        res[i] = *value
    }

    return &custom_metrics.MetricValueList{
        Items: res,
    }, nil
}
