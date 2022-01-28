package mutation

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=sidecar.infinispan.org,admissionReviewVersions=v1,sideEffects=None

// CacheInjector adds a cache side-car to pods with sidecar.infinispan.org/inject: "true"
type CacheInjector struct {
	Client  client.Client
	decoder *admission.Decoder
}

// podAnnotator adds an annotation to every incoming pods.
func (a *CacheInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	injectAnnotation, ok := pod.Annotations["sidecar.infinispan.org/inject"]
	if ok {
		inject, err := strconv.ParseBool(injectAnnotation)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}

		if inject {
			addSideCarContainer(pod)
		}
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// podAnnotator implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (a *CacheInjector) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}

func addSideCarContainer(pod *corev1.Pod) {
	container := corev1.Container{
		Name:  "caching-sidecar",
		Image: "quay.io/infinispan/server",
	}
	for i, c := range pod.Spec.Containers {
		if c.Name == container.Name {
			pod.Spec.Containers[i] = container
			return
		}
	}
	pod.Spec.Containers = append(pod.Spec.Containers, container)
}
